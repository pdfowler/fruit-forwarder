"""Validate native calendar request boundaries without requesting account access."""

import json
import subprocess
import unittest
from pathlib import Path

BINARY = Path(__file__).resolve().parents[1] / "build/icloud-reminders-eventkit"


class CalendarBoundaryTests(unittest.TestCase):
    def request(self, **overrides):
        payload = {
            "action": "calendar_snapshot", "calendar_ids": [],
            "start": "2026-01-01T00:00:00Z", "end": "2026-02-01T00:00:00Z",
        }
        payload.update(overrides)
        return subprocess.run([str(BINARY)], input=json.dumps(payload),
                              capture_output=True, text=True, timeout=10)

    def test_empty_scope_returns_no_calendars(self):
        result = self.request()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(json.loads(result.stdout), {"calendars": []})

    def test_invalid_windows_fail_before_permission_request(self):
        for values in [
            {"start": "invalid"},
            {"end": "2026-01-01T00:00:00Z"},
            {"end": "2025-01-01T00:00:00Z"},
            {"end": "2028-01-01T00:00:00Z"},
        ]:
            with self.subTest(values=values):
                result = self.request(**values)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("calendar window", result.stderr)

    def test_invalid_scopes_fail_before_permission_request(self):
        for ids in [[""], [" "], ["one", "one"], [str(i) for i in range(101)]]:
            with self.subTest(ids=ids):
                result = self.request(calendar_ids=ids)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("calendar_ids", result.stderr)


if __name__ == "__main__":
    unittest.main()
