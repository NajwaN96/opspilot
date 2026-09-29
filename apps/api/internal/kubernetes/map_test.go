package kubernetes

import (
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/opspilot/opspilot/apps/api/internal/model"
)

func TestWorkloadMapping(t *testing.T) {
	replicas := int32(2)
	dep := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "payment-api",
			Namespace: "demo-shop",
			Labels: map[string]string{
				"app.kubernetes.io/name":    "payment-api",
				"app.kubernetes.io/version": "1.4.2",
				"opspilot.io/service":       "payment-api",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Image: "nginx:1.27-alpine"}}},
			},
		},
		Status: appsv1.DeploymentStatus{ReadyReplicas: 1},
	}
	started := metav1.NewTime(time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC))
	pods := []corev1.Pod{{
		ObjectMeta: metav1.ObjectMeta{Name: "payment-api-abc", Namespace: "demo-shop", Labels: map[string]string{"opspilot.io/service": "payment-api"}},
		Spec:       corev1.PodSpec{NodeName: "node-a", Containers: []corev1.Container{{Name: "api"}}},
		Status: corev1.PodStatus{
			Phase:     corev1.PodRunning,
			StartTime: &started,
			ContainerStatuses: []corev1.ContainerStatus{{
				Ready:        true,
				RestartCount: 3,
			}},
		},
	}}
	observed := time.Date(2026, 9, 28, 12, 5, 0, 0, time.UTC)
	workload := WorkloadFromDeployment(dep, pods, observed)
	if workload.ServiceID != "k8s_demo-shop_payment-api" {
		t.Fatalf("service id %s", workload.ServiceID)
	}
	if workload.Version != "1.4.2" || workload.Image != "nginx:1.27-alpine" {
		t.Fatalf("image/version %#v", workload)
	}
	if workload.Desired != 2 || workload.Ready != 1 || workload.Restarts != 3 {
		t.Fatalf("replicas/restarts %#v", workload)
	}
	if workload.Status != model.StatusDegraded || workload.Source != "kubernetes" {
		t.Fatalf("status %s source %s", workload.Status, workload.Source)
	}
	if len(workload.Pods) != 1 || workload.Pods[0].Ready != "1/1" || workload.Pods[0].Node != "node-a" {
		t.Fatalf("pods %#v", workload.Pods)
	}
	svc := ServiceFromWorkload("opspilot-dev", workload)
	if svc.Source != "kubernetes" || svc.Telemetry != "none" || svc.ClusterID != "opspilot-dev" {
		t.Fatalf("service %#v", svc)
	}
}

func TestZeroReplicasAreIdle(t *testing.T) {
	zero := int32(0)
	one := int32(1)
	idle := WorkloadFromDeployment(appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "payment-api-canary", Namespace: "demo-shop"},
		Spec:       appsv1.DeploymentSpec{Replicas: &zero},
	}, nil, time.Now())
	if idle.Status != model.StatusIdle || idle.Desired != 0 || idle.Ready != 0 {
		t.Fatalf("idle %#v", idle)
	}
	down := WorkloadFromDeployment(appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "demo-shop"},
		Spec:       appsv1.DeploymentSpec{Replicas: &one},
		Status:     appsv1.DeploymentStatus{ReadyReplicas: 0},
	}, nil, time.Now())
	if down.Status != model.StatusDegraded {
		t.Fatalf("active without ready pods is %s", down.Status)
	}
}

func TestEventMapping(t *testing.T) {
	event := EventFrom(corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{Name: "payment.1", Namespace: "demo-shop", UID: "uid-1"},
		Type:           "Warning",
		Reason:         "Unhealthy",
		Message:        "probe failed",
		Count:          2,
		LastTimestamp:  metav1.NewTime(time.Date(2026, 9, 28, 12, 1, 0, 0, time.UTC)),
		InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "payment-api-abc"},
	})
	if event.ID != "uid-1" || event.Object != "Pod/payment-api-abc" || event.Count != 2 || event.Source != "kubernetes" {
		t.Fatalf("%#v", event)
	}
}
