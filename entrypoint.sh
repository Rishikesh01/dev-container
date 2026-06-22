#!/usr/bin/env bash
# Seed the persistent home from the baked skeleton on first boot, then run CMD.
# The PVC is mounted at $HOME (/home/dev). On a fresh volume it is empty, so we
# copy the skeleton (nvim config, pre-synced plugins, shell defaults) into it.
# On later restarts the volume already has data and we leave it untouched.
set -euo pipefail

DEVHOME="${DEVHOME:-/opt/devhome}"
HOME="${HOME:-/home/dev}"

if [ ! -f "${HOME}/.dev-seeded" ]; then
    echo "[entrypoint] fresh home detected -> seeding from ${DEVHOME}"
    # copy skeleton contents (incl. dotfiles) without clobbering anything already there
    cp -an "${DEVHOME}/." "${HOME}/" 2>/dev/null || true
    mkdir -p "${HOME}/go" "${HOME}/.local/bin" "${HOME}/work"
    touch "${HOME}/.dev-seeded"
    echo "[entrypoint] seed complete"
fi

exec "$@"
