package kubernetes

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/opspilot/opspilot/apps/api/internal/model"
)

// Reader is the only Kubernetes surface the control plane is allowed to call.
// It has no create, update, delete, or exec methods.
type Reader interface {
	Status(ctx context.Context) (model.KubernetesStatus, error)
	Namespaces(ctx context.Context) ([]model.Namespace, error)
	Workloads(ctx context.Context) ([]model.Workload, error)
	Workload(ctx context.Context, namespace, name string) (model.Workload, error)
	Pods(ctx context.Context) ([]model.Pod, error)
	Events(ctx context.Context) ([]model.ClusterEvent, error)
}

// Client lists objects in an allow-list of namespaces.
type Client struct {
	clientset  kubernetes.Interface
	cluster    string
	namespaces []string
	platform   string
}

type Options struct {
	Kubeconfig string
	Cluster    string
	Namespaces []string
	Platform   string
	Clientset  kubernetes.Interface
}

func New(opts Options) (*Client, error) {
	if opts.Clientset == nil {
		return nil, fmt.Errorf("kubernetes clientset is required")
	}
	namespaces := cleanNamespaces(opts.Namespaces)
	if len(namespaces) == 0 {
		namespaces = []string{"demo-shop"}
	}
	cluster := opts.Cluster
	if cluster == "" {
		cluster = "opspilot-dev"
	}
	platform := opts.Platform
	if platform == "" {
		platform = "opspilot-system"
	}
	return &Client{
		clientset:  opts.Clientset,
		cluster:    cluster,
		namespaces: namespaces,
		platform:   platform,
	}, nil
}

func Load(opts Options) (Reader, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if opts.Kubeconfig != "" {
		rules.ExplicitPath = opts.Kubeconfig
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, err
	}
	cfg.QPS = 20
	cfg.Burst = 40
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	opts.Clientset = clientset
	return New(opts)
}

func (c *Client) Status(ctx context.Context) (model.KubernetesStatus, error) {
	status := c.baseStatus()
	version, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		status.Connectivity = "disconnected"
		status.Message = "Kubernetes API is not reachable"
		return status, nil
	}
	status.KubernetesVersion = version.GitVersion
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		status.Connectivity = "degraded"
		status.Message = "Node list failed"
		stamp(&status)
		return status, nil
	}
	status.NodeCount = len(nodes.Items)
	for _, node := range nodes.Items {
		if nodeReady(node) {
			status.NodesReady++
		}
	}
	if status.NodeCount == 0 || status.NodesReady == 0 {
		status.Connectivity = "degraded"
		status.Message = "No Ready nodes"
	}
	stamp(&status)
	return status, nil
}

func (c *Client) Namespaces(ctx context.Context) ([]model.Namespace, error) {
	names := append([]string{}, c.namespaces...)
	if c.platform != "" && !contains(names, c.platform) {
		names = append(names, c.platform)
	}
	out := make([]model.Namespace, 0, len(names))
	for _, name := range names {
		item, err := c.clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			continue
		}
		deployments, err := c.clientset.AppsV1().Deployments(name).List(ctx, metav1.ListOptions{})
		count := 0
		if err == nil {
			count = len(deployments.Items)
		}
		phase := string(item.Status.Phase)
		if phase == "" {
			phase = "Active"
		}
		out = append(out, model.Namespace{Name: item.Name, Services: count, Status: strings.ToLower(phase)})
	}
	return out, nil
}

func (c *Client) Workloads(ctx context.Context) ([]model.Workload, error) {
	observed := time.Now().UTC()
	var out []model.Workload
	for _, namespace := range c.namespaces {
		deployments, pods, err := c.lists(ctx, namespace)
		if err != nil {
			return nil, err
		}
		for _, dep := range deployments {
			out = append(out, WorkloadFromDeployment(dep, pods, observed))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace == out[j].Namespace {
			return out[i].Name < out[j].Name
		}
		return out[i].Namespace < out[j].Namespace
	})
	return out, nil
}

func (c *Client) Workload(ctx context.Context, namespace, name string) (model.Workload, error) {
	if !c.allowed(namespace) || !validName(name) {
		return model.Workload{}, fmt.Errorf("workload %s/%s is outside the discovery scope", namespace, name)
	}
	dep, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return model.Workload{}, err
	}
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return model.Workload{}, err
	}
	return WorkloadFromDeployment(*dep, pods.Items, time.Now().UTC()), nil
}

func (c *Client) Pods(ctx context.Context) ([]model.Pod, error) {
	var out []model.Pod
	for _, namespace := range c.namespaces {
		pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, pod := range pods.Items {
			out = append(out, PodFrom(pod, ""))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (c *Client) Events(ctx context.Context) ([]model.ClusterEvent, error) {
	var out []model.ClusterEvent
	for _, namespace := range c.namespaces {
		events, err := c.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, event := range events.Items {
			out = append(out, EventFrom(event))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	if len(out) > 40 {
		out = out[:40]
	}
	return out, nil
}

func (c *Client) lists(ctx context.Context, namespace string) ([]appsv1.Deployment, []corev1.Pod, error) {
	deployments, err := c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	return deployments.Items, pods.Items, nil
}

func (c *Client) allowed(namespace string) bool {
	return contains(c.namespaces, namespace)
}

func (c *Client) baseStatus() model.KubernetesStatus {
	primary := ""
	if len(c.namespaces) > 0 {
		primary = c.namespaces[0]
	}
	return model.KubernetesStatus{
		Cluster:      c.cluster,
		Mode:         "local-kubernetes",
		Namespace:    primary,
		Namespaces:   append([]string{}, c.namespaces...),
		Connectivity: "connected",
		Source:       sourceKubernetes,
	}
}

func stamp(status *model.KubernetesStatus) {
	now := time.Now().UTC()
	status.LastSync = &now
}

func nodeReady(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func cleanNamespaces(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] || !validName(item) {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func validName(value string) bool {
	if value == "" || len(value) > 63 {
		return false
	}
	for i, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
		if !ok {
			return false
		}
		if (i == 0 || i == len(value)-1) && r == '-' {
			return false
		}
	}
	return true
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
