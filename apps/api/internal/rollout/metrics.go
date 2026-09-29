package rollout

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Counters are OpsPilot's own rollout metrics. Labels stay bounded: service, result, stage, action.
type Counters struct {
	mu     sync.Mutex
	values map[string]float64
}

func NewCounters() *Counters {
	return &Counters{values: map[string]float64{}}
}

func (c *Counters) Add(name string, labels map[string]string, value float64) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[name+series(labels)] += value
}

func (c *Counters) Render() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]string, 0, len(c.values))
	for key := range c.values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&b, "%s %g\n", key, c.values[key])
	}
	return b.String()
}

func series(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, key, labels[key]))
	}
	return "{" + strings.Join(parts, ",") + "}"
}
