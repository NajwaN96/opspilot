package kubernetes

import (
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/opspilot/opspilot/apps/api/internal/model"
)

const sourceKubernetes = "kubernetes"

// ServiceID is the OpsPilot identifier for a discovered workload.
// It avoids slashes so it can be used as a single URL path segment.
func ServiceID(namespace, name string) string {
	return "k8s_" + namespace + "_" + name
}

func WorkloadFromDeployment(dep appsv1.Deployment, pods []corev1.Pod, observed time.Time) model.Workload {
	desired := 1
	if dep.Spec.Replicas != nil {
		desired = int(*dep.Spec.Replicas)
	}
	ready := int(dep.Status.ReadyReplicas)
	owned := podsForDeployment(dep.Name, pods)
	restarts := 0
	for _, pod := range owned {
		restarts += restartCount(pod)
	}
	image, version := imageAndVersion(dep)
	labels := map[string]string{}
	for key, value := range dep.Labels {
		labels[key] = value
	}
	item := model.Workload{
		Name:         dep.Name,
		Namespace:    dep.Namespace,
		Kind:         "Deployment",
		Version:      version,
		Image:        image,
		Desired:      desired,
		Ready:        ready,
		Restarts:     restarts,
		Status:       workloadStatus(desired, ready),
		LastObserved: observed,
		ServiceID:    ServiceID(dep.Namespace, dep.Name),
		Source:       sourceKubernetes,
		Labels:       labels,
	}
	for _, pod := range owned {
		item.Pods = append(item.Pods, PodFrom(pod, dep.Name))
	}
	return item
}

func PodFrom(pod corev1.Pod, service string) model.Pod {
	if service == "" {
		service = serviceName(pod)
	}
	ready, total := readyContainers(pod)
	started := pod.Status.StartTime
	var startedAt *time.Time
	if started != nil && !started.IsZero() {
		t := started.Time
		startedAt = &t
	}
	return model.Pod{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Service:   service,
		Status:    string(pod.Status.Phase),
		Ready:     fmt.Sprintf("%d/%d", ready, total),
		Restarts:  restartCount(pod),
		Node:      pod.Spec.NodeName,
		StartedAt: startedAt,
		Source:    sourceKubernetes,
	}
}

func EventFrom(ev corev1.Event) model.ClusterEvent {
	count := int(ev.Count)
	if count == 0 {
		count = 1
	}
	at := ev.LastTimestamp.Time
	if at.IsZero() {
		at = ev.EventTime.Time
	}
	if at.IsZero() {
		at = ev.CreationTimestamp.Time
	}
	object := ev.InvolvedObject.Kind + "/" + ev.InvolvedObject.Name
	id := string(ev.UID)
	if id == "" {
		id = ev.Namespace + "/" + ev.Name
	}
	return model.ClusterEvent{
		ID:        id,
		At:        at,
		Namespace: ev.Namespace,
		Type:      ev.Type,
		Reason:    ev.Reason,
		Object:    object,
		Message:   ev.Message,
		Count:     count,
		Source:    sourceKubernetes,
	}
}

func ServiceFromWorkload(clusterID string, workload model.Workload) model.Service {
	observed := workload.LastObserved
	return model.Service{
		ID:           workload.ServiceID,
		Name:         workload.Name,
		ClusterID:    clusterID,
		Namespace:    workload.Namespace,
		Status:       workload.Status,
		Owner:        "demo-shop",
		Runtime:      "kubernetes",
		Description:  "Discovered from a Kubernetes Deployment. Reliability telemetry is not collected for this workload.",
		Version:      workload.Version,
		Source:       sourceKubernetes,
		Telemetry:    "none",
		Image:        workload.Image,
		WorkloadKind: workload.Kind,
		Restarts:     workload.Restarts,
		LastObserved: &observed,
		Replicas:     model.Replicas{Desired: workload.Desired, Ready: workload.Ready},
	}
}

func workloadStatus(desired, ready int) string {
	if desired == 0 && ready == 0 {
		return model.StatusIdle
	}
	if desired == 0 {
		return model.StatusDegraded
	}
	if ready >= desired {
		return model.StatusHealthy
	}
	return model.StatusDegraded
}

func imageAndVersion(dep appsv1.Deployment) (string, string) {
	image := ""
	if len(dep.Spec.Template.Spec.Containers) > 0 {
		image = dep.Spec.Template.Spec.Containers[0].Image
	}
	if version := dep.Labels["app.kubernetes.io/version"]; version != "" {
		return image, version
	}
	if tag := imageTag(image); tag != "" {
		return image, tag
	}
	return image, ""
}

func imageTag(image string) string {
	slash := strings.LastIndex(image, "/")
	last := image
	if slash >= 0 {
		last = image[slash+1:]
	}
	colon := strings.LastIndex(last, ":")
	if colon < 0 || colon == len(last)-1 {
		return ""
	}
	return last[colon+1:]
}

func podsForDeployment(name string, pods []corev1.Pod) []corev1.Pod {
	out := make([]corev1.Pod, 0)
	for _, pod := range pods {
		if serviceName(pod) == name || strings.HasPrefix(pod.Name, name+"-") {
			out = append(out, pod)
		}
	}
	return out
}

func serviceName(pod corev1.Pod) string {
	if value := pod.Labels["opspilot.io/service"]; value != "" {
		return value
	}
	if value := pod.Labels["app.kubernetes.io/name"]; value != "" {
		return value
	}
	return ""
}

func restartCount(pod corev1.Pod) int {
	total := 0
	for _, status := range pod.Status.ContainerStatuses {
		total += int(status.RestartCount)
	}
	return total
}

func readyContainers(pod corev1.Pod) (int, int) {
	ready := 0
	total := len(pod.Spec.Containers)
	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			ready++
		}
	}
	return ready, total
}
