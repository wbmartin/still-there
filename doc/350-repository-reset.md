# Repository reset — the tree as one root commit in a new repository

When something that must never be in git reached its history (customer
data, `DERIVING.md` section 2), deleting the file does not remove it:
every earlier commit still carries it, and a history rewrite in place
leaves the old commits reachable by sha through forks and pull request
refs. The fix is a new repository whose first commit is the current tree,
and every checkout cloned again.

**Not yet run.** The git commands were checked on a scratch repository;
the GitHub settings steps were not. Tick each step off as it runs and put
what differed in **Lessons**.

Nothing on the host is attached to the GitHub repository: the host has no
checkout, no webhook and no deploy key, and a deploy builds from the
laptop's clean checkout (doc/320). So the reset touches GitHub and the
laptops only, and the host keeps running its current release throughout.

Blocks run in your terminal from the repo root and carry no inline `#`
comments (zsh).

## What hangs off the repository

| Thing | After the reset |
|---|---|
| Collaborators, team access, branch protection, Actions secrets | re-create by hand from the record in 0 |
| Open pull requests and issues | carried by hand if any matter |
| Every clone (each laptop) | cloned again; an old clone holds the old objects |

Commit shas quoted in docs resolve only in the old repository; leave
them, they are history. The release on the host records the commit it
was built from in `stillthere-api.revision`; after the reset that sha names a
commit only the old repository holds, which is accurate and harmless.

## 0. Before

- **The data is out of the tree** in the commit you re-root, and lives
  where `CLAUDE.md` says (customer data never in git). That commit is
  the release the host runs, so the root commit carries the tree the
  host already runs.
- **A quiet moment.** No push to `main` from anyone until 5.
- **Record** what exists, from the GitHub settings pages: collaborators
  and team access, branch protection rules, Actions secrets by name.

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

1. Old repository → Settings → rename it (`<name>-old`), keep it
   private, and archive it. GitHub redirects the old name to the renamed
   repository until a new one takes the name, so do step 2 straight
   after.
2. Create the repository again under the same owner and name: private,
   empty (no README, licence or `.gitignore`).
3. Before the first push, re-create on it the collaborators, team
   access, branch protection and Actions secrets from the record in 0.

## 3. Push the root commit

Push only `fresh`, by name, from the clone that made it: any other branch
of an old clone carries the old history with it.

```bash
git push origin fresh:main
git rev-parse fresh
```

## 4. Clone everything again

**Every laptop and other machine:** a fresh clone beside the old one.
Carry uncommitted work across as files; never push from the old clone or
merge one of its branches, either of which brings the old history back.
Delete the old clone. The data directory outside the repository stays
where it is; a machine that needs it gets it copied directly, never
through git.

## 5. Checks

- In a fresh clone, `git rev-list --all --count` is `1` plus any commits
  since, and `git log --all --oneline -- <the leaked path>` prints
  nothing.
- `git status` in the fresh clone is clean and `bin/600-deploy-prd
  --check` still reads the live release (the deploy needs nothing from
  the repository's history).
- The old repository is private and archived. Delete it once the new
  one has carried one full release.
- Lift the freeze.

## Rollback

Until the old clones are deleted, rename the new repository aside and
rename the old one back. The host is untouched either way, so no host
step is undone.

## Lessons

None yet: this has not run. Add what differed.
