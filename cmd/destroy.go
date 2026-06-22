package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	purgeData     bool
	removeImage   bool
	removeStorage bool
	destroyAll    bool
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Tear down the devbox (keeps your data by default)",
	Long: `Removes the running pod. By default the PVC is KEPT so you can redeploy
and continue where you left off.

  devbox destroy                    delete StatefulSet, keep data
  devbox destroy --purge-data       also delete the PVC (wipes ~/home)
  devbox destroy --remove-image     also remove the image from k0s containerd
  devbox destroy --remove-storage   also remove local-path-provisioner + StorageClass
  devbox destroy --all              --purge-data + --remove-image
                                    (does NOT remove shared storage; use --remove-storage)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if destroyAll {
			purgeData = true
			removeImage = true
		}

		fmt.Println("==> Deleting StatefulSet")
		_ = kubectl("-n", namespace, "delete", "statefulset", "devbox", "--ignore-not-found")

		if purgeData {
			fmt.Println("==> Purging PVC (erases the persistent home)")
			_ = kubectl("-n", namespace, "delete", "pvc", pvc, "--ignore-not-found")
			// Only drop the namespace when we're also wiping data; otherwise
			// deleting the namespace would take the kept PVC with it.
			_ = kubectl("delete", "namespace", namespace, "--ignore-not-found")
		} else {
			fmt.Println("==> Keeping PVC (use --purge-data to wipe). Current PVCs:")
			_ = kubectl("-n", namespace, "get", "pvc")
		}

		if removeImage {
			fmt.Println("==> Removing devbox image from k0s containerd")
			_ = run("sudo", "k0s", "ctr", "--namespace", ctrNamespace, "images", "rm",
				"docker.io/library/"+image)
			_ = run("sudo", "k0s", "ctr", "--namespace", ctrNamespace, "images", "rm", image)
		}

		if removeStorage {
			fmt.Println("==> Removing local-path-provisioner + StorageClass (shared cluster infra!)")
			scYAML, err := readManifest("storageclass.yaml")
			if err != nil {
				return err
			}
			_ = runInput(scYAML, "kubectl", "delete", "-f", "-", "--ignore-not-found")
			_ = kubectl("delete", "-f", localPathURL, "--ignore-not-found")
			fmt.Println("    Note: backing data under /opt/local-path-provisioner/ on the node")
			fmt.Println("    is left on disk (reclaimPolicy: Retain). Remove it manually if desired.")
		} else {
			fmt.Println("==> Leaving StorageClass/local-path-provisioner installed (use --remove-storage to drop)")
		}

		fmt.Println("==> Done.")
		return nil
	},
}

func init() {
	destroyCmd.Flags().BoolVar(&purgeData, "purge-data", false, "also delete the PVC (wipes the persistent home)")
	destroyCmd.Flags().BoolVar(&removeImage, "remove-image", false, "also remove the image from k0s containerd")
	destroyCmd.Flags().BoolVar(&removeStorage, "remove-storage", false, "also remove local-path-provisioner + StorageClass (shared infra)")
	destroyCmd.Flags().BoolVar(&destroyAll, "all", false, "shorthand for --purge-data --remove-image")
	rootCmd.AddCommand(destroyCmd)
}
