import unittest

import release_meta


class ReleaseMetaTest(unittest.TestCase):
    def test_public_labels_drop_unsafe_values(self):
        body = release_meta.public_labels("arn:aws:iam::123456789012:role/x", "/var/task", "bad version")
        self.assertEqual(
            body,
            {
                "version": "aws-portfolio-demo",
                "gitCommit": "unknown",
                "buildTime": "unknown",
                "environment": "aws-portfolio-demo",
            },
        )
        self.assertNotIn("account", body)

    def test_artifact_record_keeps_checksum_and_drops_account(self):
        public = release_meta.public_labels("dd63e087d68321df4539889b1920949e4903a941", "2026-09-30T08:00:00Z")
        record = release_meta.artifact_record(public, "ab" * 32, "4242")
        self.assertEqual(record["artifactSha256"], "ab" * 32)
        self.assertEqual(record["workflowRun"], "4242")
        self.assertEqual(record["functionName"], "opspilot-portfolio")
        self.assertNotIn("account", record)
        self.assertNotIn("path", record)

    def test_artifact_record_rejects_a_short_checksum(self):
        public = release_meta.public_labels("unknown", "unknown")
        with self.assertRaises(ValueError):
            release_meta.artifact_record(public, "abcd", "local")


if __name__ == "__main__":
    unittest.main()
