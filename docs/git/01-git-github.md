# Git & GitHub บทที่ 1: git พื้นฐาน → GitHub

บทนี้เป็น **track ใหม่** (Linux บทที่ 4 เรื่อง process ยังค้างอยู่)

ความเข้าใจผิดที่พบบ่อยที่สุดคือคิดว่า git กับ GitHub เป็นสิ่งเดียวกัน:

```
git                              GitHub
─────────────────────────────    ─────────────────────────────
โปรแกรมบนเครื่องคุณ              เว็บไซต์/บริการ (remote)
เก็บ snapshot ใน .git/           เก็บสำเนา repo + เครื่องมือร่วมงาน
ทำงานได้ไม่มีเน็ต                ต้องมีเน็ต
ประวัติ = commit graph           PR, issue, Actions, review
```

**git ไม่ต้องมี GitHub ก็ใช้ได้** — และในบทนี้เราจะพิสูจน์ทั้งสองส่วน: git แบบ local 100% ก่อน แล้วค่อยต่อ GitHub

---

## 0. สถานะเครื่องนี้ (ตรวจก่อนเริ่ม)

```sh
$ git --version
git version 2.56.1
$ gh --version | head -1
gh version 2.104.0

$ git config --global user.name
$ git config --global user.email
# → ว่างทั้งคู่ (ยังไม่ตั้ง identity)

$ git config --global --list
fatal: unable to read config file '/root/.gitconfig': No such file or directory

$ gh auth status
You are not logged into any GitHub hosts. To log in, run: gh auth login
```

> `git`/`gh`/`ssh` บนเครื่องนี้เป็น binary ของ **Termux** (`/data/data/com.termux/files/usr/bin/`) ไม่ใช่ของ Debian — ใช้งานได้ปกติ แต่ `git help` จะพังเพราะไม่มี `man`:
> ```sh
> $ git help commit
> warning: failed to exec 'man': No such file or directory
> fatal: no man viewer handled the request
> ```
> ใช้ `git commit -h` (help แบบย่อ) หรือเว็บ docs แทน

---

## 1. config 3 ชั้น: system → global → local

| ชั้น | ไฟล์ | ใช้เมื่อ |
|---|---|---|
| system | `/etc/gitconfig` | ทั้งเครื่อง |
| **global** | `~/.gitconfig` | คุณคนเดียว ทุก repo |
| **local** | `.git/config` | repo นี้เท่านั้น (ชนะทุกชั้น) |

ดูว่าค่าไหนมาจากไฟล์ไหน:

```sh
$ git config --list --show-origin | head -6
file:.git/config	core.repositoryformatversion=0
file:.git/config	core.filemode=true
file:.git/config	core.bare=false
file:.git/config	core.logallrefupdates=true
file:.git/config	user.name=Lab User
file:.git/config	user.email=lab@example.com
```

**ตั้ง identity จริง** (ครั้งเดียวต่อเครื่อง — แก้เป็นชื่อ/อีเมลคุณ):

```sh
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
git config --global init.defaultBranch main
```

ทดลองใน HOME ชั่วคราวเพื่อไม่แตะของจริง:

```sh
$ HOME=/tmp/opencode/githome git config --global user.name "Demo User"
$ HOME=/tmp/opencode/githome git config --global init.defaultBranch main
$ cat /tmp/opencode/githome/.gitconfig
[user]
	name = Demo User
	email = demo@example.com
[init]
	defaultBranch = main

$ HOME=/tmp/opencode/githome git init -q sortedemo
$ HOME=/tmp/opencode/githome git -C sortedemo symbolic-ref --short HEAD
main
```

> ทำไมต้อง `init.defaultBranch main`: Git 2.56 ยังเตือนว่า default จะเปลี่ยนเป็น `main` ใน Git 3.0
> ```
> hint: Using 'master' as the name for the initial branch...
> ```

---

## 2. หัวใจ: 3 พื้นที่ของ git

```
 working tree            index (staging)           repository
 (ไฟล์จริง)     add →    (ของที่จะ commit)  commit →  (.git/commits)
     ↑                                                    │
     └──────────────── checkout / restore ────────────────┘
```

| สถานะ | หมายถึง | คำสั่งที่พาออก |
|---|---|---|
| untracked `??` | ไฟล์ใหม่ git ไม่รู้จัก | `git add` |
| modified (unstaged) | แก้แล้ว ยังไม่ stage | `git add` หรือ `git restore` |
| staged `A`/`M` | พร้อม commit | `git commit` |
| committed | อยู่ในประวัติแล้ว | — |
| conflicted `UU` | merge ชนกัน | แก้ไฟล์ + `git add` + `git commit` |

---

## 3. วงจรพื้นฐาน (ของจริง)

```sh
$ mkdir -p /tmp/opencode/gitlab && cd /tmp/opencode/gitlab
$ git init
Initialized empty Git repository in /tmp/opencode/gitlab/.git/

$ git config user.name "Lab User"
$ git config user.email "lab@example.com"

$ printf 'hello git\n' > hello.txt
$ git add hello.txt
$ git commit -m "add hello"
[master (root-commit) db39f46] add hello
 1 file changed, 1 insertion(+)
 create mode 100644 hello.txt

$ git log --oneline --decorate
db39f46 (HEAD -> master) add hello
```

**อ่าน `create mode 100644`:** นี่คือ permission จาก Linux บทที่ 3 — git เก็บ **executable bit** (`100644` vs `100755`) แต่ **ไม่เก็บ owner/group** (เพราะ repo ต้องย้ายข้ามเครื่องได้)

> `git add .` กวาดทุกอย่างรวมถึงไฟล์ที่ไม่ควร commit — ใช้ `git add -p` (เลือกทีละ hunk) หรือ `.gitignore` ก่อน

---

## 4. ดูความเปลี่ยนแปลงให้เป็น

```sh
$ printf 'line 3\n' >> notes.txt
$ git diff --stat
 notes.txt | 2 ++
 1 file changed, 2 insertions(+)

$ git diff
@@ -1,2 +1,4 @@
 line 1
 line 2
+work in progress
+line 3
```

| อยากดู | คำสั่ง |
|---|---|
| ยังไม่ stage | `git diff` |
| stage แล้ว | `git diff --staged` |
| ไฟล์เดียว | `git diff -- notes.txt` |
| ประวัติแบบกราฟ | `git log --oneline --graph --decorate --all` |
| ใครแก้บรรทัดนี้ | `git blame -L 1,2 hello.txt` |
| commit นี้เปลี่ยนอะไร | `git show <sha> --stat` |
| ไฟล์นี้เคยเปลี่ยนอะไร | `git log --oneline -- notes.txt` |

ผลจริงของ graph หลัง merge:

```sh
$ git log --oneline --graph --all
*   b2091aa merge: resolve hello conflict
|\
| * 5955cd0 left change
* | c0c9aec master change
|/
* 1b9a281 feat: notes
* db39f46 add hello
```

---

## 5. branch + merge (และ conflict ของจริง)

branch ใน git ไม่ใช่สำเนาโฟลเดอร์ — มันคือ **pointer 32–40 ไบต์** ชี้ไป commit เท่านั้น การสร้างจึงเร็วระดับ millisecond

```sh
$ git switch -c feature
Switched to a new branch 'feature'

$ printf 'line 1\nline 2\n' > notes.txt
$ git add notes.txt && git commit -m "feat: notes"
[feature 1b9a281] feat: notes

$ git switch master
$ git merge feature
Updating db39f46..1b9a281
Fast-forward                      # ← ไม่มี commit ใหม่ เพราะ master ไม่ได้ขยับ
```

### conflict เกิดขึ้นเมื่อไหร่

เมื่อ **สองสายแก้บรรทัดเดียวกัน**:

```sh
$ git switch -c left
$ printf 'hello from LEFT\n' > hello.txt && git commit -am "left change"
$ git switch master
$ printf 'hello from MASTER\n' > hello.txt && git commit -am "master change"

$ git merge left
Auto-merging hello.txt
CONFLICT (content): Merge conflict in hello.txt
Automatic merge failed; fix conflicts and then commit the result.

$ git status --short
UU hello.txt

$ cat hello.txt
<<<<<<< HEAD
hello from MASTER
=======
hello from LEFT
>>>>>>> left
```

**วิธีแก้ที่ถูก**

```sh
# 1) แก้ไฟล์ให้เหลือเนื้อหาที่ต้องการ (ลบ marker ทั้ง 3 ตัว)
$ printf 'hello from BOTH (resolved)\n' > hello.txt
# 2) tell git ว่าแก้แล้ว
$ git add hello.txt
# 3) ปิด merge
$ git commit -m "merge: resolve hello conflict"
[master b2091aa] merge: resolve hello conflict
```

ตัวช่วย: `git checkout --ours hello.txt` (เอาฝั่งเรา) / `--theirs` (เอาฝั่งเขา) / `git mergetool`

> ถ้าเลิกกลางทาง: `git merge --abort`

---

## 6. บันไดการย้อนกลับ (สำคัญที่สุดในบทนี้)

| สถานะ | คำสั่ง | ผล |
|---|---|---|
| แก้ไฟล์ ยังไม่ add | `git restore file` | ทิ้งการแก้ |
| add แล้ว ยังไม่ commit | `git restore --staged file` | ถอดออกจาก staging (ไฟล์ยังอยู่) |
| commit แล้ว (local) | `git reset --soft HEAD~1` | ย้อน commit, เก็บ staged |
| | `git reset HEAD~1` | ย้อน commit, ถอด staged (default = mixed) |
| | `git reset --hard HEAD~1` | **ทิ้งงานทั้งหมด** |
| push แล้ว | `git revert <sha>` | สร้าง commit ใหม่ที่ย้อนผล |
| merge commit | `git revert -m 1 <sha>` | ต้องบอก parent หลัก |
| หายไปแล้ว | `git reflog` + `git reset --hard HEAD@{n}` | กู้คืน |

**หลักฐานจริง**

```sh
# ถอด staging
$ git add temp.txt && git status --short
A  temp.txt
$ git restore --staged temp.txt && git status --short
?? temp.txt

# reset --hard แล้วกู้คืนด้วย reflog
$ git log --oneline -1
4078478 chore: gitignore
$ git reset --hard HEAD~1
HEAD is now at b2091aa merge: resolve hello conflict
$ git reset --hard HEAD@{1}
HEAD is now at 4078478 chore: gitignore          # ← กลับมาแล้ว
$ ls .gitignore
.gitignore
```

**revert merge ต้องมี `-m`** — ของจริงคือ error:

```sh
$ git revert --no-edit HEAD
error: commit b2091aac... is a merge but no -m option was given.
fatal: revert failed
```

`-m 1` = "ถือว่า parent ที่ 1 (master) คือสายหลัก แล้วย้อนสิ่งที่ merge เข้ามา"

> `git reflog` เก็บประวัติการขยับ HEAD ~90 วัน — เป็นตาข่ายนิรภัยที่คนส่วนใหญ่ไม่รู้ว่ามี
> `reset --hard` ไม่ได้ลบ commit ทันที; มันแค่เลื่อน pointer (ของจริงจะถูก gc ภายหลัง)

---

## 7. stash, amend, tag

```sh
# stash: พักงานที่ยังไม่พร้อม
$ git stash
Saved working directory and index state WIP on master: b2091aa ...
$ git stash pop
Dropped refs/stash@{0}

# amend: แก้ commit ล่าสุด (เปลี่ยน message หรือเพิ่มไฟล์)
$ git add notes.txt
$ git commit --amend --no-edit
[master 528cf15] Revert "chore: gitignore"        # ← SHA เปลี่ยน!
$ git reflog | head -1
528cf15 HEAD@{0}: commit (amend): Revert "chore: gitignore"

# tag: ปักหมุดเวอร์ชัน
$ git tag -a v0.1.0 -m "first lab tag"
$ git tag -n1
v0.1.0          first lab tag
```

> ⚠️ `--amend` **เขียนประวัติใหม่** (SHA เปลี่ยน) — ห้ามทำกับ commit ที่ push ไปแล้วและมีคนอื่นใช้ เว้นแต่จะ force push แบบระวัง (หัวข้อ 9)

---

## 8. remote: GitHub คือ "remote" ตัวหนึ่ง

เราพิสูจน์ได้โดยไม่ต้องใช้ GitHub เลย — ใช้ bare repo ในเครื่องเป็น remote:

```sh
$ git init --bare /tmp/opencode/remote.git

$ cd /tmp/opencode/gitlab
$ git remote add origin /tmp/opencode/remote.git
$ git push -u origin master
To /tmp/opencode/remote.git
 * [new branch]      master -> master
branch 'master' set up to track 'origin/master'.

$ cd /tmp/opencode && git clone -q remote.git clone-lab
$ cd clone-lab && git log --oneline | head -2
b2091aa merge: resolve hello conflict
c0c9aec master change
```

**คำศัพท์ที่ต้องแยก**

| คำ | หมายถึง |
|---|---|
| `origin` | ชื่อเล่นของ remote (ตั้งชื่ออะไรก็ได้) |
| `origin/master` | **remote-tracking branch** = สำเนาว่า remote อยู่ตรงไหน |
| `git fetch` | ดึงข้อมูลใหม่ (ไม่แตะไฟล์คุณ) |
| `git pull` | `fetch` + `merge` (หรือ rebase) |
| `ahead 1, behind 1` | local กับ remote แยกกันแล้ว |

```sh
$ git status -sb
## master...origin/master [ahead 1, behind 1]
```

### force push — อันตรายแต่บางครั้งจำเป็น

หลัง amend, local กับ remote แยกกัน → push ปกติถูก reject:

```sh
$ git push
 ! [rejected]        master -> master (non-fast-forward)
hint: Updates were rejected because the tip of your current branch is behind
```

ทางที่ปลอดภัยกว่าคือ **`--force-with-lease`** (ยอมทับเฉพาะเมื่อ remote ยังอยู่ที่ที่เราคิด):

```sh
$ git push --force-with-lease
 + 7959408...528cf15 master -> master (forced update)
```

| ธง | พฤติกรรม |
|---|---|
| `--force` | ทับทันที ไม่สนใจว่าใคร push อะไรไป |
| `--force-with-lease` | ทับได้ถ้า remote ไม่ได้ขยับตั้งแต่เราดึงล่าสุด |

---

## 9. `.gitignore` — และกับดักที่คนพลาด

```sh
$ printf '*.log\nbuild/\n.env\n' > .gitignore
$ printf 'noise\n' > debug.log
$ git status --short
?? .gitignore          # ← debug.log ไม่โผล่ = ถูก ignore แล้ว
```

| pattern | หมายถึง |
|---|---|
| `*.log` | ทุกไฟล์ลงท้าย .log (ทุกโฟลเดอร์) |
| `build/` | โฟลเดอร์ build |
| `!keep.log` | ยกเว้น (ต้องมากหลัง rule ที่ ignore) |
| `/secret` | เฉพาะ root ของ repo |

**กับดัก:** `.gitignore` ไม่มีผลกับไฟล์ที่ **track อยู่แล้ว**

```sh
git rm --cached .env      # เลิก track แต่เก็บไฟล์ไว้
git commit -m "chore: stop tracking .env"
```

**ไฟล์ที่ห้าม commit เด็ดขาดในโปรเจกต์นี้**

```
~/.config/opencode/service.json      # token (mode 600)
~/.local/share/opencode/opencode.db  # ประวัติ session
.env, *.pem, id_ed25519
```

ถ้าหลุดไปแล้ว การลบใน commit ใหม่ **ไม่พอ** — ต้องถือว่า secret นั้นรั่ว แล้ว rotate ทันที (ประวัติ git ยังมีอยู่)

---

## 10. GitHub: ส่วนที่ต้อง login

ตอนนี้ `gh` ยังไม่ผูกบัญชี — และ **ผม login แทนคุณไม่ได้** (ต้องยืนยันตัวตนกับ GitHub)

```sh
$ gh auth status
You are not logged into any GitHub hosts.

$ gh auth login
# → เลือก GitHub.com → HTTPS → Login with a web browser
# → ได้ code 8 หลัก → เปิด https://github.com/login/device → ใส่ code
```

ทางเลือกสำหรับเครื่อง headless/ไม่มี browser:

```sh
gh auth login --with-token < token.txt    # Personal Access Token (scopes: repo, read:org)
gh auth login --web                       # พยายามเปิด browser
gh auth setup-git                         # ให้ git ใช้ credential จาก gh
```

ตรวจว่าสำเร็จ:

```sh
gh auth status
# ✓ Logged in to github.com account <you> (keyring)

git config --global --list | grep -i github
```

### สร้าง repo + push ครั้งแรก

```sh
cd /tmp/opencode/gitlab
gh repo create my-lab --private --source=. --remote=origin --push
```

`--source=.` = ใช้ repo local ที่มีอยู่, `--push` = push ให้เลย  
(ถ้าไม่มี `--source` มันจะสร้าง repo เปล่าแล้ว clone ลงมาแทน)

### วงจรทำงานร่วมกันแบบ PR

```sh
git switch -c feat/hello
# ...แก้โค้ด...
git add -A && git commit -m "feat: add greeting"
git push -u origin feat/hello

gh pr create --fill                 # ใช้ชื่อ commit เป็น title/body
gh pr view --web                    # เปิดใน browser
gh pr checks                        # ดู CI
gh pr merge --squash --delete-branch
```

| คำสั่ง | ใช้ |
|---|---|
| `gh repo view --web` | เปิด repo |
| `gh issue list` / `gh issue create` | งาน/บั๊ก |
| `gh run list` / `gh run watch` | GitHub Actions |
| `gh release create v0.1.0` | release + asset |
| `gh api repos/{owner}/{repo}` | ยิง REST ตรง ๆ |

### กฎที่ควรตั้งตั้งแต่แรก

| สิ่งที่ทำ | เพราะ |
|---|---|
| Protect `main`: ห้าม push ตรง, ต้องมี PR + review | กันประวัติพัง |
| เปิด secret scanning | กัน token หลุด |
| `.gitignore` ตั้งแต่ commit แรก | ลบทีหลังเจ็บ |
| branch name: `feat/`, `fix/`, `chore/` | อ่าน log รู้เรื่อง |
| commit message: imperative + why | `git log` มีประโยชน์จริง |
| ไม่ commit binary ใหญ่ | repo บวมถาวร (git เก็บทุกเวอร์ชัน) |

---

## 11. caveat ของเครื่องนี้ (ทดสอบแล้ว)

| เรื่อง | ผลจริง |
|---|---|
| binary | `git`/`gh`/`ssh` มาจาก Termux ไม่ใช่ Debian |
| `git help` | พัง — ไม่มี `man` |
| global config | ยังไม่มีไฟล์ `~/.gitconfig` (จะถูกสร้างเมื่อตั้งครั้งแรก) |
| `gh` auth | ยังไม่ login — ต้องทำเองด้วยบัญชีคุณ |
| filesystem | อยู่บน Android storage ผ่าน proot — ช้าและ sensitive ต่อ permission |
| `git status` ใน `/tmp` | ปกติ (ไม่มีปัญหา dubious ownership เพราะรันเป็น root คนเดียว) |

---

## 12. กับดัก 8 ข้อ

| กับดัก | อาการ | ทางแก้ |
|---|---|---|
| commit secret | token อยู่ในประวัติตลอดไป | `.gitignore` ก่อน + rotate ถ้าหลุด |
| `git add .` มั่ว | ไฟล์ขยะ/ไฟล์ใหญ่เข้า repo | `git add -p`, `.gitignore` |
| amend/rebase ของที่ push แล้ว | คนอื่น pull ไม่ได้ | ทำบน branch ตัวเอง, ใช้ `--force-with-lease` |
| `reset --hard` พลาด | งานหาย | `git reflog` กู้ได้ |
| revert merge ไม่ใส่ `-m` | error (ของจริงในบทนี้) | `git revert -m 1 <sha>` |
| conflict แล้ว commit ทั้ง marker | โค้ดพัง | ลบ `<<<<<<<` ให้หมดก่อน `git add` |
| `.gitignore` ไม่ทำงาน | ไฟล์ยังโผล่ใน status | `git rm --cached` |
| detached HEAD | commit แล้วหลุด | `git switch -c rescue` ทันที |

---

## แบบฝึกหัด (lab อยู่ที่ `/tmp/opencode/gitlab`)

```sh
cd /tmp/opencode/gitlab

# 1) ตั้ง global identity + default branch เป็น main (ครั้งเดียว)
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
git config --global init.defaultBranch main

# 2) สร้าง repo ใหม่ชื่อ practice ที่ /tmp/opencode/practice
#    - ไฟล์ README.md
#    - commit 2 ครั้ง
#    - แตก branch แล้ว merge แบบ fast-forward

# 3) สร้าง conflict แล้วแก้ให้ผ่าน (แก้ hello.txt สองสาย)

# 4) ทดลอง reset --hard 1 ครั้ง แล้วกู้คืนด้วย reflog

# 5) สร้าง bare remote ใหม่ แล้ว push + clone + pull

# 6) ใส่ .gitignore ให้ ignore *.tmp และโฟลเดอร์ bin/
#    แล้วพิสูจน์ว่าไฟล์ที่ track อยู่แล้วยังต้อง rm --cached

# 7) (ต้อง login) gh repo create practice --private --source=. --push
```

**เฉลยข้อ 2–4 (ย่อ)**

```sh
# 2
mkdir -p /tmp/opencode/practice && cd /tmp/opencode/practice
git init
printf '# practice\n' > README.md
git add README.md && git commit -m "docs: init readme"
printf 'second\n' >> README.md && git commit -am "docs: add second line"
git switch -c feat/a && printf 'a\n' > a.txt && git add a.txt && git commit -m "feat: a"
git switch main && git merge feat/a          # Fast-forward

# 3
git switch -c left && printf 'L\n' > c.txt && git add c.txt && git commit -m "left"
git switch main  && printf 'M\n' > c.txt && git add c.txt && git commit -m "main"
git merge left                                # CONFLICT
printf 'LM\n' > c.txt && git add c.txt && git commit -m "merge: resolve"

# 4
git reset --hard HEAD~1
git reflog | head -3
git reset --hard HEAD@{1}                     # กู้คืน
```

---

## เช็คความเข้าใจ 3 ข้อ

1. `git reset --soft` กับ `--hard` ต่างกันที่อะไร และแบบไหนกู้ง่ายกว่าถ้าพลาด?
2. ทำไม `git revert` ถึงปลอดภัยกว่า `git reset` บน branch ที่ push แล้ว?
3. `.gitignore` ใส่ `.env` แล้ว แต่ `git status` ยังเห็น `.env` — เพราะอะไร?

---

## สรุปบทที่ 1

```
working tree --add--> index --commit--> history
                <--restore--     <--reset/revert--

branch = pointer ไป commit (ถูกมาก)
merge  = รวมสองสาย; ชนกัน → แก้ marker → add → commit
remote = repo อีกที่หนึ่ง (GitHub คือ remote ตัวหนึ่ง)
reflog = ตาข่ายนิรภัยเมื่อพลาด
```

**3 นิสัยที่ควรสร้าง**

1. `git status` + `git diff` ก่อน `add` ทุกครั้ง
2. commit เล็ก ๆ บ่อย ๆ อ่านง่ายกว่า commit ใหญ่ครั้งเดียว
3. secret ห้ามเข้า repo — ตรวจ `.gitignore` ก่อน commit แรก
