# Docs site — design

วันที่: 2026-11-14
สถานะ: รออนุมัติ

## ทำไม

บทเรียน 7 บท (OpenCode) + 3 บท (Linux) + 1 บท (Git) ถูกส่งในแชทแล้ว แต่ค้นยากและหายเมื่อ session ถูกลบ
เป้าหมาย: เว็บ docs ที่ค้นหาได้ ลิงก์แชร์ได้ และ deploy อัตโนมัติเมื่อ push

## ขอบเขต

**ทำ**
- `docs/` เป็น source ของเว็บ (Markdown)
- 11 บท: OpenCode 1–7, Linux 1–3, Git & GitHub 1 (โบนัส)
- หน้าแรก + navigation แยกหมวด + search
- GitHub Actions deploy ขึ้น Pages อัตโนมัติ
- เปลี่ยน repo เป็น public (ข้อจำกัด Pages ฟรี)

**ไม่ทำ**
- comments / analytics / เวอร์ชันหลายชุด / custom domain / i18n toggle

## ทางเลือกที่พิจารณา

| ทาง | ข้อดี | ข้อเสีย | สรุป |
|---|---|---|---|
| **MkDocs Material + Actions** | sidebar/search/dark mode/รองรับไทย | ต้องมี CI build | **เลือก** |
| Jekyll native | ไม่ต้อง CI | theme/search จำกัด, Jekyll เก่า | ไม่เลือก |
| Plain HTML | คุมได้ 100% | ต้องดูแล markup เอง, ไม่มี search | ไม่เลือก |

## โครงสร้าง

```
mkdocs.yml
.github/workflows/pages.yml
docs/
  index.md
  opencode/01-architecture.md ... 07-cli-api.md
  linux/01-filesystem.md 02-text-pipes.md 03-permissions.md
  git/01-git-github.md
specs/2026-11-14-docs-site-design.md   # เอกสารนี้ (ไม่ขึ้นเว็บ)
config/ commands/ plugins/             # คงเดิม
README.md                              # เพิ่มลิงก์เว็บ
```

URL: `https://sakzeed.github.io/opencode-lab/`

## แหล่งเนื้อหา

ดึงจาก `opencode session export ses_e25ed55...` (1.5 MB) — ข้อความ assistant ที่มีหัวข้อ `# บทที่ ...`
สคริปต์ extract จะ:
1. เลือก text part ตามหัวข้อที่กำหนด (ไม่ใช้การเดาจากความยาว)
2. ตัดคำนำนอกบทเรียนก่อนบรรทัด `# บท...`
3. normalize heading: H1 เดียวต่อหน้า, เลื่อน H2 → H2 คงเดิม, ไม่แตะ `#` ใน code block
4. เขียนเป็นไฟล์ตาม mapping ที่กำหนด

ตรวจผล: จำนวนไฟล์ = 11, ทุกไฟล์มี H1, ไม่มีข้อความแชทหลงเหลือ, code fence สมดุล

## การ deploy

- Workflow: `actions/checkout` → `setup-python` → `pip install mkdocs-material` → `mkdocs build --strict` → `upload-pages-artifact` → `deploy-pages`
- trigger: push `main` (paths: docs/**, mkdocs.yml, workflow) + `workflow_dispatch`
- permissions: `contents: read`, `pages: write`, `id-token: write`
- concurrency: group `pages`, cancel-in-progress

## Public

- สแกน secret ซ้ำก่อนสลับ visibility
- `gh repo edit --visibility public --accept-visibility-change-consequences`
- หลัง public: เปิด Pages (source = GitHub Actions), ตรวจ URL จริงด้วย `curl`

## ความเสี่ยง

| ความเสี่ยง | ลด |
|---|---|
| เนื้อหาดิบมีคำแชท/บรรทัดแปลก | ตรวจด้วย `mkdocs build --strict` + สุ่มอ่าน 3 บท |
| code block มี `#` ถูกตีเป็น heading | extract เฉพาะบรรทัดนอก code fence |
| public แล้วมีข้อมูลส่วนตัว | secret scan + README เตือน; path `/root` เป็นข้อมูลทั่วไป |
| Pages ต้องมี Pro ถ้า private | เปลี่ยน public ตามที่ตกลง |
