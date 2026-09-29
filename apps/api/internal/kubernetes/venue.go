package kubernetes

// Classify separates the local k3d cluster from the portfolio EKS cluster.
// An unknown name is not treated as either environment.
func Classify(cluster string) (mode, venue, label string) {
	switch cluster {
	case "opspilot-aws-dev":
		return "eks", "aws-dev", "AWS DEV"
	case "", "opspilot-dev":
		return "local-kubernetes", "local", "LOCAL"
	default:
		return "unknown", "unknown", "UNKNOWN"
	}
}
