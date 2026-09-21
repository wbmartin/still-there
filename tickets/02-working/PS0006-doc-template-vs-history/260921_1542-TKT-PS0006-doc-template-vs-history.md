# 2026-09-21 — PS0006 doc/ is template-only

Walked `doc/000-TODO.md` with Brandon: four tickets (PS0016–19), two
folds, three drops, nine "thought complete" items confirmed in code.
Ten history files into the engineering log with git dates; the three
package READMEs were describing another app and were regenerated from
`main.go` / `config.go` / the tree by a subagent, then diffed against the
code before commit.

Two finds: `APP_NAME="Proto App"` keeps its quotes all the way to the
TOTP issuer (PS0019); `tests/conftest.py` still has a `/auth2` branch
that nothing can reach (PS0014 note).

## Prompts preserved from doc/000-TODO.md

**Starting prompt**

> make sure you look at my styles and quirks in the instructions. we are
> building a template/initial scaffold CRUD app in Go and React. in the
> doc file is an 810 file that describes how to construct a new entity,
> we're currently building the first entity, "example", and documenting
> what's important to facilitate future new entity creation for apps
> built on this scaffolding.

**Code review prompt** (works well for one AI to review what you have
been doing with another)

> Please perform a detailed code review of the API Server in the
> src/api-srv folder.
> - Background: This is a golang server project that will serve as a
>   starter/base scaffold project for future projects. It is intended to
>   be paired with the web-app react app in src/web-app. The goal is to
>   have a well structured, secure, and maintainable codebase that
>   follows best practices for golang web servers.
> - Focus on inconsistencies, best practices, security, dead code, and
>   note any tradeoffs made.
> - Don't look at the documentation in the doc folder because it may
>   bias the review.
> - Output all of your results to doc/111-api-server-code-review-codex.md.
>
> Please perform a detailed code review of the WEB APP in the src/web-app
> folder.
> - Background: This is a react project that will serve as a starter/base
>   scaffold project for future projects. It is intended to be paired
>   with the api-srv in the sibling folder.
> - Focus on inconsistencies, best practices, security, dead code, and
>   note any tradeoffs made.
> - Don't look at the documentation in the doc folder because it may
>   bias the review.
> - Output all of your results to doc/215-react-web-code-review.md.
