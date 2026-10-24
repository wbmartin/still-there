#!/bin/bash
# Install or upgrade the agent on the Pi. Run as root from the directory
# bin/630-deploy-pi-agent copied it to:
#
#   sudo ./install.sh
#
# Installs stillthere-agent and its systemd unit, creates the stillthere
# user (in group bluetooth, which may talk to BlueZ), and installs
# agent.json when one was copied alongside. An existing config is replaced
# only by a copied one: the server's pushed settings live in it.

set -euo pipefail

if [ "$(id -u)" != 0 ]; then
    echo "[ERROR] run as root" >&2
    exit 1
fi
cd "$(dirname "${BASH_SOURCE[0]}")"

if ! id stillthere >/dev/null 2>&1; then
    useradd --system --no-create-home --shell /usr/sbin/nologin \
        --groups bluetooth stillthere
fi

install -d -m 0700 -o stillthere -g stillthere /etc/stillthere
if [ -f agent.json ]; then
    install -m 0600 -o stillthere -g stillthere agent.json /etc/stillthere/agent.json
elif [ ! -f /etc/stillthere/agent.json ]; then
    echo "[ERROR] no /etc/stillthere/agent.json: pass a config to bin/630-deploy-pi-agent" >&2
    exit 1
fi

install -m 0755 stillthere-agent /usr/local/bin/stillthere-agent
install -m 0644 stillthere-agent.service /etc/systemd/system/stillthere-agent.service
systemctl daemon-reload
systemctl enable stillthere-agent.service
systemctl restart stillthere-agent.service
sleep 3
systemctl is-active stillthere-agent.service
journalctl -u stillthere-agent.service -n 5 --no-pager
