"""Tests for doc-generator.py annotation parsing. Run: python -m unittest tools/test_doc_generator.py"""

import importlib.util
import tempfile
import textwrap
import unittest
from pathlib import Path

_spec = importlib.util.spec_from_file_location("doc_generator", Path(__file__).with_name("doc-generator.py"))
doc_generator = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(doc_generator)


class AnnotationParsingTest(unittest.TestCase):
    def describe(self, values: str, *path):
        with tempfile.NamedTemporaryFile("w", suffix=".yaml", delete=False) as f:
            f.write(textwrap.dedent(values))
        self.addCleanup(Path(f.name).unlink)
        _, comments = doc_generator.YAMLLoader(Path(f.name)).load()
        return doc_generator.DocumentationGenerator(check_mode=True).get_field_description(comments, *path)

    def test_single_line(self):
        got = self.describe("""
            stack:
              el:
                # @desc: One line
                enabled: false
            """, "stack", "el", "enabled")
        self.assertEqual(got, "One line")

    def test_consecutive_lines_are_joined(self):
        got = self.describe("""
            stack:
              el:
                config:
                  # @desc: Allow OSDs to be unevenly distributed. Set true on bare metal when the
                  # @desc: count is not a multiple of three.
                  flexibleScaling: null
            """, "stack", "el", "config", "flexibleScaling")
        self.assertEqual(got, "Allow OSDs to be unevenly distributed. Set true on bare metal when the "
                              "count is not a multiple of three.")

    def test_desc_and_description_prefixes_join(self):
        got = self.describe("""
            stack:
              el:
                # @description: First part,
                # @desc: second part
                toggles: {}
            """, "stack", "el", "toggles")
        self.assertEqual(got, "First part, second part")

    def test_plain_comment_starts_a_new_run(self):
        got = self.describe("""
            stack:
              el:
                config:
                  # @desc: Stale annotation
                  # Maintainer note, not rendered.
                  # @desc: Current annotation
                  key: value
            """, "stack", "el", "config", "key")
        self.assertEqual(got, "Current annotation")

    def test_blank_line_starts_a_new_run(self):
        got = self.describe("""
            stack:
              el:
                # @desc: Stale

                # @desc: Current
                enabled: true
            """, "stack", "el", "enabled")
        self.assertEqual(got, "Current")

    def test_multi_line_on_list_item_by_name(self):
        got = self.describe("""
            stack:
              el:
                policies:
                  # @description: Waits for storage nodes.
                  # @description: Requires the storage-nodes element.
                  - name: cluster
                    enabled: true
            """, "stack", "el", "policies", "cluster")
        self.assertEqual(got, "Waits for storage nodes. Requires the storage-nodes element.")

    def test_multi_line_on_key_inside_list_item(self):
        got = self.describe("""
            stack:
              el:
                policies:
                  - name: ready
                    # @desc: Other elements depend on this policy.
                    # @desc: Nothing depends on health.
                    remediationAction: inform
            """, "stack", "el", "policies", 0, "remediationAction")
        self.assertEqual(got, "Other elements depend on this policy. Nothing depends on health.")

    def test_annotation_does_not_leak_to_next_key(self):
        values = """
            stack:
              el:
                # @desc: Documented
                documented: 1
                undocumented: 2
            """
        self.assertEqual(self.describe(values, "stack", "el", "documented"), "Documented")
        self.assertIsNone(self.describe(values, "stack", "el", "undocumented"))


class WriteIfChangedTest(unittest.TestCase):
    OLD = "# el\n\n*Generated: 2026-01-01 00:00:00*\n\nbody\n"

    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.path = Path(tmp.name) / "el.md"
        self.gen = doc_generator.DocumentationGenerator(check_mode=True)

    def test_writes_missing_file(self):
        self.assertTrue(self.gen.write_if_changed(self.path, self.OLD))
        self.assertEqual(self.path.read_text(), self.OLD)

    def test_skips_timestamp_only_change(self):
        self.path.write_text(self.OLD)
        new = self.OLD.replace("2026-01-01 00:00:00", "2026-09-25 12:34:56")
        self.assertFalse(self.gen.write_if_changed(self.path, new))
        self.assertEqual(self.path.read_text(), self.OLD)

    def test_writes_content_change(self):
        self.path.write_text(self.OLD)
        new = self.OLD.replace("body", "changed body")
        self.assertTrue(self.gen.write_if_changed(self.path, new))
        self.assertEqual(self.path.read_text(), new)


if __name__ == "__main__":
    unittest.main()
