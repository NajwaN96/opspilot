import json
import unittest
from pathlib import Path

import validate


class ContractTest(unittest.TestCase):
    def test_live_catalog_is_valid(self):
        self.assertEqual(validate.validate_tree(), [])

    def test_missing_owner_is_rejected(self):
        service = {
            "apiVersion": "opspilot.io/v1",
            "kind": "Service",
            "metadata": {"name": "example"},
            "spec": {
                "description": "x",
                "team": "x",
                "runtime": "Go",
                "language": "Go",
                "repository": "gitops/base/demo-shop.yaml",
                "environment": "local-live",
                "namespace": "demo-shop",
                "version": "1",
                "slo": {"availability": 99.9, "latencyMs": 300},
                "dependencies": [],
                "runbook": "bad-release",
                "dashboard": "local",
                "onCall": "none",
                "deploymentStrategy": "rolling",
                "checks": {key: "PASS" for key in validate.CHECKS},
            },
        }
        reasons = validate.validate_service(service, {"bad-release"})
        self.assertTrue(any("owner" in reason for reason in reasons))

    def test_unknown_strategy_is_rejected(self):
        raw = json.loads((validate.repo_root() / "platform" / "catalog" / "catalog.json").read_text(encoding="utf-8"))
        raw["services"][0]["spec"]["deploymentStrategy"] = "blue-green"
        reasons = validate.validate_service(raw["services"][0], {"dependency-failure", "payment-api-high-error-rate"})
        self.assertTrue(any("deploymentStrategy" in reason for reason in reasons))

    def test_templates_exist(self):
        root = validate.repo_root() / "templates" / "services"
        required = {
            "README.md",
            "Dockerfile",
            "kustomization.yaml",
            "deploy/deployment.yaml",
            "deploy/service.yaml",
            "slo.yaml",
            "runbook.md",
            "service.yaml",
            "ci.yml",
        }
        for runtime in ("go", "node", "python"):
            base = root / runtime
            names = {path.relative_to(base).as_posix() for path in base.rglob("*") if path.is_file()}
            self.assertTrue(required <= names, runtime)
            text = (base / "deploy" / "deployment.yaml").read_text(encoding="utf-8")
            self.assertIn("runAsNonRoot: true", text)
            self.assertIn("__SERVICE_NAME__", text)


if __name__ == "__main__":
    unittest.main()
