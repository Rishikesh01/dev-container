package cmd

import (
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// Constants shared across commands.
const (
	image          = "devbox:latest"
	namespace      = "devbox"
	pod            = "devbox-0"
	pvc            = "home-devbox-0"
	ctrNamespace   = "k8s.io"
	localPathURL   = "https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.30/deploy/local-path-storage.yaml"
	localPathNS    = "local-path-storage"
	localPathLabel = "local-path-provisioner"
)

// Kubernetes manifests are embedded so the binary is fully self-contained.
//
//go:embed manifests/*.yaml
var manifests embed.FS

var rootCmd = &cobra.Command{
	Use:   "devbox",
	Short: "Manage the AI dev container (Go+Rust+Neovim+Claude) on k0s",
	Long: `devbox builds, deploys, and tears down a persistent dev container
running as a StatefulSet on your k0s cluster.

Shell completion (autocomplete) is built in:
  devbox completion bash --help   # or zsh / fish / powershell`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// --- exec helpers ---------------------------------------------------------

// run executes a command, streaming its stdout/stderr to the user.
func run(name string, args ...string) error {
	fmt.Printf("==> %s %s\n", name, strings.Join(args, " "))
	c := exec.Command(name, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// runInteractive wires up stdin too (for `exec -it ... bash`).
func runInteractive(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// runInput feeds stdin from a string (for `kubectl apply -f -`).
func runInput(stdin, name string, args ...string) error {
	fmt.Printf("==> %s %s  (stdin)\n", name, strings.Join(args, " "))
	c := exec.Command(name, args...)
	c.Stdin = strings.NewReader(stdin)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// output runs a command and returns its combined stdout (trimmed).
func output(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return strings.TrimSpace(string(out)), err
}

// pipe runs `producer | consumer`, streaming the consumer's output.
func pipe(producer *exec.Cmd, consumer *exec.Cmd) error {
	fmt.Printf("==> %s | %s\n", strings.Join(producer.Args, " "), strings.Join(consumer.Args, " "))
	r, w := io.Pipe()
	producer.Stdout = w
	producer.Stderr = os.Stderr
	consumer.Stdin = r
	consumer.Stdout = os.Stdout
	consumer.Stderr = os.Stderr
	if err := consumer.Start(); err != nil {
		return err
	}
	if err := producer.Start(); err != nil {
		return err
	}
	perr := producer.Wait()
	w.Close()
	cerr := consumer.Wait()
	if perr != nil {
		return perr
	}
	return cerr
}

// kubectl is a thin wrapper to keep call sites short.
func kubectl(args ...string) error { return run("kubectl", args...) }

// readManifest returns an embedded manifest file's contents.
func readManifest(name string) (string, error) {
	b, err := manifests.ReadFile("manifests/" + name)
	return string(b), err
}
