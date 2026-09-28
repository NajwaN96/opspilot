package kubernetes

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/opspilot/opspilot/apps/api/internal/release"
)

// SetPaymentRelease updates only demo-shop/payment-api to a known image and version.
// It is not part of Reader. The detection engine cannot call it.
func (c *Client) SetPaymentRelease(ctx context.Context, image, version string) error {
	if c == nil || c.clientset == nil {
		return fmt.Errorf("kubernetes client is not configured")
	}
	if !release.Known(image, version) {
		return fmt.Errorf("%w: image/version is not a known payment-api release", release.ErrDenied)
	}
	dep, err := c.clientset.AppsV1().Deployments(release.Namespace).Get(ctx, release.Deployment, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if dep.Namespace != release.Namespace || dep.Name != release.Deployment {
		return fmt.Errorf("%w: refusing %s/%s", release.ErrDenied, dep.Namespace, dep.Name)
	}
	container, err := paymentContainer(dep)
	if err != nil {
		return err
	}
	container.Image = image
	setEnv(container, "SERVICE_VERSION", version)
	if dep.Labels == nil {
		dep.Labels = map[string]string{}
	}
	dep.Labels["app.kubernetes.io/version"] = version
	if dep.Spec.Template.Labels == nil {
		dep.Spec.Template.Labels = map[string]string{}
	}
	dep.Spec.Template.Labels["app.kubernetes.io/version"] = version
	_, err = c.clientset.AppsV1().Deployments(release.Namespace).Update(ctx, dep, metav1.UpdateOptions{})
	return err
}

// CurrentPayment reads the live payment-api image and version label.
func (c *Client) CurrentPayment(ctx context.Context) (string, string, error) {
	if c == nil || c.clientset == nil {
		return "", "", fmt.Errorf("kubernetes client is not configured")
	}
	dep, err := c.clientset.AppsV1().Deployments(release.Namespace).Get(ctx, release.Deployment, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	container, err := paymentContainer(dep)
	if err != nil {
		return "", "", err
	}
	version := dep.Labels["app.kubernetes.io/version"]
	if version == "" {
		version = dep.Spec.Template.Labels["app.kubernetes.io/version"]
	}
	return container.Image, version, nil
}

// WaitPaymentReady blocks until the named known image is the ready spec.
func (c *Client) WaitPaymentReady(ctx context.Context, image string, timeout time.Duration) error {
	if image != release.GoodImage && image != release.BadImage {
		return fmt.Errorf("%w: refusing to wait for %s", release.ErrDenied, image)
	}
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		ready, err := c.paymentReady(ctx, image)
		if err != nil {
			return err
		}
		if ready {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("payment-api did not become ready on %s", image)
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

func (c *Client) paymentReady(ctx context.Context, image string) (bool, error) {
	dep, err := c.clientset.AppsV1().Deployments(release.Namespace).Get(ctx, release.Deployment, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	container, err := paymentContainer(dep)
	if err != nil {
		return false, err
	}
	if container.Image != image {
		return false, nil
	}
	desired := int32(1)
	if dep.Spec.Replicas != nil {
		desired = *dep.Spec.Replicas
	}
	if desired == 0 {
		return false, nil
	}
	return dep.Status.ReadyReplicas == desired &&
		dep.Status.UpdatedReplicas == desired &&
		dep.Status.AvailableReplicas == desired, nil
}

func paymentContainer(dep *appsv1.Deployment) (*corev1.Container, error) {
	for i := range dep.Spec.Template.Spec.Containers {
		if dep.Spec.Template.Spec.Containers[i].Name == release.Container {
			return &dep.Spec.Template.Spec.Containers[i], nil
		}
	}
	return nil, fmt.Errorf("%w: deployment has no %s container", release.ErrDenied, release.Container)
}

func setEnv(container *corev1.Container, name, value string) {
	for i := range container.Env {
		if container.Env[i].Name == name {
			container.Env[i].Value = value
			return
		}
	}
	container.Env = append(container.Env, corev1.EnvVar{Name: name, Value: value})
}
