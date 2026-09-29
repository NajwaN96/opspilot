package kubernetes

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/opspilot/opspilot/apps/api/internal/release"
)

// EnsureCanary creates or updates only demo-shop/payment-api-canary for a known candidate image.
func (c *Client) EnsureCanary(ctx context.Context, image, version string) error {
	if !release.Candidate(image, version) {
		return fmt.Errorf("%w: image/version is not a known canary", release.ErrDenied)
	}
	dep, err := c.clientset.AppsV1().Deployments(release.Namespace).Get(ctx, release.CanaryDeployment, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		created := canaryDeployment(image, version)
		_, err = c.clientset.AppsV1().Deployments(release.Namespace).Create(ctx, created, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	if dep.Namespace != release.Namespace || dep.Name != release.CanaryDeployment {
		return fmt.Errorf("%w: refusing %s/%s", release.ErrDenied, dep.Namespace, dep.Name)
	}
	container, err := namedContainer(dep, release.Container)
	if err != nil {
		return err
	}
	container.Image = image
	setEnv(container, "SERVICE_NAME", release.Deployment)
	setEnv(container, "SERVICE_VERSION", version)
	setEnv(container, "RELEASE_ROLE", "canary")
	replicas := int32(1)
	dep.Spec.Replicas = &replicas
	if dep.Labels == nil {
		dep.Labels = map[string]string{}
	}
	dep.Labels["app.kubernetes.io/version"] = version
	dep.Spec.Template.Labels["app.kubernetes.io/version"] = version
	_, err = c.clientset.AppsV1().Deployments(release.Namespace).Update(ctx, dep, metav1.UpdateOptions{})
	return err
}

// CanaryStatus reports the candidate Deployment. Missing means not ready.
func (c *Client) CanaryStatus(ctx context.Context) (ready bool, desired, readyCount int, image, version string, err error) {
	dep, err := c.clientset.AppsV1().Deployments(release.Namespace).Get(ctx, release.CanaryDeployment, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return false, 0, 0, "", "", nil
	}
	if err != nil {
		return false, 0, 0, "", "", err
	}
	container, err := namedContainer(dep, release.Container)
	if err != nil {
		return false, 0, 0, "", "", err
	}
	want := int32(0)
	if dep.Spec.Replicas != nil {
		want = *dep.Spec.Replicas
	}
	version = dep.Spec.Template.Labels["app.kubernetes.io/version"]
	ready = want > 0 && dep.Status.ReadyReplicas == want && dep.Status.UpdatedReplicas == want && dep.Status.AvailableReplicas == want && container.Image != ""
	return ready, int(want), int(dep.Status.ReadyReplicas), container.Image, version, nil
}

// SetCanaryWeight stores the weight and tells checkout-api. The browser cannot choose the service.
func (c *Client) SetCanaryWeight(ctx context.Context, token string, weight int) error {
	switch weight {
	case 0, 5, 25, 50, 100:
	default:
		return fmt.Errorf("%w: weight %d", release.ErrDenied, weight)
	}
	if err := c.writeWeight(ctx, weight); err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("fault token is not configured")
	}
	payload := []byte(fmt.Sprintf(`{"weight":%d}`, weight))
	code, raw, err := c.Proxy(ctx, release.Namespace, "checkout-api", 80, "POST", "/internal/canary", nil, payload, map[string]string{
		"Content-Type":           "application/json",
		"X-OpsPilot-Fault-Token": token,
	})
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("checkout canary weight returned %d: %s", code, trim(raw))
	}
	return nil
}

func (c *Client) writeWeight(ctx context.Context, weight int) error {
	cm, err := c.clientset.CoreV1().ConfigMaps(release.Namespace).Get(ctx, release.RoutingConfigMap, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = c.clientset.CoreV1().ConfigMaps(release.Namespace).Create(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: release.RoutingConfigMap, Namespace: release.Namespace},
			Data:       map[string]string{"weight": fmt.Sprintf("%d", weight)},
		}, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data["weight"] = fmt.Sprintf("%d", weight)
	_, err = c.clientset.CoreV1().ConfigMaps(release.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

// PromoteCandidate makes payment-api the healthy candidate and scales the canary to zero.
func (c *Client) PromoteCandidate(ctx context.Context, image, version string) error {
	if !release.Promotable(image, version) {
		return fmt.Errorf("%w: only %s can be promoted", release.ErrDenied, release.CandidateGoodVersion)
	}
	if err := c.setStable(ctx, image, version); err != nil {
		return err
	}
	return c.ScaleCanary(ctx, 0)
}

// RemoveCanary drops candidate traffic and scales the canary Deployment to zero.
func (c *Client) RemoveCanary(ctx context.Context, token string) error {
	if err := c.SetCanaryWeight(ctx, token, 0); err != nil {
		return err
	}
	return c.ScaleCanary(ctx, 0)
}

// RestoreBaseline returns payment-api to 1.4.2 and removes canary traffic.
func (c *Client) RestoreBaseline(ctx context.Context, token string) error {
	current, _, err := c.CurrentPayment(ctx)
	if err != nil {
		return err
	}
	if !release.BaselineImage(current) && current != "" {
		return fmt.Errorf("%w: refusing to restore from %s", release.ErrDenied, current)
	}
	if err := c.setStable(ctx, release.GoodImage, release.GoodVersion); err != nil {
		return err
	}
	return c.RemoveCanary(ctx, token)
}

func (c *Client) setStable(ctx context.Context, image, version string) error {
	dep, err := c.clientset.AppsV1().Deployments(release.Namespace).Get(ctx, release.Deployment, metav1.GetOptions{})
	if err != nil {
		return err
	}
	container, err := paymentContainer(dep)
	if err != nil {
		return err
	}
	container.Image = image
	setEnv(container, "SERVICE_VERSION", version)
	setEnv(container, "RELEASE_ROLE", "stable")
	if dep.Labels == nil {
		dep.Labels = map[string]string{}
	}
	dep.Labels["app.kubernetes.io/version"] = version
	if dep.Spec.Template.Labels == nil {
		dep.Spec.Template.Labels = map[string]string{}
	}
	dep.Spec.Template.Labels["app.kubernetes.io/version"] = version
	replicas := int32(1)
	dep.Spec.Replicas = &replicas
	_, err = c.clientset.AppsV1().Deployments(release.Namespace).Update(ctx, dep, metav1.UpdateOptions{})
	return err
}

// ScaleCanary sets the candidate replica count to 0 or 1.
func (c *Client) ScaleCanary(ctx context.Context, replicas int) error {
	if replicas != 0 && replicas != 1 {
		return fmt.Errorf("%w: canary replicas %d", release.ErrDenied, replicas)
	}
	dep, err := c.clientset.AppsV1().Deployments(release.Namespace).Get(ctx, release.CanaryDeployment, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	count := int32(replicas)
	dep.Spec.Replicas = &count
	_, err = c.clientset.AppsV1().Deployments(release.Namespace).Update(ctx, dep, metav1.UpdateOptions{})
	return err
}

// WaitCanaryReady waits until the candidate is Ready on the expected image.
func (c *Client) WaitCanaryReady(ctx context.Context, image string, timeout time.Duration) error {
	if !release.Candidate(image, release.CandidateGoodVersion) && !release.Candidate(image, release.CandidateBadVersion) {
		return fmt.Errorf("%w: refusing to wait for %s", release.ErrDenied, image)
	}
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		ready, _, _, current, _, err := c.CanaryStatus(ctx)
		if err != nil {
			return err
		}
		if ready && current == image {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("payment-api canary did not become ready on %s", image)
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func namedContainer(dep *appsv1.Deployment, name string) (*corev1.Container, error) {
	for i := range dep.Spec.Template.Spec.Containers {
		if dep.Spec.Template.Spec.Containers[i].Name == name {
			return &dep.Spec.Template.Spec.Containers[i], nil
		}
	}
	return nil, fmt.Errorf("%w: deployment has no %s container", release.ErrDenied, name)
}

func canaryDeployment(image, version string) *appsv1.Deployment {
	replicas := int32(1)
	labels := map[string]string{
		"app.kubernetes.io/name":    release.CanaryDeployment,
		"app.kubernetes.io/part-of": "demo-shop",
		"app.kubernetes.io/version": version,
		"opspilot.io/managed":       "true",
		"opspilot.io/service":       release.Deployment,
		"opspilot.io/role":          "canary",
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: release.CanaryDeployment, Namespace: release.Namespace, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": release.CanaryDeployment}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            release.Container,
						Image:           image,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Ports:           []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}},
						Env: []corev1.EnvVar{
							{Name: "SERVICE_NAME", Value: release.Deployment},
							{Name: "SERVICE_VERSION", Value: version},
							{Name: "RELEASE_ROLE", Value: "canary"},
							{Name: "NAMESPACE", Value: release.Namespace},
							{Name: "OTEL_EXPORTER_OTLP_ENDPOINT", Value: "http://otel-collector.opspilot-system.svc:4318"},
							{Name: "FAULT_TOKEN", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "payment-fault"},
								Key:                  "token",
							}}},
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler:        corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstr.FromString("http")}},
							InitialDelaySeconds: 1,
							PeriodSeconds:       5,
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler:        corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstr.FromString("http")}},
							InitialDelaySeconds: 3,
							PeriodSeconds:       10,
						},
					}},
				},
			},
		},
	}
}
