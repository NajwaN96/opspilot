import json
import unittest
from pathlib import Path

import catalog
import handler


class CatalogTest(unittest.TestCase):
    def test_reads_are_stable_and_post_is_refused(self):
        status, body = catalog.resolve("GET", "/api/v1/services")
        self.assertEqual(status, 200)
        self.assertEqual(body[0]["id"], "k8s_demo-shop_payment-api")
        status, body = catalog.resolve("POST", "/api/v1/incidents/INC-142/remediations")
        self.assertEqual(status, 403)
        self.assertIn("read-only", body["error"]["message"])

    def test_release_ignores_extra_fields(self):
        path = Path(catalog.__file__).with_name("release.json")
        path.write_text(
            '{"version":"aws-portfolio-demo","gitCommit":"dd63e087d68321df4539889b1920949e4903a941",'
            '"buildTime":"2026-09-30T08:00:00Z","account":"123456789012","path":"/var/task"}',
            encoding="utf-8",
        )
        self.addCleanup(path.unlink)
        status, body = catalog.resolve("GET", "/api/v1/release")
        self.assertEqual(status, 200)
        self.assertEqual(set(body), {"version", "gitCommit", "buildTime", "environment"})
        self.assertEqual(body["environment"], "aws-portfolio-demo")
        self.assertNotIn("account", body)

    def test_platform_catalog_is_read_only(self):
        status, body = catalog.resolve("GET", "/api/v1/platform/catalog")
        self.assertEqual(status, 200)
        names = {item["metadata"]["name"] for item in body["services"]}
        self.assertEqual(names, {"storefront", "checkout-api", "payment-api", "orders-api", "inventory-api"})
        status, body = catalog.resolve("GET", "/api/v1/platform/catalog/payment-api")
        self.assertEqual(status, 200)
        self.assertEqual(body["spec"]["deploymentStrategy"], "canary")
        status, body = catalog.resolve("POST", "/api/v1/platform/golden-path")
        self.assertEqual(status, 403)
        status, body = catalog.resolve("GET", "/api/v1/platform/runbooks/bad-release")
        self.assertEqual(status, 200)
        self.assertIn("allowedRemediation", body)

    def test_catalog_has_no_secret_material(self):
        blob = json.dumps(catalog.resolve("GET", "/api/v1/cloud/status")[1])
        blob += json.dumps(catalog.resolve("GET", "/api/v1/incidents/INC-142")[1])
        for needle in ("AKIA", "ASIA", "kubeconfig", "BEGIN ", "sk-", "postgres://", "OPENAI"):
            self.assertNotIn(needle, blob)

    def test_handler_serves_a_file_and_blocks_traversal(self):
        root = Path(handler.__file__).resolve().parent / "site"
        root.mkdir(exist_ok=True)
        target = root / "index.html"
        target.write_text("<!doctype html><p>AWS — PORTFOLIO DEMO</p>", encoding="utf-8")
        response = handler.handler({"rawPath": "/", "requestContext": {"http": {"method": "GET"}}}, None)
        self.assertEqual(response["statusCode"], 200)
        self.assertTrue(response["isBase64Encoded"])
        denied = handler.handler({"rawPath": "/../handler.py", "requestContext": {"http": {"method": "GET"}}}, None)
        self.assertEqual(denied["statusCode"], 400)
        refused = handler.handler({"rawPath": "/api/v1/demo/reset", "requestContext": {"http": {"method": "POST"}}}, None)
        self.assertEqual(refused["statusCode"], 403)


if __name__ == "__main__":
    unittest.main()
