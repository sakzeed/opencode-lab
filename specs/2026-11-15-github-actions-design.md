# GitHub บทที่ 2: GitHub Actions — design

วันที่: 2026-11-15
สถานะ: อนุมัติแล้ว (ผู้ใช้: "อนุมัติ")

## ทำไม

Git บทที่ 1 จบที่ push → branch → PR แต่ยังไม่ตอบว่า "หลัง push แล้วใครตรวจงาน" — repo นี้ publish เว็บด้วย Actions อยู่แล้ว และตอนนี้ **ไม่มี CI บน PR เลย** (workflow `docs` trigger แค่ `push main` + `workflow_dispatch`) บทนี้จึงสอนกลไก Actions จาก workflow จริง พร้อมปิดช่องว่างนั้นด้วย `ci.yml` ที่รัน validator + strict build ก่อน merge

## ขอบเขต

**ทำ**
- `docs/git/02-actions.md` 1 ไฟล์ (9 ส่วน + แบบฝึกหัด 4 ข้อ)
- `requirements.txt` (`mkdocs-material==9.*`) ให้ทั้ง два workflow ใช้ไฟล์เดียวกัน
- `.github/workflows/ci.yml` — `pull_request` + `workflow_dispatch`, matrix Python 3.11/3.12, pip cache, upload site artifact, ไม่ deploy
- แก้ `pages.yml`: ใช้ `pip install -r requirements.txt` + `cache: pip`
- `tools/check_docs.py` เพิ่มหน้าใหม่ใน EXPECTED
- `mkdocs.yml` nav, `docs/index.md`, `README.md`: Git track 1 → 2 บท

**ไม่ทำ**
- self-hosted runner, reusable workflow เชิงลึก, composite/Docker action
- OIDC ไป cloud, environment protection rules, branch protection
- secrets จริง (สอนเฉพาะกลไก + `GITHUB_TOKEN` อัตโนมัติ)
- deploy preview ต่อ PR

## ทางเลือกที่พิจารณา

| ทาง | เนื้อหา | trade-off | สรุป |
|---|---|---|---|
| **A. Anatomy + PR CI (เลือก)** | แกะ `pages.yml` → เพิ่ม `ci.yml` จริงบน PR → debug ด้วย `gh run view --log` → cache/matrix/artifact/secrets | ได้ของใช้จริงและมี run จริงให้ดู; เพิ่ม workflow 1 ไฟล์ | **เลือก** |
| B. สร้าง workflow จากศูนย์ | workflow ใหม่ใน sandbox branch แล้วเทียบของจริง | พื้นฐานครบ แต่ไม่ได้แตะ pipeline ที่ publish | ไม่เลือก |
| C. Debug/incident | ไล่ run ที่ fail, อ่าน log, permission/concurrency | ฝึกแก้ปัญหา แต่ยังไม่มี run ที่ fail | ไม่เลือก |

## ข้อเท็จจริงจากเครื่อง (ทดสอบแล้ว)

- repo มี workflow เดียว: `docs` (id 418716202); run ล่าสุด 43661114116 success, jobs `build` + `deploy`
- `gh workflow list`, `gh workflow view docs --yaml`, `gh run view --log` ใช้ได้ (log ~502 บรรทัด)
- ไม่มี `act`, `actionlint`, `yamllint` — ใช้ `gh` + Python/PyYAML ตรวจโครงสร้างแทน
- PyYAML อ่าน key `on:` เป็น `True` (YAML 1.1) — เป็นกับดักจริงที่บทจะสอน
- `setup-python` cache ต้องมี dependency file จึงต้องมี `requirements.txt`
- local build: `mkdocs build --strict` exit 0, `site/linux/04-processes/index.html` เกิดจริง

## โครงบท

1. Actions คืออะไร — runner, workflow, job, step, action; แผนที่ `pages.yml`
2. Trigger — `push` + `paths`, `workflow_dispatch`, `pull_request`
3. `permissions` + `GITHUB_TOKEN` — ทำไม Pages ต้อง `pages: write` + `id-token: write`
4. Job/step — `runs-on`, `uses` vs `run`, `needs`, `if`, output ส่งต่อ
5. Artifact + Pages — upload → deploy, `environment.url`
6. `concurrency` — กัน deploy ซ้อน
7. CI บน PR — `ci.yml`: matrix 3.11/3.12, pip cache, validator + strict build, upload artifact
8. Debug — `gh run list/view --log`, annotation, exit code; secrets และ cache hit
9. แบบฝึกหัด 4 ข้อ + สรุป

## ไฟล์ที่แตะ

- Create: `docs/git/02-actions.md`, `.github/workflows/ci.yml`, `requirements.txt`
- Modify: `.github/workflows/pages.yml`, `tools/check_docs.py`, `mkdocs.yml`, `docs/index.md`, `README.md`

## Verification

- `python3 tools/check_docs.py` → `ok: 13 pages`
- `mkdocs build --strict` → exit 0; `site/git/02-actions/index.html` มีจริง
- เปิด PR จาก branch → `ci` run success ทั้ง 2 matrix jobs; validator output ใน log คือ `ok: 13 pages`
- push ซ้ำบน branch → run ที่สองเห็น cache hit (log มี `Cache restored`)
- `gh run download` artifact `site-3.12` ได้จริง (มี `index.html`)
- merge PR → `docs` run success → หน้าใหม่ 200 และมี marker (`workflow_dispatch`, `matrix`)
- 13/13 หน้าเว็บ 200; validator + strict build ผ่านบน main

## Left to builder

- ถ้อยคำ/ลำดับย่อยในแต่ละส่วน และจำนวนตัวอย่าง log ที่ยกมา
- จะยก log run ไหนเป็นหลัก (แนะนำ run `ci` แรกของ PR)
- ถ้า cache ไม่ hit ตามคาด ให้บันทึกตามจริงและอธิบายสาเหตุ ไม่แต่ง log
