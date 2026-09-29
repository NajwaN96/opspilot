package release

import (
	"errors"
	"fmt"

	"github.com/opspilot/opspilot/apps/api/internal/model"
)

// The only Kubernetes remediation this process can run.
// Image names are fixed here. Callers cannot substitute a namespace, Deployment, or tag.
const (
	Action      = "rollback-payment-api"
	Cluster     = "opspilot-dev"
	Namespace   = "demo-shop"
	Deployment  = "payment-api"
	Container   = "payment-api"
	GoodVersion = "1.4.2"
	BadVersion  = "1.5.0-bad"
	GoodImage   = "opspilot-demo:1.4.2"
	BadImage    = "opspilot-demo:1.5.0-bad"

	CandidateGoodVersion = "1.5.0"
	CandidateGoodImage   = "opspilot-demo:1.5.0"
	CandidateBadVersion  = "1.6.0-bad"
	CandidateBadImage    = "opspilot-demo:1.6.0-bad"
	CanaryDeployment     = "payment-api-canary"
	CanaryService        = "payment-api-canary"
	RoutingConfigMap     = "payment-routing"
)

// ErrDenied is a policy or target rejection. It is not a cluster outage.
var ErrDenied = errors.New("payment rollback denied")

// Candidate reports whether the image and version are a canary release.
// Promotion is allowed only for the healthy candidate. The bad candidate can only be removed.
func Candidate(image, version string) bool {
	switch {
	case image == CandidateGoodImage && version == CandidateGoodVersion:
		return true
	case image == CandidateBadImage && version == CandidateBadVersion:
		return true
	default:
		return false
	}
}

// Promotable is the only candidate that may replace the stable payment-api Deployment.
func Promotable(image, version string) bool {
	return image == CandidateGoodImage && version == CandidateGoodVersion
}

// BaselineImage is a payment-api image this process is allowed to restore.
func BaselineImage(image string) bool {
	switch image {
	case GoodImage, BadImage, CandidateGoodImage, CandidateBadImage:
		return true
	default:
		return false
	}
}

// Known reports whether the image and version are one of the two server-side releases.
func Known(image, version string) bool {
	switch {
	case image == GoodImage && version == GoodVersion:
		return true
	case image == BadImage && version == BadVersion:
		return true
	default:
		return false
	}
}

// Propose builds the remediation offer from the observed workload version.
// The bad release is the only version that can be approved.
func Propose(version string) model.Recommendation {
	if version == BadVersion {
		return model.Recommendation{
			Action:  Action,
			Summary: "Roll demo-shop/payment-api back from 1.5.0-bad to 1.4.2.",
			From:    BadVersion,
			To:      GoodVersion,
			Risk:    "Changes only the payment-api container image, its version label, and SERVICE_VERSION. No other Deployment is updated.",
			Policy:  "Allowed only while the observed payment-api version is the known bad release 1.5.0-bad. The target image is fixed on the server.",
			Allowed: true,
		}
	}
	return model.Recommendation{
		Action:  "stop-experiment",
		Summary: "Disable the active Reliability Lab fault if one is running. This does not roll back a Deployment.",
		Risk:    "Local demo-shop only. Kubernetes mutation stays closed unless payment-api is the known bad release.",
		Policy:  "Real Kubernetes rollback is allowed only for payment-api 1.5.0-bad to 1.4.2. The current version does not match.",
		Allowed: false,
	}
}

// Validate checks an already-authorized request. It does not read the cluster.
func Validate(action, service, namespace, from, to string) error {
	if action != Action || service != Deployment || namespace != Namespace || from != BadVersion || to != GoodVersion {
		return fmt.Errorf("%w: action %s service %s namespace %s from %s to %s", ErrDenied, action, service, namespace, from, to)
	}
	return nil
}
