# Repository reset — the tree as one root commit in a new repository

When something that must never be in git reached its history (customer
data, `DERIVING.md` section 2), deleting the file does not remove it:
every earlier commit still carries it, and a history rewrite in place
leaves the old commits reachable by sha through forks and pull request
refs. The fix is a new repository whose first commit is the current tree,
and every checkout cloned again. This runbook is that, without breaking
the post-sales deploy (`infra/README.md`).

**Not yet run.** The first derived app planned this on 2026-10-02 and had
not run it when this was written. The git commands were checked on a
scratch repository; the GitHub settings steps and the host steps were
not. Tick each step off as it runs and put what differed in **Lessons**.

Local blocks run in your terminal from the repo root and carry no inline
`#` comments (zsh, `doc/330` section 0). Host blocks run in bash as root
(`sudo -i`). The branch UAT builds is `BRANCH_TO_CHECKOUT`, `main` in the
template; the blocks say `main`.

## What hangs off the repository

A GitHub repository is the object these are attached to; a new one has
none of them.

| Thing | Set up in | After the reset |
|---|---|---|
| UAT build checkout's read-only deploy key | `doc/330` A3 | add the same public key to the new repository |
| UAT build webhook (push → `uat-stillthere-build`) | `doc/330` B1 | re-create: same payload URL, same `HOOK_SECRET` |
| Prod promote webhook, if there is one | `doc/331` B6 | re-create the same way |
| Collaborators, team access, branch protection, Actions secrets | GitHub settings | re-create by hand from the record in 0 |
| Every clone: UAT's `BUILD_STG`, each laptop | — | cloned again; an old clone holds the old objects |

Unchanged when the new repository keeps the old owner and name:
`REPO_URL` in `etc/uat-stillthere-deploy.env`, the `github-stillthere` ssh alias,
the hook files under `/srv/shrmpl/etc/hooks/` and every doc path. Prod has
no checkout (`doc/331`), so nothing on prod is cloned. Commit shas quoted
in docs resolve only in the old repository; leave them, they are history.
The registry is not covered here: if a build ever carried the data (a web
bundle built with mock data from real records), its `sha-<commit>` tags
hold it too and are removed on the registry's side.

## 0. Before

- **The data is out of the tree** in the commit you re-root, and lives
  where `DERIVING.md` section 2 says. That commit is released: UAT built
  and passed it, and prod runs it, so both hosts already run the tree the
  root commit will carry.
- **A quiet moment.** No push to `main` (or a prod release branch) from
  anyone until 5.
- **Record** what exists, from the GitHub settings pages: each webhook's
  payload URL, the deploy key's fingerprint, collaborators and team
  access, branch protection rules, Actions secrets by name. Carry open
  pull requests and issues by hand if any matter. The deploy key's
  fingerprint on UAT, to compare:

```bash
ssh-keygen -lf ~shrmpl-hook/.ssh/stillthere-deploy.pub
```

- **What each host runs now**, the rollback reference. On UAT and on
  prod:

```bash
cat /srv/*-stillthere/libexec/stillthere-api.revision
```

## 1. The root commit (local)

From a clone that has the released `main`. The commit message names the
old head, the only link back to the old history.

```bash
git fetch origin
OLD=$(git rev-parse origin/main); echo "OLD=$OLD"
git checkout --orphan fresh "${OLD:?}"
git commit -m "Fresh root at ${OLD:?}"
[ "$(git rev-parse "fresh^{tree}")" = "$(git rev-parse "${OLD:?}^{tree}")" ] && echo "same tree"
git rev-list --count fresh
```

Expect `same tree` and `1`.

## 2. GitHub: the old repository aside, the new one empty

1. Old repository → Settings → rename it (`<name>-old`). Delete its
   webhooks, so nothing fires from it, keep it private, and archive it.
   GitHub redirects the old name to the renamed repository until a new
   one takes the name, so do step 2 straight after.
2. Create the repository again under the same owner and name: private,
   empty (no README, licence or `.gitignore`).
3. Before the first push, re-create on it:
   - **Deploy key**, read-only: the public key from UAT
     (`cat ~shrmpl-hook/.ssh/stillthere-deploy.pub`); its fingerprint matches
     0.
   - **UAT webhook**: payload URL as recorded in 0, content type
     `application/json`, *just the push event*, and the secret from the
     host file. On UAT:

```bash
sed -n 's/^HOOK_SECRET=//p' /srv/shrmpl/etc/hooks/build-stillthere_*.env
```

   - **Prod webhook**, if prod had one: the same, with
     `/srv/shrmpl/etc/hooks/prd-promote-stillthere_*.env` on prod.
   - Collaborators, team access, branch protection and Actions secrets
     from the record in 0.
4. The deploy key reaches the new repository. On UAT (an empty
   repository prints nothing and exits 0; a key problem prints
   `Permission denied (publickey)`):

```bash
REPO_URL=$(sed -n 's/^REPO_URL=//p' /srv/uat-stillthere/etc/uat-stillthere-deploy.env)
: "${REPO_URL:?}"
sudo -u shrmpl-hook -H git ls-remote "${REPO_URL:?}"; echo "exit $?"
```

## 3. Push the root commit

Push only `fresh`, by name, from the clone that made it: any other branch
of an old clone carries the old history with it.

```bash
git push origin fresh:main
git rev-parse fresh
```

The push builds UAT (`infra/README.md` Promotion flow): Slack *started*,
*deployed*, *tests passed*, each naming the root commit's full sha, and
the webhook's delivery log on GitHub shows the push answered. The build
runs in UAT's old checkout, which fetches the unrelated history and
resets to it; 4 replaces that checkout.

Then put the root commit on prod. With a prod webhook on its own release
branch, push it there once UAT says *tests passed*
(`git push origin fresh:<release-branch>` with the real name), which also
proves that webhook. Without one, on prod, with `ROOT_SHA` set to the full
sha `git rev-parse fresh` printed:

```bash
: "${ROOT_SHA:?}"
sudo -u shrmpl-hook -H /srv/prd-stillthere/bin/prd-stillthere-promote-uat-and-deploy "${ROOT_SHA:?}"
```

If the prod webhook is on `main` too, the push above fired it while UAT
was still building: it re-promoted the previous build (the same tree) or
refused (`doc/331` B6), and prod runs an identical tree either way.
Promote the root commit by hand as above.

## 4. Clone everything again

**UAT's build checkout**, once 3's build has finished (it holds the
build lock). The old checkout is deleted only after the new one checks
out, because until then it is the only copy on the host:

```bash
E=/srv/uat-stillthere/etc/uat-stillthere-deploy.env
BUILD_STG=$(sed -n 's/^BUILD_STG=//p' "${E:?}"); REPO_URL=$(sed -n 's/^REPO_URL=//p' "${E:?}")
: "${BUILD_STG:?}" "${REPO_URL:?}"
mv "${BUILD_STG:?}" "${BUILD_STG:?}.old"
mkdir -p "${BUILD_STG:?}" && chown shrmpl-hook:shrmpl-hook "${BUILD_STG:?}"
sudo -u shrmpl-hook -H git clone "${REPO_URL:?}" "${BUILD_STG:?}"
sudo -u shrmpl-hook -H git -C "${BUILD_STG:?}" rev-list --all --count
```

Expect `1`, then `rm -rf "${BUILD_STG:?}.old"`.

**Every laptop and other machine:** a fresh clone beside the old one.
Carry uncommitted work across as files; never push from the old clone or
merge one of its branches, either of which brings the old history back.
Delete the old clone. The data directory outside the repository stays
where it is; a machine that needs it gets it copied directly, never
through git.

## 5. Checks

- GitHub's delivery log shows each webhook answering 3's pushes; Slack
  shows UAT *tests passed* and prod deployed for the root commit.
- `cat /srv/*-stillthere/libexec/stillthere-api.revision` on each host names the
  root commit.
- In a fresh clone, `git rev-list --all --count` is `1` plus any commits
  since, and `git log --all --oneline -- <the leaked path>` prints
  nothing.
- The old repository is private, archived and has no webhooks. Delete it
  once the new one has carried one full release (UAT build, tests
  passed, prod promote).
- Lift the freeze.

## Rollback

Until 4, nothing on the hosts depends on the new repository but the
webhooks and the deploy key: rename the new repository aside, rename the
old one back, and re-create its webhooks. Both hosts run the same tree
throughout, so no host step is undone. After 4, re-clone UAT's checkout
from whichever repository holds the name.

## Lessons

None yet: this has not run. Add what differed.
