# GitHub Actions Chapter 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** เพิ่มบทเรียน GitHub บทที่ 2 (GitHub Actions) พร้อม CI จริงบน PR และ publish ขึ้นเว็บ

**Architecture:** ใช้ workflow `docs` ที่ deploy อยู่แล้วเป็นบทเรียนหลัก เพิ่ม `ci.yml` ที่รัน validator + strict build บน `pull_request` ด้วย matrix Python 2 เวอร์ชันและ pip cache แล้วเขียนบทจาก run จริง จากนั้น merge ผ่าน PR เพื่อพิสูจน์ว่า CI ทำงานก่อนถึง main

**Tech Stack:** GitHub Actions, `gh` CLI, Markdown, MkDocs Material 9.x, Python 3.11/3.12, PyYAML

**Spec:** `specs/2026-11-15-github-actions-design.md`

## Global Constraints

- เนื้อหาไทย; log/ตัวอย่างทุกชิ้นต้องมาจาก run หรือคำสั่งจริง ห้ามแต่งตัวเลข
- 1 ไฟล์ = 1 บท, H1 เดียว, ชื่อไฟล์ `docs/git/02-actions.md`
- `mkdocs build --strict` ต้อง exit 0; `python3 tools/check_docs.py` ต้องผ่าน
- CI ต้องไม่ deploy — deploy เกิดจาก `pages.yml` บน `main` เท่านั้น
- ห้าม commit secret; ใช้ `GITHUB_TOKEN` อัตโนมัติเท่านั้น
- commit message แบบ conventional; git commands รันทีละคำสั่ง (กัน index.lock)
- action majors คงตามที่ verify แล้ว: checkout@v4, setup-python@v5, upload-artifact@v4, upload-pages-artifact@v3, deploy-pages@v4
- ถ้า cache ไม่ hit หรือ matrix fail ให้บันทึกตามจริงแล้วแก้ chapter ไม่ใช่แก้ log

## Review Focus

- **`on:` ถูก PyYAML อ่านเป็น `True`** — บทต้องเตือนและแสดงวิธีตรวจที่ถูก (quote `"on"` หรืออ่านด้วย `yaml.compose`)
- **artifact ชื่อซ้ำใน matrix** — upload-artifact@v4 error ถ้าชื่อเดียวกัน ต้องมี `site-${{ matrix.python-version }}`
- **paths filter ของ `ci`** — ต้องรวม `requirements.txt` และ `.github/workflows/**` ไม่งั้นแก้ workflow แล้ว CI ไม่รัน
- **secret ห้ามพิมพ์ค่า** — สอนเฉพาะชื่อ/กลไก และ `GITHUB_TOKEN` ที่ Actions ให้มา
- **PR ต้องไม่ deploy** — `permissions` ของ `ci` มีแค่ `contents: read`
- **cache key** — `cache-dependency-path: requirements.txt` ต้องมีจริง ไม่งั้น setup-python fail

---

### Task 1: CI foundation

**Files:**
- Create: `requirements.txt`
- Create: `.github/workflows/ci.yml`
- Modify: `.github/workflows/pages.yml` (setup-python + install step)

**Interfaces:**
- Produces: workflow `ci` (job `check`, matrix `3.11`/`3.12`, artifact `site-<version>`), ไฟล์ dependency กลาง `requirements.txt`
- Consumes: `tools/check_docs.py`, `mkdocs.yml`

- [ ] **Step 1: สร้าง branch**

Run: `git switch -c docs/github-actions-ch2`

- [ ] **Step 2: เขียน `requirements.txt`**

```text
mkdocs-material==9.*
```

- [ ] **Step 3: เขียน `.github/workflows/ci.yml`**

```yaml
name: ci

on:
  pull_request:
    paths:
      - "docs/**"
      - "mkdocs.yml"
      - "tools/**"
      - "requirements.txt"
      - ".github/workflows/**"
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true

jobs:
  check:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        python-version: ["3.11", "3.12"]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: ${{ matrix.python-version }}
          cache: pip
          cache-dependency-path: requirements.txt
      - name: Install MkDocs Material
        run: pip install -r requirements.txt
      - name: Validate lesson pages
        run: python3 tools/check_docs.py
      - name: Build site
        run: mkdocs build --strict
      - name: Upload site
        uses: actions/upload-artifact@v4
        with:
          name: site-${{ matrix.python-version }}
          path: site
          retention-days: 7
```

- [ ] **Step 4: แก้ `pages.yml` ให้ใช้ dependency กลาง + cache**

แทนบล็อกเดิม:

```yaml
      - uses: actions/setup-python@v5
        with:
          python-version: "3.12"
      - name: Install MkDocs Material
        run: pip install "mkdocs-material==9.*"
```

ด้วย:

```yaml
      - uses: actions/setup-python@v5
        with:
          python-version: "3.12"
          cache: pip
          cache-dependency-path: requirements.txt
      - name: Install MkDocs Material
        run: pip install -r requirements.txt
```

- [ ] **Step 5: ตรวจ YAML และ local gates**

Run:
```bash
python3 - <<'PY'
import yaml
for f in [".github/workflows/ci.yml", ".github/workflows/pages.yml"]:
    d = yaml.safe_load(open(f))
    print(f, "keys:", [k for k in d if k is not True])
    print("  triggers:", d.get(True))
PY
python3 tools/check_docs.py
/tmp/opencode/venv/bin/mkdocs build --strict >/tmp/ch2-build.log 2>&1; echo "build=$?"
```
Expected: YAML parse ได้ทั้งสองไฟล์, triggers แสดง `pull_request`/`workflow_dispatch` และ `push`/`workflow_dispatch`; validator `ok: 12 pages`; `build=0`

- [ ] **Step 6: Commit**

```bash
git add requirements.txt .github/workflows/ci.yml .github/workflows/pages.yml
git commit -m "ci: add pull request checks and shared requirements"
```

---

### Task 2: พิสูจน์ CI บน PR

**Files:** ไม่มีไฟล์ใหม่ — push และเก็บหลักฐานจาก run จริง

**Interfaces:**
- Consumes: workflow `ci` จาก Task 1
- Produces: run id, ชื่อ matrix job, log validator, artifact — หลักฐานสำหรับบท

- [ ] **Step 1: Push branch**

Run: `git push -u origin docs/github-actions-ch2`

- [ ] **Step 2: เปิด PR**

Run:
```bash
gh pr create --base main --head docs/github-actions-ch2 \
  --title "docs(git): add GitHub Actions chapter and PR CI" \
  --body "เพิ่ม CI สำหรับ PR (matrix 3.11/3.12 + cache + artifact) และบทเรียน GitHub Actions บทที่ 2 — ไม่ deploy จาก PR"
```

- [ ] **Step 3: รอ CI และบันทึกผล**

Run:
```bash
gh pr checks --watch --interval 10
gh run list --workflow ci --limit 1 --json databaseId,headSha,status,conclusion --jq '.[]'
```
Expected: check ทั้งสองตัว success; เก็บ `databaseId` ไว้

- [ ] **Step 4: เก็บหลักฐาน log/artifact**

Run:
```bash
RUN=<databaseId>
gh run view "$RUN" --log | grep -E 'ok: 12 pages|Documentation built|Job .* matrix' | head
gh run view "$RUN" --json jobs --jq '.jobs[].name'
gh run download "$RUN" -n site-3.12 -D /tmp/ch2-artifact
ls /tmp/ch2-artifact | head
```
Expected: เห็น `ok: 12 pages`, ชื่อ job `check (3.11)`/`check (3.12)`, ดาวน์โหลด artifact ได้และมี `index.html`

> ถ้า CI fail: อ่าน `gh run view "$RUN" --log-failed`, แก้ workflow, commit, push ซ้ำ แล้วเก็บหลักฐานรอบใหม่ — ห้ามเขียนบทจาก run ที่ fail

---

### Task 3: บทเรียน + nav

**Files:**
- Create: `docs/git/02-actions.md`
- Modify: `tools/check_docs.py:14-27` (EXPECTED +1)
- Modify: `mkdocs.yml` (nav ใต้ Git & GitHub)
- Modify: `docs/index.md`, `README.md` (Git 1 → 2 บท)

**Interfaces:**
- Consumes: หลักฐานจาก Task 2
- Produces: หน้าที่ 13 ของเว็บ, nav 2 ลิงก์ Git

- [ ] **Step 1: เพิ่มหน้าลง EXPECTED (test ที่ยัง fail)**

ใน `tools/check_docs.py` เพิ่มต่อจาก `docs/git/01-git-github.md`:

```python
    "docs/git/02-actions.md",
```

- [ ] **Step 2: รัน validator ให้เห็น RED**

Run: `python3 tools/check_docs.py`
Expected: `FAIL` + `- missing: docs/git/02-actions.md`, exit 1

- [ ] **Step 3: เขียน `docs/git/02-actions.md`**

โครงบังคับ:

```
# GitHub บทที่ 2: GitHub Actions — ให้ robot ตรวจงานก่อน merge
## 2.1 Actions คืออะไร — runner, workflow, job, step, action
## 2.2 trigger — push, paths, pull_request, workflow_dispatch
## 2.3 permissions และ GITHUB_TOKEN
## 2.4 job และ step — runs-on, uses vs run, needs
## 2.5 artifact และ Pages
## 2.6 concurrency — กัน deploy ซ้อน
## 2.7 เพิ่ม CI บน PR — matrix, cache, artifact
## 2.8 debug — gh run list/view --log, annotation, secrets
## 2.9 แบบฝึกหัด 4 ข้อ + สรุป
```

ค่าที่ต้องใส่ (จากของจริงเท่านั้น):

| หัวข้อ | หลักฐาน |
|---|---|
| workflow เดิม | `gh workflow list` → `docs active`; run 43661114116 jobs `build`,`deploy` |
| trigger | `gh workflow view docs --yaml` แสดง `push` + `paths` + `workflow_dispatch` |
| permissions | `contents: read`, `pages: write`, `id-token: write` พร้อมเหตุผล |
| artifact/Pages | upload-pages-artifact v3 → deploy-pages v4, `environment.url` จาก `steps.deployment.outputs.page_url` |
| concurrency | `group: pages`, `cancel-in-progress: false` |
| ci.yml | matrix `3.11`/`3.12`, job ชื่อ `check (3.11)`/`check (3.12)`, `ok: 12 pages`, artifact `site-3.12` |
| YAML trap | PyYAML `yaml.safe_load` คืน key `True` แทน `on` (แสดงจริง) |
| debug | `gh run view --log`, `gh run view --json jobs`, `gh run download` |

ต้องมี caveat อย่างน้อย 2 จุด: (1) `on:` เป็น boolean ใน YAML 1.1, (2) Node20 shim warning เป็นข้อมูลไม่ใช่ failure

- [ ] **Step 4: รัน validator ให้ผ่าน**

Run: `python3 tools/check_docs.py`
Expected: `ok: 13 pages`, exit 0

- [ ] **Step 5: ต่อเข้า nav และหน้าแรก**

- `mkdocs.yml` ใต้ `git/01-git-github.md`:
```yaml
      - 2. GitHub Actions (CI/CD): git/02-actions.md
```
- `docs/index.md`: `**Git & GitHub** (1 บท)` → `(2 บท)` และเพิ่ม `GitHub Actions/CI`
- `README.md`: `**Git & GitHub (1 บท)**` → `(2 บท)` และเพิ่ม `GitHub Actions`

- [ ] **Step 6: ตรวจ build + nav**

Run:
```bash
python3 tools/check_docs.py
/tmp/opencode/venv/bin/mkdocs build --strict; echo "build=$?"
test -f site/git/02-actions/index.html && echo page-ok
grep -o 'href="git/0[12][^"]*"' site/index.html | sort -u
```
Expected: `ok: 13 pages`; `build=0`; `page-ok`; เห็น 2 ลิงก์ Git

- [ ] **Step 7: ตรวจ mixed script / chat text**

Run: `grep -nE 'ครับ|ผม|เดี๋ยว|所以你|主线|причина|�' docs/git/02-actions.md || echo clean`
Expected: `clean`

- [ ] **Step 8: Commit**

```bash
git add docs/git/02-actions.md tools/check_docs.py mkdocs.yml docs/index.md README.md
git commit -m "docs(git): add chapter 2 on GitHub Actions"
```

- [ ] **Step 9: Push และเก็บ run ที่สอง (cache hit)**

Run: `git push`
แล้ว:
```bash
gh pr checks --watch --interval 10
gh run list --workflow ci --limit 1 --json databaseId --jq '.[0].databaseId'
gh run view <id> --log | grep -Ei 'cache (restored|hit)|ok: 13 pages' | head
```
Expected: CI success; log แสดง `ok: 13 pages` และ cache restored (ถ้าไม่ hit ให้บันทึกตามจริงและอธิบายในบท)

---

### Task 4: Merge, deploy และตรวจของจริง

**Files:** ไม่มีไฟล์ใหม่ — merge และตรวจ production

**Interfaces:**
- Consumes: commits ทั้งหมดบน branch
- Produces: หน้าใหม่บนเว็บ, 13/13 หน้า 200

- [ ] **Step 1: Merge ผ่าน PR**

Run: `gh pr merge --rebase --delete-branch`
Expected: merge สำเร็จ; `git switch main && git pull --ff-only`

- [ ] **Step 2: ดู deploy run**

Run:
```bash
gh run list --workflow docs --limit 1 --json databaseId --jq '.[0].databaseId'
gh run watch <databaseId> --exit-status --interval 10
```
Expected: build + deploy success

- [ ] **Step 3: Poll หน้าใหม่**

Run:
```bash
for i in $(seq 1 18); do
  code=$(curl -s -o /dev/null -w '%{http_code}' https://sakzeed.github.io/opencode-lab/git/02-actions/)
  echo "$i: $code"; [ "$code" = 200 ] && break; sleep 15
done
```
Expected: `200`

- [ ] **Step 4: ตรวจ marker และทุกหน้า**

Run:
```bash
curl -s https://sakzeed.github.io/opencode-lab/git/02-actions/ | grep -c 'workflow_dispatch'
curl -s https://sakzeed.github.io/opencode-lab/git/02-actions/ | grep -c 'matrix'
curl -s https://sakzeed.github.io/opencode-lab/ | grep -c '02-actions'
for p in opencode/01-architecture opencode/02-configuration opencode/03-agents opencode/04-commands-skills opencode/05-permissions opencode/06-mcp-plugins opencode/07-cli-api linux/01-filesystem linux/02-text-pipes linux/03-permissions linux/04-processes git/01-git-github git/02-actions; do
  printf '%-34s %s\n' "$p" "$(curl -s -o /dev/null -w '%{http_code}' https://sakzeed.github.io/opencode-lab/$p/)"
done
```
Expected: marker ≥1; 13/13 = 200

- [ ] **Step 5: ตรวจ main ตรงกับ origin และ append ledger**

Run: `git status --short; git log -1 --format='%h %s'; git rev-list --left-right --count origin/main...HEAD`
แล้ว append `/tmp/opencode/sdd-docs-site/progress.md`: วันที่, commits, run ids, ผล CI/deploy/200
