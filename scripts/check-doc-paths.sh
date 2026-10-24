#!/usr/bin/env bash
# Every repo path a document names in backticks must exist.
#
# Reads the backticked tokens out of the documents (DOC_LIST), keeps the
# ones that look like repo paths, and resolves each one. Resolution is
# tolerant, because documents name files the way an engineer does:
#   - a short name resolves by prefix  (`bin/215` -> bin/215-run-api-srv,
#     `doc/105` -> doc/105-api-srv-tech-spec.md)
#   - a package-relative name resolves against a package root (BASE_LIST:
#     `internal/auth/jwt.go` -> src/api-srv/internal/auth/jwt.go,
#     `bin/prd-stillthere-install` -> infra/prd-wbm/bin/prd-stillthere-install)
#   - a bare basename resolves anywhere in the tree (`conftest.py`)
# Anything else is reported with the file and line that names it.
#
# A token counts as a repo path when it ends in one of this repo's file
# extensions or its first segment is a top-level directory. That keeps Go
# import paths (`net/http`) and media types (`application/json`) out.
#
# Skipped: URLs, absolute or ~ paths (a host's /srv/... or the reader's
# checkout), placeholders (`<entity>`, `NNNN`, `5xx`, `*`, `{a,b}`, `...`),
# anything with `=` or `$` or a space, and the gitignored runtime
# directories (RUNTIME_PATTERN).
#
# A document that names a deleted file says so in prose, without
# backticks: the rule is literally "every backticked repo path exists".
# A name that is real but not in this tree (a file inside the deploy
# bundle, a template's copy target) goes in scripts/doc-paths-allow.txt
# with the reason.
#
# Usage: scripts/check-doc-paths.sh    (any cwd; exit 1 = a miss)
# Run by bin/300-run-tests docs (and all). Bash 3.2 compatible (macOS).
set -euo pipefail
export LC_ALL=C
cd "$(dirname "${BASH_SOURCE[0]}")/.." || exit 2

ALLOW_FILE=scripts/doc-paths-allow.txt
DOC_LIST=()
for doc in CLAUDE.md DERIVING.md README.md doc/*.md infra/README.md \
        src/*/README.md src/*/*/README.md tests/README.md; do
    [ -f "$doc" ] || continue
    DOC_LIST+=("$doc")
done
BASE_LIST=(. src src/api-srv src/api-srv/internal src/api-srv/sql
    src/web-app tests doc bin infra/prd-wbm)
TOPLEVEL_PATTERN='^(doc|src|bin|tests|etc|infra|scripts|tasks)/'
# Written at run time and gitignored: present on a machine that has run the
# gate, absent on a fresh clone, so their absence is not a doc bug.
# etc/crt/ is the local TLS pair bin/400 writes.
RUNTIME_PATTERN='^(tmp/|dist/|tests/\.venv|src/web-app/node_modules'
RUNTIME_PATTERN+='|src/web-app/dist|src/web-app/coverage'
RUNTIME_PATTERN+='|(src/web-app/)?tests/e2e/\.auth|etc/crt/)'
ELLIPSIS=$'\xe2\x80\xa6'

miss_count=0
checked_count=0

resolve() {
    local path="$1" base
    for base in "${BASE_LIST[@]}"; do
        [ -e "$base/$path" ] && return 0
        compgen -G "$base/$path"'*' > /dev/null && return 0
    done
    case "$path" in
        */*) ;;
        *) [ -n "$(find . -name "$path" -not -path './.git/*' \
                -not -path '*/node_modules/*' -print -quit)" ] \
                && return 0 ;;
    esac
    return 1
}

is_allowed() {
    grep -qxF -- "$1:$2" "$ALLOW_FILE"
}

for doc in "${DOC_LIST[@]}"; do
    # shellcheck disable=SC2016  # literal backticks in the grep pattern
    while IFS=: read -r line token; do
        [ -n "${token:-}" ] || continue
        case "$token" in
            http*|/*|~*|*'<'*|*'>'*|*'{'*|*'*'*|*' '*|*'$'*|*=*) continue ;;
            *NNN*|*nnn*|*'...'*|*"$ELLIPSIS"*|*[0-9]xx*) continue ;;
        esac
        case "$token" in
            *.md|*.go|*.ts|*.tsx|*.sql|*.py|*.json|*.mjs|*.env|*.sh|\
            *.yml|*.yaml|*.css|*.pem|*.conf|*.txt|*.html) ;;
            *) [[ $token =~ $TOPLEVEL_PATTERN ]] || continue ;;
        esac
        [[ $token =~ $RUNTIME_PATTERN ]] && continue
        checked_count=$((checked_count + 1))
        resolve "$token" && continue
        is_allowed "$doc" "$token" && continue
        echo "MISS  $doc:$line  $token"
        miss_count=$((miss_count + 1))
    done < <(grep -noE '`[^`]+`' "$doc" | sed 's/`//g' || true)
done

# An entry whose document no longer names the token is dead weight that
# would hide the next real miss of the same name; it fails too.
while IFS= read -r entry; do
    case "$entry" in ''|'#'*) continue ;; esac
    doc="${entry%%:*}"
    token="${entry#*:}"
    if [ ! -f "$doc" ] || ! grep -qF -- "\`$token\`" "$doc"; then
        echo "STALE $ALLOW_FILE  $entry"
        miss_count=$((miss_count + 1))
    fi
done < "$ALLOW_FILE"

echo "checked $checked_count path tokens in ${#DOC_LIST[@]} documents," \
    "$miss_count missing"
[ "$miss_count" -eq 0 ] || exit 1
