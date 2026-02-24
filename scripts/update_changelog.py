#!/usr/bin/env python3
"""Update CHANGELOG.md with PR descriptions from the latest release.

Called by CI after auto-tag to inject the merged PR's summary section
into the CHANGELOG under the new version heading.

Usage:
    python scripts/update_changelog.py <version> <repo>

    version: semver string like "0.1.24" (without 'v' prefix)
    repo:    GitHub repo in "owner/repo" format

Environment:
    GH_TOKEN or GITHUB_TOKEN must be set for GitHub API access.

The script:
1. Finds the merge commit for the new tag
2. Extracts the PR number from the merge commit
3. Fetches the PR's ## Summary section via GitHub API
4. Converts the summary into Keep a Changelog entries
5. Inserts a new version section into CHANGELOG.md
6. Updates the comparison links at the bottom
"""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
from datetime import datetime, timezone


def run(cmd: list[str], **kwargs) -> str:
    """Run a command and return stripped stdout."""
    result = subprocess.run(cmd, capture_output=True, text=True, **kwargs)
    if result.returncode != 0:
        print(f"Command failed: {' '.join(cmd)}", file=sys.stderr)
        print(f"stderr: {result.stderr}", file=sys.stderr)
        sys.exit(1)
    return result.stdout.strip()


def gh_api(endpoint: str, repo: str) -> dict | list:
    """Call the GitHub REST API via gh CLI."""
    raw = run(["gh", "api", f"repos/{repo}/{endpoint}"])
    return json.loads(raw)


def find_pr_for_tag(tag: str, repo: str) -> int | None:
    """Find the PR number associated with a tag's merge commit.

    Strategy: look at the commits between the previous tag and this tag,
    find the squash-merge commit, and extract the PR number from its
    message (GitHub appends ' (#N)' to squash merge titles).
    """
    # Get all tags sorted by version
    tags_raw = run(
        ["git", "tag", "--list", "v[0-9]*.[0-9]*.[0-9]*", "--sort=-v:refname"]
    )
    tags = tags_raw.split("\n")

    tag_index = tags.index(tag) if tag in tags else -1
    if tag_index < 0:
        print(f"Tag {tag} not found in git tags", file=sys.stderr)
        return None

    # Previous tag is the next one in descending order
    prev_tag = tags[tag_index + 1] if tag_index + 1 < len(tags) else None

    if prev_tag:
        log_range = f"{prev_tag}..{tag}"
    else:
        log_range = tag

    # Get commit messages between tags (excluding [skip ci] version bumps)
    log = run(
        [
            "git",
            "log",
            log_range,
            "--oneline",
            "--no-merges",
            "--grep=skip ci",
            "--invert-grep",
        ]
    )

    if not log:
        return None

    # Look for PR number in commit messages: "some title (#123)"
    for line in log.split("\n"):
        match = re.search(r"\(#(\d+)\)", line)
        if match:
            return int(match.group(1))

    return None


def get_pr_summary(pr_number: int, repo: str) -> str:
    """Fetch a PR's body and extract the ## Summary section."""
    pr_data = gh_api(f"pulls/{pr_number}", repo)
    body = pr_data.get("body", "") or ""

    # Extract the Summary section
    match = re.search(r"## Summary\s*\n(.*?)(?=\n## |\Z)", body, re.DOTALL)
    if match:
        return match.group(1).strip()

    # Fall back to everything before ## Test plan
    parts = body.split("## Test")
    if parts:
        return parts[0].strip()

    return body.strip()


def get_pr_title(pr_number: int, repo: str) -> str:
    """Fetch a PR's title."""
    pr_data = gh_api(f"pulls/{pr_number}", repo)
    return pr_data.get("title", "")


def classify_pr(title: str) -> str:
    """Classify a PR title into a changelog category."""
    title_lower = title.lower()
    if title_lower.startswith("feat"):
        return "Added"
    elif title_lower.startswith("fix"):
        return "Fixed"
    elif title_lower.startswith("docs"):
        return "Changed"
    elif title_lower.startswith("chore") or title_lower.startswith("style"):
        return "Changed"
    elif title_lower.startswith("refactor"):
        return "Changed"
    elif title_lower.startswith("perf"):
        return "Changed"
    else:
        return "Changed"


def summary_to_changelog_entries(summary: str, pr_number: int, category: str) -> str:
    """Convert a PR summary into changelog entries.

    Extracts bullet points from the summary, cleans them up,
    and groups them under the appropriate category heading.
    """
    lines = summary.split("\n")
    entries = []
    in_code_block = False

    for line in lines:
        line = line.strip()
        if not line:
            continue
        # Track code blocks (``` fenced blocks)
        if line.startswith("```"):
            in_code_block = not in_code_block
            continue
        if in_code_block:
            continue
        # Skip markdown headers, tables
        if line.startswith("#") or line.startswith("|"):
            continue
        # Skip lines that are just links or images
        if line.startswith("![") or (
            line.startswith("[") and "](" in line and line.endswith(")")
        ):
            continue
        # Clean up bullet points
        if line.startswith("- "):
            entry = line[2:].strip()
        elif line.startswith("* "):
            entry = line[2:].strip()
        else:
            # Non-bullet line — use as-is if it's meaningful content
            if len(line) > 10 and not line.startswith("###"):
                entry = line
            else:
                continue

        # Remove trailing PR links that GitHub adds
        entry = re.sub(r"\s*\(https://github\.com/[^)]+\)\s*$", "", entry)
        # Remove emoji prefixes
        entry = re.sub(r"^[\U0001F300-\U0001F9FF]\s*", "", entry)
        # Remove "Generated with Claude Code" lines
        if "Generated with" in entry or "Claude Code" in entry:
            continue

        if entry:
            entries.append(entry)

    if not entries:
        return ""

    # Build the changelog section
    ref = f"[#{pr_number}](https://github.com/SCGIS-Wales/helm-mcp/pull/{pr_number})"
    result = f"### {category}\n"
    for entry in entries:
        # Add PR reference if not already present
        if f"#{pr_number}" not in entry:
            result += f"- {entry} ({ref})\n"
        else:
            result += f"- {entry}\n"

    return result


def update_changelog(version: str, date: str, content: str, pr_number: int) -> None:
    """Insert a new version section into CHANGELOG.md."""
    changelog_path = os.path.join(os.path.dirname(__file__), "..", "CHANGELOG.md")
    changelog_path = os.path.abspath(changelog_path)

    with open(changelog_path) as f:
        text = f.read()

    # Build the new section
    new_section = f"## [{version}] - {date}\n\n{content}\n"

    # Insert after [Unreleased] section
    unreleased_pattern = r"(## \[Unreleased\]\s*\n)"
    match = re.search(unreleased_pattern, text)
    if match:
        insert_pos = match.end()
        text = text[:insert_pos] + "\n" + new_section + text[insert_pos:]
    else:
        # Fallback: insert after the header
        header_end = text.find("\n## ")
        if header_end > 0:
            text = text[:header_end] + "\n\n" + new_section + text[header_end:]

    # Update the [Unreleased] comparison link
    text = re.sub(
        r"\[Unreleased\]: https://github\.com/SCGIS-Wales/helm-mcp/compare/v[\d.]+\.\.\.HEAD",
        f"[Unreleased]: https://github.com/SCGIS-Wales/helm-mcp/compare/v{version}...HEAD",
        text,
    )

    # Add the new version comparison link if not present
    version_link_prefix = f"[{version}]:"
    if version_link_prefix not in text:
        # Find the previous version from existing links (matches both
        # /compare/ and /releases/tag/ URL formats)
        existing_links = re.findall(
            r"\[(\d+\.\d+\.\d+)\]: https://github\.com/SCGIS-Wales/helm-mcp/",
            text,
        )
        if existing_links:
            prev_version = existing_links[0]  # First match is the latest
            new_link = (
                f"[{version}]: https://github.com/SCGIS-Wales/helm-mcp/compare/"
                f"v{prev_version}...v{version}\n"
            )
            # Insert before the first existing version link
            first_link_pos = text.find(f"[{prev_version}]:")
            if first_link_pos > 0:
                text = text[:first_link_pos] + new_link + text[first_link_pos:]

    with open(changelog_path, "w") as f:
        f.write(text)

    print(f"Updated CHANGELOG.md with v{version} from PR #{pr_number}")


def main() -> None:
    if len(sys.argv) < 3:
        print(f"Usage: {sys.argv[0]} <version> <repo>", file=sys.stderr)
        sys.exit(1)

    version = sys.argv[1]
    repo = sys.argv[2]
    tag = f"v{version}"

    print(f"Updating CHANGELOG for {tag} in {repo}")

    # Find the PR that triggered this release
    pr_number = find_pr_for_tag(tag, repo)
    if pr_number is None:
        print(f"No PR found for tag {tag}, skipping CHANGELOG update")
        sys.exit(0)

    print(f"Found PR #{pr_number} for {tag}")

    # Get PR details
    title = get_pr_title(pr_number, repo)
    summary = get_pr_summary(pr_number, repo)
    category = classify_pr(title)

    if not summary:
        print(f"PR #{pr_number} has no summary, skipping")
        sys.exit(0)

    # Convert to changelog entries
    content = summary_to_changelog_entries(summary, pr_number, category)
    if not content:
        print(f"No changelog entries extracted from PR #{pr_number}")
        sys.exit(0)

    # Get the release date
    today = datetime.now(timezone.utc).strftime("%Y-%m-%d")

    # Check if this version already exists in the changelog
    changelog_path = os.path.join(os.path.dirname(__file__), "..", "CHANGELOG.md")
    changelog_path = os.path.abspath(changelog_path)
    with open(changelog_path) as f:
        existing = f.read()

    if f"## [{version}]" in existing:
        print(f"Version {version} already in CHANGELOG.md, skipping")
        sys.exit(0)

    update_changelog(version, today, content, pr_number)


if __name__ == "__main__":
    main()
