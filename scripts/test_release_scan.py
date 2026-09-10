import json
import unittest

from check_release import inspect


class ReleaseScanTests(unittest.TestCase):
    def test_reports_private_paths_without_echoing_values(self):
        private = "/" + "Users" + "/private-person/code"
        result = inspect("fixture.txt", private)
        self.assertTrue(any(hit["rule"] == "personal-home-path" for hit in result))
        self.assertNotIn("private-person", json.dumps(result))

    def test_allows_reserved_example_emails_only(self):
        self.assertFalse(inspect("fixture.txt", "synthetic@example.test"))
        self.assertTrue(inspect("fixture.txt", "person" + "@" + "private.invalid"))

    def test_preserves_license_attribution(self):
        email = "maintainer" + "@" + "upstream.invalid"
        self.assertFalse(inspect("THIRD_PARTY_NOTICES.md", email))
        self.assertTrue(inspect("test.go", email))


if __name__ == "__main__":
    unittest.main()
