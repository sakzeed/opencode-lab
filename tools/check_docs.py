#!/usr/bin/env python3
"""Validate the generated lesson pages before they ship to Pages.

Checks the failure modes we care about: missing chapters, missing/duplicate
H1, unbalanced code fences, and chat text that leaked in from the session.
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

EXPECTED = [
    "docs/opencode/01-architecture.md",
    "docs/opencode/02-configuration.md",
    "docs/opencode/03-agents.md",
    "docs/opencode/04-commands-skills.md",
    "docs/opencode/05-permissions.md",
    "docs/opencode/06-mcp-plugins.md",
    "docs/opencode/07-cli-api.md",
    "docs/linux/01-filesystem.md",
    "docs/linux/02-text-pipes.md",
    "docs/linux/03-permissions.md",
    "docs/linux/04-processes.md",
    "docs/git/01-git-github.md",
]

CHAT_MARKERS = [
    "มีครับ 👍",
    "บอกได้เลยครับ",
    "พร้อมลงมือจริงไหมครับ",
    "อยากให้ผม **",
    "จะไปทางไหนต่อครับ",
    "หรือถ้าอยากพัก",
    "ก่อนไปบทถัดไป",
    "บอกได้ถ้าอยากให้",
    "เดี๋ยวผม",
]

# A lesson may end with real "## สรุปบทที่ N" / "## แบบฝึกหัด" sections, but these
# headings only ever introduce chat.
CHAT_HEADINGS = (
    "## ก่อนไปบทถัดไป",
    "## อยากให้ผม",
    "## พร้อมลงมือจริงไหม",
    "## จะไปทางไหนต่อ",
)

# Scripts that never belong in the Thai lessons unless they are inside a code
# fence (where they could be sample data). Mixed Thai/Latin adjacency catches
# corruption such as "ไฟล์ystem".
CJK = re.compile(r"[\u3000-\u303f\u3400-\u4dbf\u4e00-\u9fff\uf900-\ufaff]")
CYRILLIC = re.compile(r"[\u0400-\u04ff]")
MIXED_THAI_LATIN = re.compile(r"[\u0e00-\u0e7f][A-Za-z]|[A-Za-z][\u0e00-\u0e7f]")


def main() -> int:
    problems: list[str] = []

    for rel in EXPECTED:
        path = ROOT / rel
        if not path.exists():
            problems.append(f"missing: {rel}")
            continue

        lines = path.read_text(encoding="utf-8").splitlines()
        in_fence = False
        h1 = 0
        fences = 0

        for number, line in enumerate(lines, 1):
            stripped = line.strip()
            if stripped.startswith("```"):
                fences += 1
                in_fence = not in_fence
                continue
            if in_fence:
                continue
            if line.startswith("# "):
                h1 += 1
            if line.startswith(CHAT_HEADINGS):
                problems.append(f"chat section: {rel}:{number}: {line.strip()[:60]}")
            if "\ufffd" in line:
                problems.append(f"replacement char: {rel}:{number}")
            for label, pattern in (
                ("CJK text", CJK),
                ("Cyrillic text", CYRILLIC),
                ("Thai/Latin run-together", MIXED_THAI_LATIN),
            ):
                if pattern.search(line):
                    problems.append(f"{label}: {rel}:{number}: {line.strip()[:70]}")
            for marker in CHAT_MARKERS:
                if marker in line:
                    problems.append(f"chat text: {rel}:{number}: {marker}")

        if not lines or not lines[0].startswith("# "):
            problems.append(f"first line is not H1: {rel}")
        if h1 != 1:
            problems.append(f"expected 1 H1, found {h1}: {rel}")
        if fences % 2:
            problems.append(f"unbalanced code fences: {rel}")

    if problems:
        print("FAIL")
        for problem in problems:
            print("  -", problem)
        return 1

    print(f"ok: {len(EXPECTED)} pages")
    return 0


if __name__ == "__main__":
    sys.exit(main())
