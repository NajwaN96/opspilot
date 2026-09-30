import json
import tempfile
import unittest
from pathlib import Path

import ci_policy


def write(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")


class CiPolicyTest(unittest.TestCase):
    def test_apply_in_a_workflow_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write(root / ".github" / "workflows" / "ci.yml", "name: ci\n")
            write(root / ".github" / "workflows" / "deploy-aws.yml", "run: tofu apply\n")
            reasons = ci_policy.scan_workflows(root)
            self.assertTrue(any("tofu or terraform apply" in reason for reason in reasons))

    def test_static_keys_and_paid_override_are_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write(root / ".github" / "workflows" / "ci.yml", "name: ci\n")
            write(
                root / ".github" / "workflows" / "deploy-aws.yml",
                "env:\n  AWS_ACCESS_KEY_ID: example\n  OPSPILOT_PAID_CLOUD_OVERRIDE=yes\n  run: aws eks update-kubeconfig\n",
            )
            reasons = ci_policy.scan_workflows(root)
            self.assertTrue(any("static AWS access key" in reason for reason in reasons))
            self.assertTrue(any("paid cloud override" in reason for reason in reasons))
            self.assertTrue(any("EKS command" in reason for reason in reasons))

    def test_zero_cost_stack_rejects_forbidden_resources_and_plan_only_may_describe_eks(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write(root / "infra" / "aws-zero-cost" / "main.tf", 'resource "aws_nat_gateway" "bad" {}\n')
            write(root / "infra" / "terraform" / "main.tf", 'resource "aws_eks_cluster" "this" {}\n')
            self.assertTrue(any("aws_nat_gateway" in reason for reason in ci_policy.scan_zero_cost_stack(root)))
            self.assertTrue(ci_policy.plan_only_eks_present(root))

    def test_iam_policy_is_limited_to_the_portfolio_function(self):
        policy = {
            "Statement": [
                {
                    "Effect": "Allow",
                    "Action": ["lambda:UpdateFunctionCode", "lambda:GetFunction"],
                    "Resource": "arn:aws:lambda:us-east-1:*:function:opspilot-portfolio",
                },
                {"Effect": "Allow", "Action": "freetier:GetAccountPlanState", "Resource": "*"},
            ]
        }
        self.assertEqual(ci_policy.scan_iam_policy(policy), [])
        broad = {
            "Statement": [
                {
                    "Effect": "Allow",
                    "Action": ["lambda:UpdateFunctionCode", "eks:CreateCluster"],
                    "Resource": "*",
                }
            ]
        }
        reasons = ci_policy.scan_iam_policy(broad)
        self.assertTrue(reasons)
        self.assertTrue(any("eks:" in reason for reason in reasons))

    def test_trust_rejects_wildcards(self):
        rendered = ci_policy.render_trust("123456789012", "example-org/opspilot")
        condition = rendered["Statement"][0]["Condition"]["StringEquals"]
        self.assertEqual(condition["token.actions.githubusercontent.com:aud"], "sts.amazonaws.com")
        self.assertIn("repo:example-org/opspilot:environment:aws-portfolio", condition["token.actions.githubusercontent.com:sub"])
        self.assertEqual(condition["token.actions.githubusercontent.com:ref"], "refs/heads/main")
        self.assertNotIn("*", json.dumps(rendered["Statement"][0]["Principal"]))
        with self.assertRaises(ValueError):
            ci_policy.render_trust("123456789012", "*/*")
        with self.assertRaises(ValueError):
            ci_policy.render_trust("123", "example-org/opspilot")

    def test_tracked_state_and_kubeconfig_are_rejected(self):
        found = ci_policy.forbidden_tracked(
            ["README.md", "infra/terraform/terraform.tfstate", "infra/kubeconfig", "keys/id_rsa"]
        )
        self.assertEqual(len(found), 3)

    def test_live_repository_passes(self):
        reasons = ci_policy.scan(ci_policy.repo_root())
        self.assertEqual(reasons, [], reasons)


if __name__ == "__main__":
    unittest.main()
