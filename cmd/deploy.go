package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the devbox onto k0s (installs a StorageClass if missing)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Ensure a default StorageClass exists.
		sc, _ := output("kubectl", "get", "storageclass")
		if !strings.Contains(sc, "(default)") {
			fmt.Println("==> No default StorageClass found -> installing local-path-provisioner")
			if err := kubectl("apply", "-f", localPathURL); err != nil {
				return err
			}
			if err := kubectl("-n", localPathNS, "rollout", "status",
				"deploy/local-path-provisioner", "--timeout=120s"); err != nil {
				return err
			}
			scYAML, err := readManifest("storageclass.yaml")
			if err != nil {
				return err
			}
			if err := runInput(scYAML, "kubectl", "apply", "-f", "-"); err != nil {
				return err
			}
		}

		// 2. Apply namespace + workload from embedded manifests.
		for _, m := range []string{"namespace.yaml", "statefulset.yaml"} {
			y, err := readManifest(m)
			if err != nil {
				return err
			}
			if err := runInput(y, "kubectl", "apply", "-f", "-"); err != nil {
				return err
			}
		}

		// 3. Wait for readiness.
		if err := kubectl("-n", namespace, "rollout", "status",
			"statefulset/devbox", "--timeout=180s"); err != nil {
			return err
		}

		fmt.Printf("\n==> devbox is up. Get a shell with:\n    devbox shell\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
