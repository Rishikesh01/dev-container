package cmd

import "github.com/spf13/cobra"

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open an interactive shell in the running devbox pod",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInteractive("kubectl", "-n", namespace, "exec", "-it", pod, "--", "bash")
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
}
