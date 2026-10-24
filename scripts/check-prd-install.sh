#!/usr/bin/env bash
# Runs infra/prd-wbm/bin/prd-stillthere-install and prd-stillthere-rollback against a
# fake app root under tmp/check-prd-install/, with no root, systemd,
# Postgres or network, and checks what each run leaves behind.
#
# The scripts are the committed files, unchanged, copied into
# <fake>/srv/prd-stillthere/bin/, so they derive the fake root from their own
# location as they derive /srv/prd-stillthere on the host. They read the
# committed etc/prd-stillthere-deploy.env and etc/prd-stillthere-migrate.env, with only
# HEALTH_WAIT_SECONDS cut to 1. The host commands a laptop cannot run as
# the host does are stubs first on PATH, each logging its arguments:
#   id         answers the uid in state/uid (0 = root)
#   systemctl  logs only
#   curl       "status":"ok" unless the live libexec/api-srv says broken
#   runuser    logs, writes a fake dump to stdout (pg_dump)
#   flock      always gets the lock
# The bundles' own api-srv and stillthere-migrate are shell stubs: stillthere-migrate
# answers verify from state/verify and pending from state/pending, and
# up zeroes it.
#
# Usage: scripts/check-prd-install.sh   (any cwd; exit 1 = a failed check)
# Run by bin/300-run-tests infra (and all). Bash 3.2 compatible (macOS).
set -euo pipefail
export LC_ALL=C
cd "$(dirname "${BASH_SOURCE[0]}")/.." || exit 2
PROJECT_ROOT="$PWD"
HOST_DIR="$PROJECT_ROOT/infra/prd-wbm"

if command -v sha256sum > /dev/null; then
    SHA256=(sha256sum)
else
    echo "[ERROR] sha256sum not on PATH (the install script needs it)"
    exit 1
fi

WORK="$PROJECT_ROOT/tmp/check-prd-install"
ROOT="$WORK/srv/prd-stillthere"
STUB="$WORK/stub-bin"
STATE="$WORK/state"
CALLS="$STATE/calls.log"
rm -rf "$WORK"
mkdir -p "$ROOT/bin" "$ROOT/etc" "$ROOT/incoming" "$STUB" "$STATE"
cp "$HOST_DIR/bin/prd-stillthere-install" "$HOST_DIR/bin/prd-stillthere-rollback" \
    "$ROOT/bin/"
sed 's/^HEALTH_WAIT_SECONDS=.*/HEALTH_WAIT_SECONDS=1/' \
    "$HOST_DIR/etc/prd-stillthere-deploy.env" > "$ROOT/etc/prd-stillthere-deploy.env"
cp "$HOST_DIR/etc/prd-stillthere-migrate.env" "$ROOT/etc/"
echo 0 > "$STATE/uid"
echo ok > "$STATE/verify"
echo 0 > "$STATE/pending"
: > "$CALLS"

cat > "$STUB/id" <<EOF
#!/bin/sh
cat "$STATE/uid"
EOF
cat > "$STUB/systemctl" <<EOF
#!/bin/sh
echo "systemctl \$*" >> "$CALLS"
EOF
cat > "$STUB/curl" <<EOF
#!/bin/sh
grep -q broken "$ROOT/libexec/api-srv" 2>/dev/null && exit 7
echo '{"data":{"status":"ok"}}'
EOF
cat > "$STUB/runuser" <<EOF
#!/bin/sh
echo "runuser \$*" >> "$CALLS"
echo FAKE-PG-DUMP
EOF
cat > "$STUB/flock" <<'EOF'
#!/bin/sh
exit 0
EOF
chmod +x "$STUB"/*
export PATH="$STUB:$PATH"

# make_bundle <name> <label>: a bundle as bin/600-deploy-prd uploads it.
# A label containing "broken" makes an api-srv the health stub fails.
make_bundle() {
    local name="$1" label="$2" dir="$ROOT/incoming/$1"
    mkdir -p "$dir" "$WORK/web-$name"
    printf '#!/bin/sh\necho api %s\n' "$label" > "$dir/api-srv"
    cat > "$dir/stillthere-migrate" <<EOF
#!/bin/sh
echo "migrate-$label \$*" >> "$CALLS"
case "\$3" in
    verify) [ "\$(cat "$STATE/verify")" = ok ] || { echo refused; exit 1; } ;;
    pending) cat "$STATE/pending" ;;
    up) echo 0 > "$STATE/pending" ;;
esac
EOF
    echo "web $label" > "$WORK/web-$name/index.html"
    tar -czf "$dir/web-pub.tar.gz" -C "$WORK/web-$name" .
    printf 'commit=%s\nbranch=main\n' \
        "$(printf '%s' "$label" | "${SHA256[@]}" | cut -c1-40)" \
        > "$dir/stillthere-api.revision"
    ( cd "$dir" && "${SHA256[@]}" api-srv stillthere-migrate web-pub.tar.gz \
        stillthere-api.revision > SHA256SUMS )
}

OUT=""
RC=0
run_script() {
    RC=0
    OUT="$("$ROOT/bin/$1" "${@:2}" 2>&1)" || RC=$?
}

check_count=0
fail_count=0
check() {
    local description="$1"
    shift
    check_count=$((check_count + 1))
    if "$@"; then
        return 0
    fi
    fail_count=$((fail_count + 1))
    echo "FAIL  $description"
    echo "      output of the last run (exit $RC):"
    echo "$OUT" | sed 's/^/        /' | head -30
}
is() { [ "$1" = "$2" ]; }
has() { case "$1" in *"$2"*) return 0 ;; esac; return 1; }
lacks() { ! has "$1" "$2"; }
file_is() { [ -f "$1" ] && [ "$(cat "$1")" = "$2" ]; }
live_api() { sh "$ROOT/libexec/api-srv"; }
prev_api() { sh "$ROOT/libexec/api-srv.prev"; }
revision_bundle() { sed -n 's/^bundle=//p' "$ROOT/libexec/$1"; }

A=20261016T000001Z-aaaaaaa1
B=20261016T000002Z-bbbbbbb2
C=20261016T000003Z-ccccccc3
D=20261016T000004Z-ddddddd4
E=20261016T000005Z-eeeeeee5
F=20261016T000006Z-fffffff6
for pair in "$A:A" "$B:B" "$C:C" "$D:D" "$E:E-broken" "$F:F"; do
    make_bundle "${pair%%:*}" "${pair#*:}"
done

# --- refusals before anything is written ----------------------------------
echo 1000 > "$STATE/uid"
run_script prd-stillthere-install "$A"
check "not root: refused" is "$RC" 1
check "not root: says so" has "$OUT" "run as root"
check "not root: libexec untouched" [ ! -e "$ROOT/libexec" ]
echo 0 > "$STATE/uid"
run_script prd-stillthere-install ../incoming
check "bad bundle name: refused" has "$OUT" "is not <YYYYmmddTHHMMSSZ>-<commit>"
run_script prd-stillthere-install 20261016T000009Z-0000000
check "absent bundle: refused" has "$OUT" "no bundle at"
check "absent bundle: libexec untouched" [ ! -e "$ROOT/libexec" ]

# --- first install, with pending migrations -------------------------------
echo 3 > "$STATE/pending"
run_script prd-stillthere-install "$A"
CALLS_A="$(cat "$CALLS")"
check "A: exit 0" is "$RC" 0
check "A: api-srv live" is "$(live_api)" "api A"
check "A: web-pub live" file_is "$ROOT/web-pub/index.html" "web A"
check "A: revision names the bundle" is "$(revision_bundle stillthere-api.revision)" "$A"
check "A: no .prev on a first install" [ ! -e "$ROOT/libexec/api-srv.prev" ]
check "A: verify ran" has "$CALLS_A" "migrate-A -config $ROOT/etc/prd-stillthere-migrate.env verify"
check "A: dump before up" has "$CALLS_A" "runuser -u postgres -- pg_dump -Fc -d prd_stillthere_db"
check "A: dump written" file_is "$ROOT/backup/pre-migrate-$A.dump" FAKE-PG-DUMP
check "A: up ran" has "$CALLS_A" "migrate-A -config $ROOT/etc/prd-stillthere-migrate.env up"
check "A: restarted" has "$CALLS_A" "systemctl restart prd-stillthere-api"
check "A: says healthy" has "$OUT" "prd-stillthere-api healthy"
check "A: stage cleaned" [ ! -e "$ROOT/stage/$A" ]
check "A: no pending mark" [ ! -e "$ROOT/libexec/install.pending" ]

# --- the same bundle again: nothing changes -------------------------------
: > "$CALLS"
run_script prd-stillthere-install "$A"
check "A again: exit 0" is "$RC" 0
check "A again: says already live" has "$OUT" "is already live: nothing changed"
check "A again: still says healthy" has "$OUT" "prd-stillthere-api healthy"
check "A again: no migrate, no restart" is "$(cat "$CALLS")" ""
check "A again: still no .prev" [ ! -e "$ROOT/libexec/api-srv.prev" ]

# --- a second release, nothing pending -------------------------------------
: > "$CALLS"
run_script prd-stillthere-install "$B"
CALLS_B="$(cat "$CALLS")"
check "B: exit 0" is "$RC" 0
check "B: api-srv live" is "$(live_api)" "api B"
check "B: A kept as api-srv.prev" is "$(prev_api)" "api A"
check "B: web-pub live" file_is "$ROOT/web-pub/index.html" "web B"
check "B: A kept as web-pub.prev" file_is "$ROOT/web-pub.prev/index.html" "web A"
check "B: A kept as revision .prev" is "$(revision_bundle stillthere-api.revision.prev)" "$A"
check "B: pending was asked" has "$CALLS_B" "migrate-B -config $ROOT/etc/prd-stillthere-migrate.env pending"
check "B: no up" lacks "$CALLS_B" "migrate-B -config $ROOT/etc/prd-stillthere-migrate.env up"
check "B: no dump" lacks "$CALLS_B" "pg_dump"
check "B: says nothing pending" has "$OUT" "migrate: nothing pending"

# --- refused bundles leave B live ------------------------------------------
echo "tampered" >> "$ROOT/incoming/$C/api-srv"
: > "$CALLS"
run_script prd-stillthere-install "$C"
check "C damaged: refused" is "$RC" 1
check "C damaged: says checksums" has "$OUT" "checksums do not match"
check "C damaged: no migrate ran" is "$(cat "$CALLS")" ""
check "C damaged: B still live" is "$(live_api)" "api B"

echo fail > "$STATE/verify"
run_script prd-stillthere-install "$D"
CALLS_D="$(cat "$CALLS")"
check "D verify refused: exit 1" is "$RC" 1
check "D verify refused: verify ran" has "$CALLS_D" "migrate-D -config"
check "D verify refused: no up" lacks "$CALLS_D" " up"
check "D verify refused: no restart" lacks "$CALLS_D" "systemctl"
check "D verify refused: B still live" is "$(live_api)" "api B"
check "D verify refused: A still .prev" is "$(prev_api)" "api A"
echo ok > "$STATE/verify"

# --- rollback swaps, and a second rollback swaps back ----------------------
: > "$CALLS"
run_script prd-stillthere-rollback
CALLS_R="$(cat "$CALLS")"
check "rollback: exit 0" is "$RC" 0
check "rollback: A live" is "$(live_api)" "api A"
check "rollback: B now .prev" is "$(prev_api)" "api B"
check "rollback: web A live" file_is "$ROOT/web-pub/index.html" "web A"
check "rollback: revision names A" is "$(revision_bundle stillthere-api.revision)" "$A"
check "rollback: the now-live migrator verified" has "$CALLS_R" "migrate-A -config $ROOT/etc/prd-stillthere-migrate.env verify"
check "rollback: restarted" has "$CALLS_R" "systemctl restart prd-stillthere-api"
run_script prd-stillthere-rollback
check "rollback again: B live" is "$(live_api)" "api B"
check "rollback again: web B live" file_is "$ROOT/web-pub/index.html" "web B"
check "rollback again: revision names B" is "$(revision_bundle stillthere-api.revision)" "$B"

# --- an unhealthy release rolls itself back --------------------------------
echo 1 > "$STATE/pending"
run_script prd-stillthere-install "$E"
check "E unhealthy: exit 1" is "$RC" 1
check "E unhealthy: says rolling back" has "$OUT" "rolling back to the .prev release"
check "E unhealthy: says the schema stays" has "$OUT" "the schema stays"
check "E unhealthy: the rollback was healthy" has "$OUT" "prd-stillthere-api healthy"
check "E unhealthy: B live again" is "$(live_api)" "api B"
check "E unhealthy: E kept as .prev" is "$(prev_api)" "api E-broken"
check "E unhealthy: web B live again" file_is "$ROOT/web-pub/index.html" "web B"
check "E unhealthy: revision names B" is "$(revision_bundle stillthere-api.revision)" "$B"
check "E unhealthy: its migration stayed applied" file_is "$STATE/pending" 0
check "E unhealthy: its dump kept" [ -f "$ROOT/backup/pre-migrate-$E.dump" ]

# --- an install stopped part-way resumes without losing .prev --------------
# As if a run of F died after swapping api-srv: B's api-srv is .prev, F's
# is live, and install.pending names F.
ln -f "$ROOT/libexec/api-srv" "$ROOT/libexec/api-srv.prev"
cp "$ROOT/incoming/$F/api-srv" "$ROOT/libexec/api-srv.tmp"
mv -f "$ROOT/libexec/api-srv.tmp" "$ROOT/libexec/api-srv"
echo "$F" > "$ROOT/libexec/install.pending"
run_script prd-stillthere-rollback
check "rollback mid-install: refused" has "$OUT" "stopped part-way"
run_script prd-stillthere-install "$A"
check "another bundle mid-install: refused" has "$OUT" "an install of $F stopped part-way"
check "another bundle mid-install: F still live" is "$(live_api)" "api F"
run_script prd-stillthere-install "$F"
check "F resumed: exit 0" is "$RC" 0
check "F resumed: says resuming" has "$OUT" "resuming it"
check "F resumed: api-srv not swapped twice" has "$OUT" "api-srv already swapped"
check "F resumed: api-srv live" is "$(live_api)" "api F"
check "F resumed: B kept as api-srv.prev" is "$(prev_api)" "api B"
check "F resumed: web F live" file_is "$ROOT/web-pub/index.html" "web F"
check "F resumed: web B kept as .prev" file_is "$ROOT/web-pub.prev/index.html" "web B"
check "F resumed: revision names F" is "$(revision_bundle stillthere-api.revision)" "$F"
check "F resumed: B kept as revision .prev" is "$(revision_bundle stillthere-api.revision.prev)" "$B"
check "F resumed: no pending mark" [ ! -e "$ROOT/libexec/install.pending" ]

echo "checked $check_count assertions on prd-stillthere-install and" \
    "prd-stillthere-rollback, $fail_count failed"
[ "$fail_count" -eq 0 ] || exit 1
rm -rf "$WORK"
