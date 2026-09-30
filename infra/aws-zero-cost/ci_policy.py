"""CI policy for the two AWS stacks.

infra/terraform may describe EKS. That stack is plan-only.
infra/aws-zero-cost is the only deployable stack, and CI may not apply either one.
GitHub Actions may update the existing portfolio Lambda. It may not run tofu apply.
"""

from __future__ import annotations

import json
import re
import subprocess
import sys
from pathlib import Path

APPLY_RE = re.compile(r"\b(?:tofu|terraform)\s+apply\b")
KEY_RE = re.compile(r"\bAWS_(?:ACCESS_KEY_ID|SECRET_ACCESS_KEY)\b")
OVERRIDE_RE = re.compile(r"OPSPILOT_PAID_CLOUD_OVERRIDE\s*[:=]\s*['\"]?yes")
EKS_CMD_RE = re.compile(r"\b(?:aws\s+eks|eksctl)\b")
REPO_RE = re.compile(r"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$")
LAMBDA_RESOURCE_RE = re.compile(r"^arn:aws:lambda:us-east-1:(?:\*|\d{12}):function:opspilot-portfolio$")

LAMBDA_ACTIONS = frozenset(
    {
        "lambda:GetFunction",
        "lambda:GetFunctionConfiguration",
        "lambda:UpdateFunctionCode",
        "lambda:GetFunctionUrlConfig",
    }
)

FORBIDDEN_TF = (
    "aws_eks_",
    "aws_instance",
    "aws_nat_gateway",
    "aws_lb",
    "aws_alb",
    "aws_db_instance",
    "aws_opensearch",
    "aws_elasticache",
    "aws_msk_",
    "aws_eip",
    "aws_s3_bucket",
    "aws_cloudfront",
    "aws_api_gateway",
    "aws_apigateway",
    "aws_dynamodb",
    "aws_amplify",
    "aws_ecs_",
    "aws_ecr_",
    "map_public_ip_on_launch",
)


def repo_root() -> Path:
    return Path(__file__).resolve().parents[2]


def scan_workflows(root: Path) -> list[str]:
    workflow_dir = root / ".github" / "workflows"
    files = sorted(workflow_dir.glob("*.yml")) + sorted(workflow_dir.glob("*.yaml"))
    reasons = []
    if not files:
        return ["no GitHub Actions workflows"]
    names = {path.name for path in files}
    if "ci.yml" not in names or "deploy-aws.yml" not in names:
        reasons.append("ci.yml and deploy-aws.yml are both required")
    for path in files:
        text = path.read_text(encoding="utf-8")
        if APPLY_RE.search(text):
            reasons.append(f"{path.name} contains tofu or terraform apply")
        if KEY_RE.search(text):
            reasons.append(f"{path.name} references a static AWS access key")
        if OVERRIDE_RE.search(text):
            reasons.append(f"{path.name} sets the paid cloud override")
        if EKS_CMD_RE.search(text):
            reasons.append(f"{path.name} invokes an EKS command")
        if "AdministratorAccess" in text:
            reasons.append(f"{path.name} grants AdministratorAccess")
    deploy = workflow_dir / "deploy-aws.yml"
    if deploy.is_file():
        text = deploy.read_text(encoding="utf-8")
        if "opspilot-aws-production" not in text:
            reasons.append("deploy workflow is missing the production concurrency group")
        if "id-token: write" not in text:
            reasons.append("deploy workflow does not request an OIDC token")
        if "aws-portfolio" not in text:
            reasons.append("deploy workflow is missing the aws-portfolio environment")
        if "role-to-assume" not in text:
            reasons.append("deploy workflow does not assume an IAM role")
    return reasons


def scan_zero_cost_stack(root: Path) -> list[str]:
    stack = root / "infra" / "aws-zero-cost"
    reasons = []
    for path in sorted(stack.rglob("*.tf")):
        if ".terraform" in path.parts:
            continue
        text = path.read_text(encoding="utf-8")
        for marker in FORBIDDEN_TF:
            if marker in text:
                reasons.append(f"{path.relative_to(root)} contains {marker}")
    return reasons


def plan_only_eks_present(root: Path) -> bool:
    stack = root / "infra" / "terraform"
    for path in stack.rglob("*.tf"):
        if ".terraform" in path.parts:
            continue
        if 'resource "aws_eks_cluster"' in path.read_text(encoding="utf-8"):
            return True
    return False


def forbidden_tracked(paths: list[str]) -> list[str]:
    reasons = []
    for path in paths:
        lowered = path.lower()
        name = Path(path).name.lower()
        if lowered.endswith(".tfstate") or ".tfstate." in lowered:
            reasons.append(path)
        elif name == "kubeconfig" or name.endswith(".kubeconfig") or "/kubeconfig" in lowered:
            reasons.append(path)
        elif name.endswith(".pem") or name in {"id_rsa", "id_dsa", "id_ecdsa", "id_ed25519"}:
            reasons.append(path)
    return reasons


def tracked_files(root: Path) -> list[str]:
    completed = subprocess.run(
        ["git", "ls-files"],
        cwd=root,
        check=True,
        capture_output=True,
        text=True,
    )
    return [line for line in completed.stdout.splitlines() if line]


def scan_iam_policy(policy: dict) -> list[str]:
    reasons = []
    statements = policy.get("Statement") or []
    if not statements:
        return ["deployment policy has no statements"]
    for statement in statements:
        if statement.get("Effect") != "Allow":
            reasons.append("deployment policy contains a non-allow statement")
            continue
        actions = statement.get("Action") or []
        if isinstance(actions, str):
            actions = [actions]
        resources = statement.get("Resource") or []
        if isinstance(resources, str):
            resources = [resources]
        action_set = set(actions)
        if action_set == {"freetier:GetAccountPlanState"}:
            if resources != ["*"]:
                reasons.append("free-plan read must be the only use of Resource *")
            continue
        if not action_set <= LAMBDA_ACTIONS:
            reasons.append("deployment policy grants actions outside the portfolio Lambda update set")
            continue
        if not resources or any(not LAMBDA_RESOURCE_RE.fullmatch(item) for item in resources):
            reasons.append("Lambda permissions are not limited to opspilot-portfolio")
    blob = json.dumps(policy)
    for needle in ("eks:", "ec2:", "rds:", "s3:", "cloudformation:", "iam:", "organizations:", "elasticloadbalancing:", "AdministratorAccess"):
        if needle in blob:
            reasons.append(f"deployment policy contains {needle}")
    return reasons


IMMUTABLE_PREFIX_RE = re.compile(r"repo:[A-Za-z0-9_.-]+@[0-9]+/[A-Za-z0-9_.-]+@[0-9]+")


def render_trust(account_id: str, repository: str, subject_prefix: str | None = None) -> dict:
    if not re.fullmatch(r"\d{12}", account_id or ""):
        raise ValueError("account id must be 12 digits")
    if not REPO_RE.fullmatch(repository or "") or "*" in repository or ".." in repository.split("/"):
        raise ValueError("repository must be owner/name with no wildcard")
    if subject_prefix:
        if not IMMUTABLE_PREFIX_RE.fullmatch(subject_prefix) or "*" in subject_prefix:
            raise ValueError("immutable subject prefix is not a single repository")
        base = subject_prefix
    else:
        base = f"repo:{repository}"
    return {
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Principal": {
                    "Federated": f"arn:aws:iam::{account_id}:oidc-provider/token.actions.githubusercontent.com"
                },
                "Action": "sts:AssumeRoleWithWebIdentity",
                "Condition": {
                    "StringEquals": {
                        "token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
                        "token.actions.githubusercontent.com:sub": f"{base}:environment:aws-portfolio",
                        "token.actions.githubusercontent.com:ref": "refs/heads/main",
                    }
                },
            }
        ],
    }


def scan(root: Path | None = None) -> list[str]:
    root = root or repo_root()
    reasons = []
    reasons.extend(scan_workflows(root))
    reasons.extend(scan_zero_cost_stack(root))
    if not plan_only_eks_present(root):
        reasons.append("plan-only EKS architecture is missing from infra/terraform")
    policy_path = root / "infra" / "aws-zero-cost" / "iam" / "github-deployer-policy.json"
    if not policy_path.is_file():
        reasons.append("github deployer policy is missing")
    else:
        reasons.extend(scan_iam_policy(json.loads(policy_path.read_text(encoding="utf-8"))))
    trust_path = root / "infra" / "aws-zero-cost" / "iam" / "github-deployer-trust.template.json"
    if not trust_path.is_file():
        reasons.append("github deployer trust template is missing")
    else:
        trust = trust_path.read_text(encoding="utf-8")
        if '"Principal": "*"' in trust or '"AWS": "*"' in trust or "__GITHUB_SUBJECT_PREFIX__" not in trust:
            reasons.append("trust template is not restricted to one repository placeholder")
        if "StringLike" in trust:
            reasons.append("trust template must use StringEquals")
    if (root / ".git").exists():
        reasons.extend(forbidden_tracked(tracked_files(root)))
    return reasons


def main(argv: list[str]) -> int:
    root = Path(argv[1]).resolve() if len(argv) == 2 else repo_root()
    reasons = scan(root)
    if reasons:
        print("BLOCKED BY CI POLICY", file=sys.stderr)
        for reason in reasons:
            print(reason, file=sys.stderr)
        return 1
    print("ci policy accepted")
    print("plan-only: infra/terraform")
    print("deployable: infra/aws-zero-cost Lambda code only")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
