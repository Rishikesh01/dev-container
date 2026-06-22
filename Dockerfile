# Dev container for AI project work: Go + Rust + Neovim (AstroNvim) + Claude Code
# Arch base: toolchains (go/rust/node) come from pacman at the latest version.
# Neovim is the exception -> see below.
FROM archlinux:latest

# Neovim is PINNED to a specific upstream release, NOT installed from pacman.
# Arch is rolling and drops old versions, so pacman would silently move nvim to
# whatever is newest on each rebuild -- which can outrun the AstroNvim config /
# lazy-lock.json that were tested against this version. Pin it here, bump on
# purpose (and re-test the config) when you want a newer nvim.
ARG NVIM_VERSION=0.12.3

# ---- system packages (toolchains + everything AstroNvim wants; NO neovim) ----
RUN pacman -Syu --noconfirm --needed \
        base-devel git curl wget openssh \
        go rust \
        nodejs npm \
        ripgrep fd fzf unzip tar which sudo less procps-ng tmux \
        lazygit \
    && pacman -Scc --noconfirm

# ---- Neovim (pinned upstream release, decoupled from Arch rolling) ----
RUN curl -fsSL "https://github.com/neovim/neovim/releases/download/v${NVIM_VERSION}/nvim-linux-x86_64.tar.gz" -o /tmp/nvim.tgz \
    && tar -C /opt -xzf /tmp/nvim.tgz \
    && ln -s /opt/nvim-linux-x86_64/bin/nvim /usr/local/bin/nvim \
    && rm /tmp/nvim.tgz \
    && nvim --version | head -1

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
