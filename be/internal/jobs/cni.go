package jobs

import (
	"context"
	"fmt"

	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
)

// cniInstallers maps a validated CNI value (see handlers.validateCNI) to its
// install routine. Adding a new CNI later means adding one function below
// and one entry here — runNodeJob's control flow never changes.
var cniInstallers = map[string]func(ctx context.Context, m *Manager, incusName string) error{
	string(models.CNITypeCilium): installCilium,
}

// installCNI installs cni onto the master node's cluster. cni must already
// be validated by the caller — an unrecognized value here is a programming
// error, not user input.
func installCNI(ctx context.Context, m *Manager, incusName, cni string) error {
	installer, ok := cniInstallers[cni]
	if !ok {
		return fmt.Errorf("no installer registered for cni %q", cni)
	}
	return installer(ctx, m, incusName)
}

// ciliumCLIInstallScript follows docs.cilium.io's Linux Cilium CLI install
// steps, minus sudo (already root inside the VM).
const ciliumCLIInstallScript = `set -euo pipefail
CILIUM_CLI_VERSION=$(curl -sL https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
CLI_ARCH=amd64
if [ "$(uname -m)" = "aarch64" ]; then CLI_ARCH=arm64; fi
cd /tmp
curl -L --fail --remote-name-all "https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-${CLI_ARCH}.tar.gz"{,.sha256sum}
sha256sum --check "cilium-linux-${CLI_ARCH}.tar.gz.sha256sum"
tar xzvfC "cilium-linux-${CLI_ARCH}.tar.gz" /usr/local/bin
rm -f "cilium-linux-${CLI_ARCH}.tar.gz"{,.sha256sum}
`

// installCilium installs the Cilium CLI on incusName, runs `cilium install`
// against the root kubeconfig runNodeJob already copied there, and blocks
// on `cilium status --wait` until Cilium reports healthy or times out.
func installCilium(ctx context.Context, m *Manager, incusName string) error {
	if _, err := m.incus.Run(ctx, incusName, []string{"bash", "-c", ciliumCLIInstallScript}); err != nil {
		return fmt.Errorf("installing cilium CLI: %w", err)
	}

	installCmd := []string{"cilium", "install", "--kubeconfig=" + rootKubeconfigPath}
	if _, err := m.incus.Run(ctx, incusName, installCmd); err != nil {
		return fmt.Errorf("cilium install: %w", err)
	}

	statusCmd := []string{"cilium", "status", "--wait", "--kubeconfig=" + rootKubeconfigPath, "--wait-duration=8m"}
	if _, err := m.incus.Run(ctx, incusName, statusCmd); err != nil {
		return fmt.Errorf("cilium status --wait: %w", err)
	}

	return nil
}
