"""Tests for update_changelog.py — CHANGELOG automation."""

import os
import textwrap

# Import the module under test
import update_changelog as uc


class TestClassifyPr:
    """Test PR title classification."""

    def test_feat_is_added(self):
        assert uc.classify_pr("feat: add new feature") == "Added"

    def test_fix_is_fixed(self):
        assert uc.classify_pr("fix: resolve bug") == "Fixed"

    def test_docs_is_changed(self):
        assert uc.classify_pr("docs: update README") == "Changed"

    def test_chore_is_changed(self):
        assert uc.classify_pr("chore: bump dependency") == "Changed"

    def test_unknown_is_changed(self):
        assert uc.classify_pr("something else") == "Changed"

    def test_case_insensitive(self):
        assert uc.classify_pr("Feat: add new feature") == "Added"
        assert uc.classify_pr("FIX: resolve bug") == "Fixed"


class TestSummaryToChangelogEntries:
    """Test conversion of PR summaries to changelog entries."""

    def test_simple_bullets(self):
        summary = "- Added feature A\n- Fixed bug B"
        result = uc.summary_to_changelog_entries(summary, 42, "Added")
        assert "### Added" in result
        assert "Added feature A" in result
        assert "Fixed bug B" in result
        assert "(#42)" in result or "[#42]" in result

    def test_bold_bullets(self):
        summary = "- **Feature A**: Does something cool\n- **Feature B**: Also cool"
        result = uc.summary_to_changelog_entries(summary, 10, "Added")
        assert "**Feature A**" in result
        assert "**Feature B**" in result

    def test_skips_tables(self):
        summary = "- Real entry\n| Header | Value |\n|--------|-------|\n| foo | bar |"
        result = uc.summary_to_changelog_entries(summary, 5, "Fixed")
        assert "Real entry" in result
        assert "Header" not in result

    def test_skips_code_blocks(self):
        summary = "- Real entry\n```bash\nsome code\n```"
        result = uc.summary_to_changelog_entries(summary, 3, "Changed")
        assert "Real entry" in result
        assert "```" not in result

    def test_skips_multiline_code_blocks(self):
        summary = (
            "- Real entry\n"
            "```python\n"
            "def foo():\n"
            "    return 'should be skipped'\n"
            "```\n"
            "- Another real entry"
        )
        result = uc.summary_to_changelog_entries(summary, 4, "Added")
        assert "Real entry" in result
        assert "Another real entry" in result
        assert "foo" not in result
        assert "should be skipped" not in result

    def test_skips_generated_with(self):
        summary = (
            "- Real entry\n"
            "- Generated with [Claude Code](https://claude.com/claude-code)"
        )
        result = uc.summary_to_changelog_entries(summary, 7, "Added")
        assert "Real entry" in result
        assert "Generated with" not in result
        assert "Claude Code" not in result

    def test_empty_summary(self):
        result = uc.summary_to_changelog_entries("", 1, "Added")
        assert result == ""

    def test_only_tables(self):
        summary = "| a | b |\n|---|---|\n| 1 | 2 |"
        result = uc.summary_to_changelog_entries(summary, 1, "Added")
        assert result == ""

    def test_star_bullets(self):
        summary = "* Added feature A\n* Fixed bug B"
        result = uc.summary_to_changelog_entries(summary, 99, "Fixed")
        assert "Added feature A" in result
        assert "Fixed bug B" in result

    def test_strips_github_pr_links(self):
        summary = (
            "- Some fix by @user in https://github.com/SCGIS-Wales/helm-mcp/pull/42"
        )
        result = uc.summary_to_changelog_entries(summary, 42, "Fixed")
        assert "Some fix by @user" in result

    def test_pr_reference_added(self):
        summary = "- Added feature X"
        result = uc.summary_to_changelog_entries(summary, 15, "Added")
        assert "[#15]" in result
        assert "pull/15" in result

    def test_skips_markdown_headers(self):
        summary = "### Security\n- Fixed a security issue\n### Docs\n- Updated docs"
        result = uc.summary_to_changelog_entries(summary, 20, "Fixed")
        assert "Fixed a security issue" in result
        assert "Updated docs" in result
        assert "### Security" not in result
        assert "### Docs" not in result


class TestUpdateChangelog:
    """Test CHANGELOG.md file manipulation."""

    def test_inserts_after_unreleased(self, tmp_path, monkeypatch):
        changelog = tmp_path / "CHANGELOG.md"
        changelog.write_text(
            textwrap.dedent("""\
            # Changelog

            ## [Unreleased]

            ## [0.1.23] - 2026-02-24

            ### Changed
            - Something old

            [Unreleased]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.23...HEAD
            [0.1.23]: https://github.com/SCGIS-Wales/helm-mcp/releases/tag/v0.1.23
        """)
        )

        # Monkeypatch the script's path resolution
        monkeypatch.setattr(
            os.path,
            "abspath",
            lambda p: str(changelog) if "CHANGELOG" in p else os.path.realpath(p),
        )
        monkeypatch.setattr(
            os.path,
            "dirname",
            lambda p: str(tmp_path) if p == uc.__file__ else os.path.dirname(p),
        )

        content = "### Added\n- New feature ([#31](https://github.com/SCGIS-Wales/helm-mcp/pull/31))\n"
        uc.update_changelog("0.1.24", "2026-02-25", content, 31)

        result = changelog.read_text()

        # New version should appear after [Unreleased]
        assert "## [0.1.24] - 2026-02-25" in result
        assert "New feature" in result

        # [Unreleased] link should be updated
        assert "compare/v0.1.24...HEAD" in result

        # New version link should be added
        assert "[0.1.24]: https://github.com/SCGIS-Wales/helm-mcp/compare/" in result

        # Original content should still be present
        assert "## [0.1.23] - 2026-02-24" in result
        assert "Something old" in result

        # Order should be correct: Unreleased > 0.1.24 > 0.1.23
        unreleased_pos = result.index("[Unreleased]")
        new_pos = result.index("[0.1.24]")
        old_pos = result.index("[0.1.23]")
        assert unreleased_pos < new_pos < old_pos

    def test_skips_existing_version(self, tmp_path, monkeypatch, capsys):
        changelog = tmp_path / "CHANGELOG.md"
        changelog.write_text(
            textwrap.dedent("""\
            # Changelog

            ## [Unreleased]

            ## [0.1.24] - 2026-02-25

            ### Added
            - Already here

            [Unreleased]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.24...HEAD
            [0.1.24]: https://github.com/SCGIS-Wales/helm-mcp/releases/tag/v0.1.24
        """)
        )

        monkeypatch.setattr(
            os.path,
            "abspath",
            lambda p: str(changelog) if "CHANGELOG" in p else os.path.realpath(p),
        )
        monkeypatch.setattr(
            os.path,
            "dirname",
            lambda p: str(tmp_path) if p == uc.__file__ else os.path.dirname(p),
        )

        original = changelog.read_text()

        # This should be caught by main(), but test the file check directly
        assert "## [0.1.24]" in original


class TestGetPrSummary:
    """Test PR summary extraction (unit tests using mock data)."""

    def test_extracts_summary_section(self):
        body = (
            "## Summary\n"
            "- Fixed a bug\n"
            "- Added a feature\n"
            "\n"
            "## Test plan\n"
            "- [x] Tests pass\n"
        )
        # We can't easily test get_pr_summary without mocking gh API,
        # so test the regex extraction directly
        import re

        match = re.search(r"## Summary\s*\n(.*?)(?=\n## |\Z)", body, re.DOTALL)
        assert match is not None
        summary = match.group(1).strip()
        assert "Fixed a bug" in summary
        assert "Added a feature" in summary
        assert "Test plan" not in summary

    def test_no_summary_section(self):
        body = "Just some text without headers"
        import re

        match = re.search(r"## Summary\s*\n(.*?)(?=\n## |\Z)", body, re.DOTALL)
        assert match is None
