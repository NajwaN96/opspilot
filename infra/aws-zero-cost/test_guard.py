import json
import tempfile
import unittest
from pathlib import Path

import guard


def plan_with(types: list[str], extra: str = "") -> dict:
    return {
        "resource_changes": [
            {"type": kind, "change": {"actions": ["create"]}} for kind in types
        ],
        "configuration": {"root_module": {"resources": [{"mode": "data", "type": "aws_caller_identity"}]}},
        "note": extra,
    }


class GuardTest(unittest.TestCase):
    def test_allowlist_accepts_the_portfolio_resources(self):
        reasons = guard.reject_reasons(
            plan_with(
                [
                    "aws_iam_role",
                    "aws_iam_role_policy",
                    "aws_lambda_function",
                    "aws_lambda_function_url",
                    "aws_lambda_permission",
                    "aws_cloudwatch_log_group",
                ]
            )
        )
        self.assertEqual(reasons, [])

    def test_eks_and_s3_are_rejected(self):
        reasons = guard.reject_reasons(plan_with(["aws_lambda_function", "aws_eks_cluster", "aws_s3_bucket"]))
        self.assertTrue(any("aws_eks_cluster" in reason for reason in reasons))
        self.assertTrue(any("aws_s3_bucket" in reason for reason in reasons))

    def test_cli_fails_closed(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "plan.json"
            path.write_text(json.dumps(plan_with(["aws_instance"])))
            self.assertEqual(guard.main(["guard.py", str(path)]), 1)
            path.write_text(json.dumps(plan_with(["aws_lambda_function"])))
            self.assertEqual(guard.main(["guard.py", str(path)]), 0)


if __name__ == "__main__":
    unittest.main()
