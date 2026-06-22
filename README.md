# devbox

A persistent dev environment running as a pod on your single-node k0s cluster
(`homeserver`). Built on **Arch Linux**, with **Go**, **Rust**, **Node** (native
from the Arch repos, latest), **Neovim + your AstroNvim config (plugins
pre-installed)**, and **Claude Code**.

Neovim is **pinned** to a specific upstream release (`NVIM_VERSION` build arg,
default `0.12.3`) rather than installed from pacman — Arch is rolling and drops
old versions, so pacman would otherwise move nvim ahead of the AstroNvim config /
`lazy-lock.json` it was tested against. Bump it deliberately (and re-test) with
`docker build --build-arg NVIM_VERSION=0.13.0 ...` or by editing the Dockerfile.

Managed by a single self-contained Go binary (`devbox`) with Cobra, so you get
shell autocompletion for free.

## Design (the original question: KubeVirt vs Deployment+PVC)

We went with a **StatefulSet + PVC**, not KubeVirt — you don't need a full VM,
just a persistent containerized workspace, and KubeVirt would add KVM/CDI/operator
overhead for no benefit here.

- **StatefulSet (not Deployment)** so the PVC binds cleanly and there's no
  two-pods-mount-one-RWO-volume deadlock on restart.
- **Persistent home pattern**: the PVC mounts at `/home/dev`. The image bakes a
  skeleton home (nvim config + synced plugins + shell defaults) at `/opt/devhome`;
  `entrypoint.sh` seeds it into the empty PVC on first boot. So your nvim state,
  shell history, **and Claude login all survive pod restarts**.
- Toolchains live in the image (via pacman), not on the volume.

## Layout

```
dev-container/
├── Dockerfile          # Arch base; go/rust/neovim/node + claude
├── entrypoint.sh       # seeds persistent home on first boot
├── nvim/               # a COPY of your ~/.config/nvim, baked in
├── main.go             # the devbox CLI (Cobra)
├── cmd/
│   ├── root.go         # shared helpers + embedded manifests
│   ├── build.go        # build + import into k0s containerd
│   ├── deploy.go       # storageclass + namespace + statefulset
│   ├── destroy.go      # teardown with flags
│   ├── shell.go        # exec into the pod
│   └── manifests/      # k8s YAML, embedded into the binary
└── go.mod
```

## Build the CLI

```bash
cd ~/dev-container
go build -o devbox .
# optional: put it on your PATH
sudo install devbox /usr/local/bin/
```

## Command reference

| Command | What it does |
| --- | --- |
| `devbox build` | Build the image with Docker and import it into k0s containerd (no registry). |
| `devbox build --dir <path>` | Same, with a custom build-context dir (default `.`). |
| `devbox deploy` | Install the local-path StorageClass if missing, then apply namespace + StatefulSet and wait for readiness. |
| `devbox shell` | Open an interactive `bash` inside the running pod (`devbox-0`). |
| `devbox backup` | Stream a gzipped tar of the persistent home (`/home/dev`) out to a local file. |
| `devbox backup -o <file>` | Same, with a chosen filename (default `devbox-backup-<timestamp>.tar.gz`). |
| `devbox restore <file>` | Restore a backup tarball back into the home (prompts before overwriting; `--force` to skip). |
| `devbox destroy` | Delete the StatefulSet, **keep** the PVC (data safe). |
| `devbox destroy --purge-data` | Also delete the PVC → wipes `~/home` (nvim state, Claude login, `~/work`). |
| `devbox destroy --remove-image` | Also remove the `devbox:latest` image from k0s containerd. |
| `devbox destroy --remove-storage` | Also remove local-path-provisioner + StorageClass (shared cluster infra; never implied by `--all`). |
| `devbox destroy --all` | Shorthand for `--purge-data --remove-image` (does **not** touch storage). |
| `devbox completion bash\|zsh\|fish\|powershell` | Print the shell autocompletion script. |
| `devbox help [command]` | Help for any command. `-h`/`--help` works on every command too. |

Typical flow:

```bash
devbox build      # docker build + load into k0s (no registry needed)
devbox deploy     # installs local-path StorageClass if missing, then deploys
devbox shell      # interactive bash inside the pod
```

Verify tools inside:

```bash
go version && rustc --version && nvim --version | head -1 && claude --version
```

Your project lives in `~/work` (persisted). First `nvim` launch installs LSPs via
Mason on demand (the pod has network).

## Autocompletion

Cobra generates it; install for your shell once:

```bash
# bash
devbox completion bash | sudo tee /etc/bash_completion.d/devbox >/dev/null
# zsh
devbox completion zsh > "${fpath[1]}/_devbox"
# fish
devbox completion fish > ~/.config/fish/completions/devbox.fish
```

## Removing / tearing down

`devbox destroy` is safe by default — it removes the pod but **keeps your PVC**.

```bash
devbox destroy                  # delete StatefulSet, KEEP data
devbox destroy --purge-data     # also delete the PVC -> wipes ~/home
devbox destroy --remove-image   # also remove the image from k0s containerd
devbox destroy --remove-storage # also remove local-path-provisioner + StorageClass (shared infra)
devbox destroy --all            # --purge-data + --remove-image (NOT storage)
```

`--remove-storage` is intentionally separate and never implied by `--all`, because
the StorageClass is cluster-wide and other workloads may rely on it. With
`reclaimPolicy: Retain`, the backing data under `/opt/local-path-provisioner/` on
the node stays on disk until you remove it manually.

## Updating

- **Change a tool / the nvim config**: edit files here, `devbox build`, then
  `kubectl -n devbox rollout restart statefulset/devbox`.
  Note: an existing PVC keeps its seeded home, so config changes won't auto-apply
  to an already-seeded volume — either edit config live in the pod, or
  `devbox destroy --purge-data` to re-seed from the image on next deploy.
- Arch is rolling, so each rebuild pulls the latest Go/Rust/Node. Neovim is the
  exception — it's pinned (see top). Pin other packages in the Dockerfile too if
  you ever need full reproducibility.

## Notes

- `imagePullPolicy: Never` — image is loaded into k0s containerd, not pulled.
- Resource limits default to 4 CPU / 8Gi; tune in `cmd/manifests/statefulset.yaml`
  (rebuild the binary after editing — manifests are embedded).
