#!/usr/bin/env python3
"""Turn the exported session transcript into one Markdown page per lesson.

The lessons were written inside a chat, so each source part carries a short
preamble (and often a "what next?" footer). This script slices exactly the
lesson, demotes stray H1s to H2, and writes the 11 pages the site expects.

Source: /tmp/opencode/session-export.json (override with LESSON_EXPORT=...)
"""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EXPORT = Path(os.environ.get("LESSON_EXPORT", "/tmp/opencode/session-export.json"))

MAPPING = [
    ("# บทเรียน OpenCode (V2)", "docs/opencode/01-architecture.md"),
    ("# บทที่ 2: Configuration", "docs/opencode/02-configuration.md"),
    ("# บทที่ 3: Agents", "docs/opencode/03-agents.md"),
    ("# บทที่ 4: Commands", "docs/opencode/04-commands-skills.md"),
    ("# บทที่ 5: Permissions", "docs/opencode/05-permissions.md"),
    ("# บทที่ 6: MCP", "docs/opencode/06-mcp-plugins.md"),
    ("# บทที่ 7: CLI", "docs/opencode/07-cli-api.md"),
    ("# บทที่ 1: ระบบไฟล์", "docs/linux/01-filesystem.md"),
    ("# Linux บทที่ 2:", "docs/linux/02-text-pipes.md"),
    ("# Linux บทที่ 3:", "docs/linux/03-permissions.md"),
    ("# Git & GitHub บทที่ 1:", "docs/git/01-git-github.md"),
]


def assistant_texts(export: Path) -> list[str]:
    data = json.loads(export.read_text(encoding="utf-8"))
    texts: list[str] = []
    for message in data.get("messages", []):
        if message.get("type") != "assistant":
            continue
        for part in message.get("content") or []:
            if part.get("type") == "text" and part.get("text"):
                texts.append(part["text"])
    return texts


def slice_lesson(text: str, marker: str) -> str | None:
    lines = text.splitlines()
    for index, line in enumerate(lines):
        if line.startswith(marker):
            return "\n".join(lines[index:])
    return None


def trim_footer(text: str) -> str:
    """Drop the trailing "want to continue?" chat after the last rule.

    A real closing section (e.g. "## สรุป") is long or starts with a heading,
    so it is kept; a short tail after a horizontal rule is chat.
    """
    lines = text.splitlines()
    for index in range(len(lines) - 1, -1, -1):
        if lines[index].strip() != "---":
            continue
        rest = "\n".join(lines[index + 1 :]).strip()
        if not rest:
            return "\n".join(lines[:index]).rstrip() + "\n"
        if len(rest) < 1200 and "```" not in rest and not rest.startswith("## "):
            return "\n".join(lines[:index]).rstrip() + "\n"
    return text.rstrip() + "\n"


def normalize_headings(text: str) -> str:
    """Keep exactly one H1: demote any later H1 outside code fences to H2."""
    out: list[str] = []
    in_fence = False
    seen_h1 = False
    for line in text.splitlines():
        if line.strip().startswith("```"):
            in_fence = not in_fence
            out.append(line)
            continue
        if not in_fence and line.startswith("# "):
            if seen_h1:
                out.append("## " + line[2:])
                continue
            seen_h1 = True
        out.append(line)
    return "\n".join(out).rstrip() + "\n"


def main() -> int:
    if not EXPORT.exists():
        print(f"export not found: {EXPORT}", file=sys.stderr)
        return 1

    texts = assistant_texts(EXPORT)
    written = 0
    missing: list[str] = []

    for marker, rel in MAPPING:
        lesson = None
        for text in texts:
            lesson = slice_lesson(text, marker)
            if lesson is not None:
                break
        if lesson is None:
            missing.append(marker)
            continue

        content = normalize_headings(trim_footer(lesson))
        target = ROOT / rel
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")
        written += 1

    if missing:
        print("markers not found:", file=sys.stderr)
        for marker in missing:
            print("  -", marker, file=sys.stderr)
        return 1

    print(f"wrote {written} files")
    return 0 if written == len(MAPPING) else 1


if __name__ == "__main__":
    sys.exit(main())
