package experiment

import (
	"fmt"
	"strings"
	"time"
)

const (
	ScenarioPaymentDegraded = "payment-api-degraded"
	Cluster                 = "opspilot-dev"
	Namespace               = "demo-shop"
	Service                 = "payment-api"
	ServiceID               = "k8s_demo-shop_payment-api"
	MaxDuration             = 5 * time.Minute
	Actor                   = "local-operator"
)

// Policy is the server-side allowlist for a real local experiment.
// The browser cannot widen the cluster, namespace, service, or duration.
type Policy struct {
	Allow bool
}

func (p Policy) Validate(serviceID, scenario string, durationSec int) error {
	if !p.Allow {
		return fmt.Errorf("real experiments are disabled outside local development")
	}
	if scenario != ScenarioPaymentDegraded {
		return fmt.Errorf("scenario is not in the real experiment allowlist")
	}
	if serviceID != ServiceID {
		return fmt.Errorf("real experiments can target only %s in %s/%s", Service, Namespace, Cluster)
	}
	switch durationSec {
	case 30, 60, 120:
	default:
		return fmt.Errorf("duration must be 30, 60, or 120 seconds")
	}
	if time.Duration(durationSec)*time.Second > MaxDuration {
		return fmt.Errorf("duration exceeds 5 minutes")
	}
	return nil
}

func Parameters() map[string]string {
	return map[string]string{
		"cluster":      Cluster,
		"namespace":    Namespace,
		"service":      Service,
		"errorPercent": "40",
		"latency":      "500ms",
	}
}

func SafeDetail(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	if len(value) > 180 {
		return value[:180]
	}
	return value
}
