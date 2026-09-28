package kubernetes

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/opspilot/opspilot/apps/api/internal/release"
)

func TestSetPaymentReleaseRejectsUnknownImagesAndOtherDeployments(t *testing.T) {
	storefront := deployment("demo-shop", "storefront", "storefront", "opspilot-demo:0.3.0", "1.6.0", 1)
	payment := deployment(release.Namespace, release.Deployment, release.Container, release.GoodImage, release.GoodVersion, 1)
	otherNS := deployment("opspilot-system", release.Deployment, release.Container, release.GoodImage, release.GoodVersion, 1)
	client := testClient(t, storefront, payment, otherNS)

	if err := client.SetPaymentRelease(context.Background(), "nginx:latest", "1"); err == nil {
		t.Fatal("unknown image accepted")
	}
	if err := client.SetPaymentRelease(context.Background(), release.BadImage, release.GoodVersion); err == nil {
		t.Fatal("mismatched pair accepted")
	}

	got, err := client.clientset.AppsV1().Deployments("demo-shop").Get(context.Background(), "storefront", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.Template.Spec.Containers[0].Image != "opspilot-demo:0.3.0" {
		t.Fatalf("storefront changed to %s", got.Spec.Template.Spec.Containers[0].Image)
	}
	system, err := client.clientset.AppsV1().Deployments("opspilot-system").Get(context.Background(), release.Deployment, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if system.Spec.Template.Spec.Containers[0].Image != release.GoodImage {
		t.Fatalf("opspilot-system changed to %s", system.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestSetPaymentReleaseUpdatesOnlyTheKnownDeployment(t *testing.T) {
	client := testClient(t, deployment(release.Namespace, release.Deployment, release.Container, release.GoodImage, release.GoodVersion, 0))
	if err := client.SetPaymentRelease(context.Background(), release.BadImage, release.BadVersion); err != nil {
		t.Fatal(err)
	}
	got, err := client.clientset.AppsV1().Deployments(release.Namespace).Get(context.Background(), release.Deployment, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	container := got.Spec.Template.Spec.Containers[0]
	if container.Image != release.BadImage {
		t.Fatalf("image %s", container.Image)
	}
	if container.Env[0].Value != release.BadVersion {
		t.Fatalf("env %s", container.Env[0].Value)
	}
	if got.Labels["app.kubernetes.io/version"] != release.BadVersion {
		t.Fatalf("label %s", got.Labels["app.kubernetes.io/version"])
	}
	if got.Spec.Template.Labels["app.kubernetes.io/version"] != release.BadVersion {
		t.Fatalf("template label %s", got.Spec.Template.Labels["app.kubernetes.io/version"])
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := client.WaitPaymentReady(ctx, release.BadImage, 15*time.Millisecond); err == nil {
		t.Fatal("not-ready deployment reported ready")
	}
}

func testClient(t *testing.T, objects ...runtime.Object) *Client {
	t.Helper()
	client, err := New(Options{Clientset: fake.NewSimpleClientset(objects...)})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func deployment(namespace, name, container, image, version string, ready int32) *appsv1.Deployment {
	replicas := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app.kubernetes.io/version": version},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app.kubernetes.io/version": version}},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Name:  container,
					Image: image,
					Env:   []corev1.EnvVar{{Name: "SERVICE_VERSION", Value: version}},
				}}},
			},
		},
		Status: appsv1.DeploymentStatus{ReadyReplicas: ready, UpdatedReplicas: ready, AvailableReplicas: ready},
	}
}
