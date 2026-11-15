# GitHub บทที่ 2: GitHub Actions — ให้ robot ตรวจงานก่อน merge

บทที่ 1 จบที่ `push` ขึ้น GitHub และเปิด PR ได้ แต่ยังเหลือคำถามสำคัญ: **"ใครตรวจว่าโค้ดที่ push ไปยังใช้ได้จริง ก่อนที่มันจะถึง main"**

คำตอบของ repo นี้คือ GitHub Actions — และเว็บที่คุณกำลังอ่านอยู่ก็ถูก build และ deploy ด้วย Actions ทุกครั้งที่ push `main` บทนี้จึงแกะของจริงที่ทำงานอยู่ แล้วเพิ่ม CI สำหรับ PR เข้าไปด้วย

---

## 2.1 Actions คืออะไร — runner, workflow, job, step, action

GitHub Actions คือเครื่องยนต์ที่รันคำสั่งบนเครื่องของ GitHub เมื่อมี event เกิดขึ้น องค์ประกอบมี 4 ชั้น:

| ชั้น | คืออะไร | ตัวอย่างใน repo นี้ |
|---|---|---|
| **workflow** | ไฟล์ YAML 1 ไฟล์ใน `.github/workflows/` | `pages.yml`, `ci.yml` |
| **event** | สิ่งที่ทำให้ workflow เริ่มทำงาน | `push`, `pull_request`, `workflow_dispatch` |
| **job** | กลุ่มคำสั่งที่รันบน runner 1 เครื่อง | `build`, `deploy`, `check` |
| **step** | คำสั่งเดี่ยวใน job (`run:`) หรือ action สำเร็จรูป (`uses:`) | `mkdocs build --strict`, `actions/checkout@v4` |

**action** คือชิ้นงานที่คนอื่นเขียนไว้ให้เรียกใช้ เช่น `actions/checkout@v4` (โหลด repo ลง runner) หรือ `actions/setup-python@v5` (ติดตั้ง Python) — เรียกด้วย `uses:` ส่วน `run:` คือคำสั่ง shell ที่เราเขียนเอง

repo นี้มี 2 workflow:

```sh
$ gh workflow list
ci	active	418781578
docs	active	418716202
```

- `docs` — build เว็บแล้ว deploy ขึ้น Pages (ทำงานบน `main`)
- `ci` — ตรวจ validator + build บน PR (ไม่ deploy) — เพิ่มในบทนี้

แผนที่ของ `pages.yml`:

```yaml
on: push (main) ──► job: build ──► job: deploy
                     checkout           deploy-pages
                     setup-python
                     pip install
                     check_docs.py
                     mkdocs build
                     upload-pages-artifact
```

---

## 2.2 trigger — push, paths, pull_request, workflow_dispatch

trigger อยู่ในบล็อก `on:` ของ workflow:

```yaml
on:
  push:
    branches: [main]
    paths:
      - "docs/**"
      - "mkdocs.yml"
      - "tools/**"
      - ".github/workflows/pages.yml"
  workflow_dispatch:
```

| ส่วน | ความหมาย |
|---|---|
| `push.branches: [main]` | ทำงานเมื่อ push เข้า `main` เท่านั้น |
| `push.paths` | ทำงานเมื่อมีไฟล์ในรายการเปลี่ยน — แก้ `specs/**` หรือ `README.md` ไม่ trigger |
| `workflow_dispatch` | ปุ่มกดรันเอง (หรือ `gh workflow run`) |

นี่คือเหตุผลที่ commit `458fd8b` (แก้แค่ `specs/`) ไม่ทำให้เว็บ deploy ใหม่ — ไม่ใช่บั๊ก แต่เป็น `paths` ที่ตั้งใจ

สั่งรันเองด้วย `gh` ได้ และมันคืน URL ของ run:

```sh
$ gh workflow run ci --ref docs/github-actions-ch2
https://github.com/sakzeed/opencode-lab/actions/runs/43662751198
```

`pull_request` ใช้กับ CI: ทุกครั้งที่เปิด PR หรือ push เพิ่มใน PR นั้น workflow จะรัน โดย `github.ref` เป็น `refs/pull/<เลข>/merge`

> ⚠️ caveat: `paths` ที่แคบเกินไปทำให้ CI ไม่รันในจุดที่ควร — ตอนเพิ่ม `requirements.txt` จึงต้องใส่ path นี้ใน `ci.yml` ด้วย ไม่งั้นแก้ dependency แล้วไม่มีอะไรตรวจ

---

## 2.3 permissions และ GITHUB_TOKEN

ทุก workflow ได้ `GITHUB_TOKEN` มาโดยอัตโนมัติ — เป็น token ชั่วคราวของ repo นั้น อายุเท่า run และไม่ต้องสร้าง secret เอง

แต่ token จะมีสิทธิ์เท่าที่ workflow ขอ:

```yaml
permissions:
  contents: read
  pages: write
  id-token: write
```

| permission | ใช้ทำอะไร |
|---|---|
| `contents: read` | อ่านโค้ดใน repo (checkout) |
| `pages: write` | เขียน artifact ลง Pages |
| `id-token: write` | ขอ OIDC token สำหรับ deploy-pages |

`ci.yml` ขอแค่ `contents: read` เพราะแค่อ่านโค้ดกับ build — **PR จาก branch อื่นไม่ควรได้สิทธิ์ deploy**

ส่วน secret ที่เราเพิ่มเอง (API key ฯลฯ) เก็บใน Settings → Secrets and variables → Actions และอ่านได้อย่างเดียวผ่าน `${{ secrets.NAME }}` — ดูรายชื่อได้แต่ดูค่าคืนไม่ได้:

```sh
$ gh secret list
# (repo นี้ยังไม่มี secret ที่ตั้งเอง)
```

> ⚠️ ห้าม `echo` secret ลง log — log ของ Actions อ่านได้โดยคนที่เข้าถึง repo และ mask ของ GitHub ไม่ครอบคลุมทุกกรณี

---

## 2.4 job และ step — runs-on, uses vs run, needs

job หนึ่งอันคือ container/VM หนึ่งเครื่อง:

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install MkDocs Material
        run: pip install -r requirements.txt
```

- `runs-on` — เลือก runner (`ubuntu-latest` คือ VM Linux ของ GitHub)
- `uses` — เรียก action สำเร็จรูป พร้อม version tag
- `run` — คำสั่ง shell; ใส่ `name:` ให้อ่าน log ง่าย
- `needs: build` — job `deploy` รอ `build` สำเร็จก่อน
- `environment: github-pages` — ผูก job กับ environment ที่มี URL

job ส่งข้อมูลถึงกันด้วย output:

```yaml
deploy:
  needs: build
  environment:
    name: github-pages
    url: ${{ steps.deployment.outputs.page_url }}
  steps:
    - id: deployment
      uses: actions/deploy-pages@v4
```

`steps.deployment.outputs.page_url` คือ URL จริงที่ Pages คืนมา — ไม่ต้อง hardcode

**matrix** คือการรัน job เดียวกันหลายค่า:

```yaml
strategy:
  matrix:
    python-version: ["3.11", "3.12"]
```

ผลคือได้ 2 job แยกกันชื่อ `check (3.11)` และ `check (3.12)` — เห็นจริงจาก PR:

```sh
$ gh pr checks 2
check (3.11)	pass	23s	https://github.com/sakzeed/opencode-lab/actions/runs/43662647654/job/132655112748
check (3.12)	pass	16s	https://github.com/sakzeed/opencode-lab/actions/runs/43662647654/job/132655112538
```

---

## 2.5 artifact และ Pages

**artifact** คือไฟล์ที่ job ผลิตแล้วเก็บไว้ให้ job อื่นหรือคนดาวน์โหลด — ไม่ใช่ commit

`pages.yml` ใช้ 2 action คู่กัน:

```yaml
# job build
- uses: actions/upload-pages-artifact@v3
  with:
    path: site
# job deploy
- uses: actions/deploy-pages@v4
```

`upload-pages-artifact` บีบ `site/` เป็น artifact ชื่อ `github-pages` แล้ว `deploy-pages` ใน job ถัดไปหยิบไปเผยแพร่ — นี่คือเหตุผลที่ deploy ไม่ต้อง checkout หรือ build ซ้ำ

`ci.yml` ก็อัป artifact เหมือนกัน แต่เพื่อให้คนดาวน์โหลดไปตรวจ:

```sh
$ gh run download 43662647654 -n site-3.12 -D /tmp/ch2-artifact
$ ls /tmp/ch2-artifact | head -5
404.html
assets
git
index.html
linux
```

> ⚠️ caveat: `upload-artifact@v4` ไม่ยอมให้ artifact ชื่อซ้ำกันใน run เดียว — ใน matrix ต้องใส่ชื่อไม่เหมือนกัน เช่น `site-${{ matrix.python-version }}` ไม่งั้น job หลังจะ error
>
> ⚠️ caveat ของเครื่องนี้ (proot): `gh run download` ใช้ `/data/local/tmp` เป็น temp แล้ว fail เพราะ path ของ Android ไม่มีใน container — แก้ด้วย `TMPDIR=/tmp gh run download ...`

---

## 2.6 concurrency — กัน deploy ซ้อน

ถ้า push 2 ครั้งติดกัน deploy อาจทับกัน ตัวกันคือ `concurrency`:

```yaml
concurrency:
  group: pages
  cancel-in-progress: false
```

| ค่า | ความหมาย |
|---|---|
| `group` | ชื่อคิว — workflow ที่ group เดียวกันจะรันทีละอัน |
| `cancel-in-progress: false` | รอตัวที่รันอยู่จบก่อน ไม่ฆ่ากลางคัน |

deploy ไม่ควรถูกยกเลิกกลางทางเพราะ Pages อาจค้างสถานะ — จึงเป็น `false` ต่างจาก `ci.yml` ที่ใช้:

```yaml
concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true
```

PR ที่ push รัว ๆ ไม่ต้องเสียเวลา run เวอร์ชันเก่า — ยกเลิกได้ปลอดภัยเพราะแค่ตรวจ ไม่มี side effect

---

## 2.7 เพิ่ม CI บน PR — matrix, cache, artifact

นี่คือ workflow ที่เพิ่มในบทนี้ทั้งไฟล์:

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

จุดสำคัญ:

- **ไม่มี job deploy** และ `permissions` มีแค่ `contents: read` — PR ตรวจได้อย่างเดียว
- **matrix 3.11/3.12** จับความต่างของ Python ก่อนถึง main
- **`fail-fast: false`** — ถ้า 3.11 พัง ยังได้ผลของ 3.12 ไว้เทียบ
- **cache** — `cache: pip` + `cache-dependency-path: requirements.txt` ให้ setup-python แคช wheel ไว้ใช้รอบหน้า
- **artifact** — เก็บ `site/` ไว้ 7 วัน ให้ดาวน์โหลดมาตรวจได้

ผลจริงจาก PR แรก (run `43662647654`):

```sh
$ gh run view 43662647654 --json jobs --jq '.jobs[].name'
check (3.12)
check (3.11)

$ gh run view 43662647654 --log | grep 'ok: 12 pages'
check (3.12)	Validate lesson pages	... ok: 12 pages
check (3.11)	Validate lesson pages	... ok: 12 pages
```

รอบนั้นยังไม่มีหน้าบทนี้ (12 หน้า) — หลังเพิ่มบทนี้เป็น 13 หน้า CI จะตรวจ 13

cache ทำงานแบบ "เขียนก่อน อ่านรอบหน้า" — log รอบแรกบอกตรง ๆ:

```sh
$ gh run view 43662647654 --log | grep -i 'pip cache'
... pip cache is not found
```

> ⚠️ cache ผูกกับ ref: cache ที่ PR สร้างจะถูกใช้ใน PR เดิม (`refs/pull/2/merge`) แต่ `workflow_dispatch` ที่สั่งบน branch (`refs/heads/...`) คนละ scope จึงยัง miss — เป็นพฤติกรรมของ Actions ไม่ใช่ความผิดพลาด

---

## 2.8 debug — gh run list/view --log, annotation, secrets

คำสั่งที่ใช้บ่อยที่สุด:

```sh
$ gh run list --workflow ci --limit 2
completed	success	ci	ci	docs/github-actions-ch2	workflow_dispatch	43662751198	25s	2026-11-15T01:21:57Z
completed	success	docs(git): add GitHub Actions chapter and PR CI	ci	docs/github-actions-ch2	pull_request	43662647654	26s	2026-11-15T01:20:41Z

$ gh run view 43662647654 --json jobs --jq '.jobs[] | {name, conclusion}'
{"name":"check (3.12)","conclusion":"success"}
{"name":"check (3.11)","conclusion":"success"}
```

| ต้องการ | คำสั่ง |
|---|---|
| ดู run ล่าสุด | `gh run list --limit 5` |
| ตามดูสด ๆ | `gh run watch <id> --exit-status` |
| ดู log ทั้งหมด | `gh run view <id> --log` |
| ดูเฉพาะ step ที่ fail | `gh run view <id> --log-failed` |
| ดู job/step | `gh run view <id> --json jobs` |
| ดู PR checks | `gh pr checks <pr>` |
| ดาวน์โหลด artifact | `gh run download <id> -n site-3.12` |

exit code ของ workflow คือ exit code ของ step ที่ fail — `mkdocs build --strict` ที่เจอ link เสียจะคืน 1 และ run เป็น `failure` ทันที

**annotation** คือข้อความเตือนเหนือ log เช่น Node20 deprecation ที่เห็นใน run ของ repo นี้:

```text
! Node.js 20 is deprecated. The following actions target Node.js 20 but are being forced to run on Node.js 24...
```

เป็นข้อมูลให้อัปเกรด action ในอนาคต ไม่ใช่สาเหตุที่ run fail — อย่าตกใจถ้าเห็น warning นี้ทั้งที่ run เป็นเขียว

### กับดัก YAML: `on:` ไม่ใช่ string

ถ้าใช้ Python อ่าน workflow:

```sh
$ python3 -c 'import yaml; print(list(yaml.safe_load(open(".github/workflows/ci.yml"))))'
['name', True, 'permissions', 'concurrency', 'jobs']
```

`on` หายไป กลายเป็น `True` เพราะ YAML 1.1 ตีความ `on` เป็น boolean — เครื่องมือที่ parse ด้วย PyYAML ต้องรู้เรื่องนี้ (หรืออ่านด้วย `yaml.compose`) ไม่ใช่แปลว่าไฟล์ผิด

> ⚠️ caveat: `gh run view --log` ของ run เก่าอาจหมดอายุตาม retention (default 90 วัน) — เก็บ log สำคัญไว้เองถ้าต้องอ้างอิงนาน

---

## 2.9 แบบฝึกหัด 4 ข้อ + สรุป

### แบบฝึกหัด

1. เพิ่ม step ชื่อ `Show Python` ที่รัน `python3 --version` ใน `ci.yml` แล้ว push ไป branch ใหม่ — ตรวจว่าเห็น step นี้ใน `gh run view --log` ทั้ง 2 matrix job
2. เปลี่ยน matrix เป็น `["3.12"]` อย่างเดียว แล้วเทียบชื่อ job และเวลา run กับก่อนแก้
3. ทำให้ validator fail โดยสร้างไฟล์ `docs/tmp-broken.md` ที่ไม่มี H1 แล้ว push ขึ้น branch (อย่า merge) — ดู `gh run view --log-failed` และ annotation ว่า job แรกที่ fail คือ matrix ตัวไหน
4. เพิ่ม `specs/**` ใน `paths` ของ `ci.yml` แล้ว push commit ที่แก้แค่ `specs/` — สังเกตว่า CI รันหรือไม่ และอธิบายว่าทำไม

### เช็คความเข้าใจ

1. ทำไม `ci.yml` ไม่มี job `deploy` และทำไม `permissions` ของมันมีแค่ `contents: read`
2. `needs: build` กับ `concurrency.group` แก้ปัญหาต่างกันอย่างไร
3. ถ้า artifact ใน matrix ใช้ชื่อ `site` เหมือนกันทั้ง 2 job จะเกิดอะไร และแก้อย่างไร

---

## สรุปบทที่ 2

1. **workflow = event → job → step** — `uses` เรียก action, `run` รัน shell, `needs` เรียงลำดับ
2. **สิทธิ์ต้องน้อยที่สุด** — PR ตรวจอย่างเดียวพอ; deploy เก็บไว้ให้ `main`
3. **ของที่ผลิตควรถูกเก็บ** — artifact ทำให้ตรวจย้อนหลังได้; cache ทำให้รอบถัดไปเร็วขึ้น (แต่มันผูกกับ ref)
4. **debug ด้วย `gh`** — `run list`, `run view --log`, `log-failed`, `pr checks`, `run download`
5. **อย่าเชื่อ YAML 1.1** — `on:` กลายเป็น `True` ได้; และ `paths` ที่แคบเกินคือสาเหตุอันดับต้น ๆ ของ "ทำไม CI ไม่รัน"
