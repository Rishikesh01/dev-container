package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the devbox onto k0s (installs the storage provisioner if missing)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Ensure local-path-provisioner is running (don't rely on a default
		//    StorageClass — we use our own dedicated, explicitly-named class).
		if err := exec.Command("kubectl", "-n", localPathNS, "get",
			"deploy", "local-path-provisioner").Run(); err != nil {
			fmt.Println("==> local-path-provisioner not found -> installing it")
			if err := kubectl("apply", "-f", localPathURL); err != nil {
				return err
			}
			if err := kubectl("-n", localPathNS, "rollout", "status",
				"deploy/local-path-provisioner", "--timeout=120s"); err != nil {
				return err
			}
		}

		// 2. Apply our dedicated StorageClass (idempotent; own name avoids the
		//    upstream class's immutable-reclaimPolicy collision).
		scYAML, err := readManifest("storageclass.yaml")
		if err != nil {
			return err
		}
		if err := runInput(scYAML, "kubectl", "apply", "-f", "-"); err != nil {
			return err
		}

		// 3. Apply namespace + workload from embedded manifests.
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
