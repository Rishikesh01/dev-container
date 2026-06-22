package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var backupOut string

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Back up the persistent home (PVC) to a local tarball",
	Long: `Streams a gzipped tar of /home/dev out of the running pod into a local
file. Works while the pod is running; captures everything on the PVC (nvim state,
Claude login, ~/work, shell history).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := backupOut
		if out == "" {
			out = "devbox-backup-" + time.Now().Format("20060102-150405") + ".tar.gz"
		}

		f, err := os.Create(out)
		if err != nil {
			return err
		}
		defer f.Close()

		// kubectl -n devbox exec devbox-0 -- tar czf - -C /home/dev .
		fmt.Printf("==> backing up %s:/home/dev -> %s\n", pod, out)
		c := exec.Command("kubectl", "-n", namespace, "exec", pod, "--",
			"tar", "czf", "-", "-C", "/home/dev", ".")
		c.Stdout = f
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			os.Remove(out) // don't leave a half-written/empty archive around
			return err
		}

		if fi, err := os.Stat(out); err == nil {
			fmt.Printf("==> done: %s (%.1f MiB)\n", out, float64(fi.Size())/(1024*1024))
		}
		return nil
	},
}

var restoreForce bool

var restoreCmd = &cobra.Command{
	Use:   "restore <backup.tar.gz>",
	Short: "Restore a backup tarball into the persistent home (PVC)",
	Long: `Streams a backup tarball back into /home/dev in the running pod.
This OVERWRITES files in the home with those from the archive.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		defer f.Close()

		if !restoreForce {
			fmt.Printf("This overwrites %s:/home/dev with the contents of %s.\nContinue? [y/N] ", pod, src)
			r := bufio.NewReader(os.Stdin)
			ans, _ := r.ReadString('\n')
			if a := strings.ToLower(strings.TrimSpace(ans)); a != "y" && a != "yes" {
				fmt.Println("aborted.")
				return nil
			}
		}

		// kubectl -n devbox exec -i devbox-0 -- tar xzf - -C /home/dev
		fmt.Printf("==> restoring %s -> %s:/home/dev\n", src, pod)
		c := exec.Command("kubectl", "-n", namespace, "exec", "-i", pod, "--",
			"tar", "xzf", "-", "-C", "/home/dev")
		c.Stdin = f
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return err
		}
		fmt.Println("==> done.")
		return nil
	},
}

func init() {
	backupCmd.Flags().StringVarP(&backupOut, "output", "o", "", "output file (default devbox-backup-<timestamp>.tar.gz)")
	restoreCmd.Flags().BoolVar(&restoreForce, "force", false, "skip the overwrite confirmation prompt")
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
}
