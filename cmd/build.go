package cmd

import (
	"os/exec"

	"github.com/spf13/cobra"
)

var buildDir string

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the image and import it into k0s containerd (no registry)",
	Long: `Builds the dev image with Docker, then loads it directly into k0s's
containerd in the k8s.io namespace so the pod can use imagePullPolicy: Never.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := run("docker", "build", "-t", image, buildDir); err != nil {
			return err
		}
		// docker save devbox:latest | sudo k0s ctr --namespace k8s.io images import -
		producer := exec.Command("docker", "save", image)
		consumer := exec.Command("sudo", "k0s", "ctr", "--namespace", ctrNamespace, "images", "import", "-")
		return pipe(producer, consumer)
	},
}

func init() {
	buildCmd.Flags().StringVar(&buildDir, "dir", ".", "build context dir (must contain the Dockerfile and nvim/)")
	rootCmd.AddCommand(buildCmd)
}
