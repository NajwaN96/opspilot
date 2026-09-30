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
