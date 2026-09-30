import json
import unittest
import zipfile
from io import BytesIO
from pathlib import Path

import delivery


def sample_zip() -> bytes:
    buffer = BytesIO()
    with zipfile.ZipFile(buffer, "w") as archive:
        archive.writestr("handler.py", "# portfolio handler\n" + ("# pad\n" * 40))
        archive.writestr("catalog.py", "# portfolio catalog\n" + ("# pad\n" * 40))
        archive.writestr("release.json", "{}\n")
    return buffer.getvalue()


class Fake:
    def __init__(self, **overrides):
        self.ctx = {
            "account_id": "000000003851",
            "region": "us-east-1",
            "function_name": "opspilot-portfolio",
            "plan": {"accountPlanType": "FREE", "accountPlanStatus": "ACTIVE"},
            "override": None,
        }
        self.ctx.update(overrides)
        self.current = sample_zip()
        self.updates = []
        self.mode = "ok"

    def context(self):
        return dict(self.ctx)

    def previous_package(self):
        return self.current

    def update_package(self, payload):
        self.updates.append(payload)
        self.current = payload

    def wait_until_updated(self):
        return None

    def request(self, method, path):
        if self.mode == "bad-smoke" and len(self.updates) == 1 and path == "/health":
            return 500, "unavailable"
        if self.mode == "mutation-open" and method == "POST":
            return 200, "{}"
        if method == "POST":
            return 403, json.dumps({"error": {"code": "read_only"}})
        if path == "/api/v1/cloud/status":
            return 200, json.dumps({"cloudDeployment": "AWS — PORTFOLIO DEMO"})
        if path == "/api/v1/release":
            return 200, json.dumps(
                {
                    "version": "aws-portfolio-demo",
                    "gitCommit": "unknown",
                    "buildTime": "unknown",
                    "environment": "aws-portfolio-demo",
                }
            )
        return 200, "ok"


class DeliveryTest(unittest.TestCase):
    def test_context_rejects_wrong_account_region_function_and_plan(self):
        base = {
            "account_id": "000000003851",
            "region": "us-east-1",
            "function_name": "opspilot-portfolio",
            "plan": {"accountPlanType": "FREE", "accountPlanStatus": "ACTIVE"},
            "override": None,
        }
        self.assertEqual(delivery.check_context(**base), [])
        self.assertIn("wrong account", delivery.check_context(**{**base, "account_id": "000000003852"}))
        self.assertIn("wrong region", delivery.check_context(**{**base, "region": "eu-west-1"}))
        self.assertIn("wrong Lambda name", delivery.check_context(**{**base, "function_name": "other"}))
        self.assertIn("account plan is not FREE and ACTIVE", delivery.check_context(**{**base, "plan": {"accountPlanType": "PAID", "accountPlanStatus": "ACTIVE"}}))
        self.assertIn("paid override is set", delivery.check_context(**{**base, "override": "yes"}))

    def test_bad_artifact_is_rejected_before_update(self):
        client = Fake()
        with self.assertRaises(delivery.DeliveryError):
            delivery.publish_artifact(client, b"not-a-zip")
        self.assertEqual(client.updates, [])

    def test_failed_smoke_restores_the_previous_package(self):
        client = Fake()
        previous = client.current
        client.mode = "bad-smoke"
        fresh = sample_zip()
        with self.assertRaises(delivery.DeliveryError) as caught:
            delivery.publish_artifact(client, fresh)
        self.assertIn("previous package restored", str(caught.exception))
        self.assertEqual(client.updates, [fresh, previous])
        self.assertEqual(client.current, previous)

    def test_open_mutation_fails_verification_and_rolls_back(self):
        client = Fake()
        previous = client.current
        client.mode = "mutation-open"
        with self.assertRaises(delivery.DeliveryError):
            delivery.publish_artifact(client, sample_zip())
        self.assertEqual(client.current, previous)

    def test_wrong_account_does_not_update(self):
        client = Fake(account_id="111111111111")
        with self.assertRaises(delivery.DeliveryError):
            delivery.publish_artifact(client, sample_zip())
        self.assertEqual(client.updates, [])

    def test_successful_publish_updates_once(self):
        client = Fake()
        delivery.publish_artifact(client, sample_zip())
        self.assertEqual(len(client.updates), 1)

    def test_previous_package_without_release_metadata_can_be_replaced(self):
        client = Fake()
        buffer = BytesIO()
        with zipfile.ZipFile(buffer, "w") as archive:
            archive.writestr("handler.py", "# portfolio handler\n" + ("# pad\n" * 40))
            archive.writestr("catalog.py", "# portfolio catalog\n" + ("# pad\n" * 40))
        client.current = buffer.getvalue()
        delivery.publish_artifact(client, sample_zip())
        self.assertEqual(len(client.updates), 1)

    def test_new_artifact_must_include_release_metadata(self):
        client = Fake()
        buffer = BytesIO()
        with zipfile.ZipFile(buffer, "w") as archive:
            archive.writestr("handler.py", "# portfolio handler\n" + ("# pad\n" * 40))
            archive.writestr("catalog.py", "# portfolio catalog\n" + ("# pad\n" * 40))
        with self.assertRaises(delivery.DeliveryError) as caught:
            delivery.publish_artifact(client, buffer.getvalue())
        self.assertIn("artifact is missing release.json", str(caught.exception))
        self.assertEqual(client.updates, [])

    def test_publisher_does_not_publish_versions_or_apply_tofu(self):
        text = Path(delivery.__file__).read_text(encoding="utf-8")
        self.assertNotIn("publish-version", text)
        self.assertNotIn("tofu apply", text.lower())
        self.assertNotIn("terraform apply", text.lower())


if __name__ == "__main__":
    unittest.main()
