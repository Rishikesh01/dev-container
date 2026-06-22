# Dev container for AI project work: Go + Rust + Neovim (AstroNvim) + Claude Code
# Arch base: everything installs natively from pacman at the latest version,
# so there are no manual tarball downloads / symlinks to maintain.
FROM archlinux:latest

# ---- system packages (toolchains + everything AstroNvim wants) ----
# go, rust, neovim, nodejs/npm all come straight from the repos at latest.
RUN pacman -Syu --noconfirm --needed \
        base-devel git curl wget openssh \
        go rust \
        neovim \
        nodejs npm \
        ripgrep fd fzf unzip tar which sudo less procps-ng tmux \
        lazygit \
    && pacman -Scc --noconfirm

ENV GOPATH=/home/dev/go \
    GOTOOLCHAIN=local

# ---- Claude Code CLI (npm, global) ----
RUN npm install -g @anthropic-ai/claude-code && npm cache clean --force

# ---- non-root dev user ----
RUN useradd -m -s /bin/bash -u 1000 dev \
    && echo 'dev ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/dev

# ---- skeleton home: nvim config (copied, not linked) + pre-synced plugins ----
# Lives at /opt/devhome and is seeded into the PVC on first boot (entrypoint.sh).
ENV DEVHOME=/opt/devhome
RUN mkdir -p ${DEVHOME}/.config
COPY --chown=root:root nvim/ ${DEVHOME}/.config/nvim/

# Pre-install AstroNvim plugins headlessly so first launch is instant.
RUN HOME=${DEVHOME} XDG_CONFIG_HOME=${DEVHOME}/.config XDG_DATA_HOME=${DEVHOME}/.local/share \
        XDG_STATE_HOME=${DEVHOME}/.local/state XDG_CACHE_HOME=${DEVHOME}/.cache \
        nvim --headless "+Lazy! sync" +qa 2>/dev/null || true

# helpful shell defaults baked into the skeleton
RUN printf '%s\n' \
        'export PATH=$HOME/go/bin:$HOME/.local/bin:$PATH' \
        'export GOPATH=$HOME/go' \
        'export EDITOR=nvim' \
        'alias vi=nvim' 'alias vim=nvim' 'alias ll="ls -alh"' \
        >> ${DEVHOME}/.bashrc \
    && chown -R dev:dev ${DEVHOME}

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

USER dev
ENV HOME=/home/dev
WORKDIR /home/dev
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["sleep", "infinity"]
