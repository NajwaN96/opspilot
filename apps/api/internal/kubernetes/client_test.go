package kubernetes

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestClientDiscoversOnlyConfiguredNamespace(t *testing.T) {
	replicas := int32(1)
	client := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "demo-shop"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "opspilot-system"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "k3d-opspilot-dev-server-0"}, Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}}},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "demo-shop", Labels: map[string]string{"app.kubernetes.io/version": "1.4.2", "opspilot.io/service": "payment-api"}},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas, Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: "nginx:1.27-alpine"}}}}},
			Status:     appsv1.DeploymentStatus{ReadyReplicas: 1},
		},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "hidden", Namespace: "kube-system"}, Spec: appsv1.DeploymentSpec{Replicas: &replicas}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "payment-api-a", Namespace: "demo-shop", Labels: map[string]string{"opspilot.io/service": "payment-api"}}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "ev", Namespace: "demo-shop"}, Reason: "Pulled", Type: "Normal", Message: "image pulled", InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "payment-api-a"}},
	)
	reader, err := New(Options{Clientset: client, Cluster: "opspilot-dev", Namespaces: []string{"demo-shop"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	status, err := reader.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.Connectivity != "connected" || status.Cluster != "opspilot-dev" || status.NodesReady != 1 {
		t.Fatalf("status %#v", status)
	}
	workloads, err := reader.Workloads(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(workloads) != 1 || workloads[0].Name != "payment-api" {
		t.Fatalf("workloads %#v", workloads)
	}
	again, err := reader.Workloads(ctx)
	if err != nil || len(again) != 1 || again[0].ServiceID != workloads[0].ServiceID {
		t.Fatalf("sync not stable %#v", again)
	}
	if _, err := reader.Workload(ctx, "kube-system", "hidden"); err == nil {
		t.Fatal("expected out-of-scope workload to be rejected")
	}
	namespaces, err := reader.Namespaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(namespaces) != 2 {
		t.Fatalf("namespaces %#v", namespaces)
	}
	pods, err := reader.Pods(ctx)
	if err != nil || len(pods) != 1 || pods[0].Service != "payment-api" {
		t.Fatalf("pods %#v %v", pods, err)
	}
	events, err := reader.Events(ctx)
	if err != nil || len(events) != 1 || events[0].Reason != "Pulled" {
		t.Fatalf("events %#v %v", events, err)
	}
}
