package cloudstatus

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Status is the only cloud view the console may show.
// It never carries an account id, caller ARN, or credential.
type Status struct {
	AccountState                   string `json:"accountState"`
	AccountMasked                  string `json:"accountMasked,omitempty"`
	Region                         string `json:"region,omitempty"`
	Authentication                 string `json:"authentication"`
	CloudDeployment                string `json:"cloudDeployment"`
	KubernetesRuntime              string `json:"kubernetesRuntime"`
	PaidCloudResources             string `json:"paidCloudResources"`
	IntentionalInfrastructureSpend string `json:"intentionalInfrastructureSpend"`
	EKS                            string `json:"eks"`
}

// Probe is a read-only identity result. ARN is accepted so callers can drop it.
type Probe struct {
	Account string
	Region  string
	Method  string
	Err     error
}

var (
	accountID = regexp.MustCompile(`^[0-9]{12}$`)
	regionRE  = regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-\d+$`)
)

// FromProbe builds the portfolio cloud card. A full account id is masked to
// the last four digits and is not retained.
func FromProbe(p Probe) Status {
	status := Status{
		AccountState:                   "not-connected",
		Authentication:                 "none",
		CloudDeployment:                "PLAN ONLY",
		KubernetesRuntime:              "LOCAL k3d",
		PaidCloudResources:             "BLOCKED",
		IntentionalInfrastructureSpend: "$0",
		EKS:                            "plan-only",
	}
	if p.Err != nil || !accountID.MatchString(p.Account) {
		return status
	}
	status.AccountState = "connected"
	status.AccountMasked = "********" + p.Account[8:]
	switch p.Method {
	case "console-login":
		status.Authentication = "console-login"
	default:
		status.Authentication = "profile"
	}
	if regionRE.MatchString(p.Region) && !strings.Contains(p.Region, "gov") && !strings.HasPrefix(p.Region, "cn-") {
		status.Region = p.Region
	}
	return status
}

// Current asks the AWS CLI for the caller account. It discards the ARN.
func Current(ctx context.Context) Status {
	return FromProbe(readProbe(ctx))
}

func readProbe(ctx context.Context) Probe {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity", "--output", "json")
	out, err := cmd.Output()
	if err != nil {
		return Probe{Err: err}
	}
	var identity struct {
		Account string `json:"Account"`
		ARN     string `json:"Arn"`
	}
	if err := json.Unmarshal(out, &identity); err != nil {
		return Probe{Err: err}
	}
	// identity.ARN is intentionally unused.
	_ = identity.ARN
	region, _ := configuredRegion(ctx)
	method := "profile"
	if loginSession(ctx) {
		method = "console-login"
	}
	if identity.Account == "" {
		return Probe{Err: errors.New("missing account")}
	}
	return Probe{Account: identity.Account, Region: region, Method: method}
}

func configuredRegion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "aws", "configure", "get", "region")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func loginSession(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "aws", "configure", "list")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	text := string(out)
	return strings.Contains(text, "login") && !strings.Contains(text, "AKIA") && !strings.Contains(text, "ASIA")
}
