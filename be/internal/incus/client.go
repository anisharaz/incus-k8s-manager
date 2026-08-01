// Package incus wraps the Incus client SDK to provide instance lifecycle
// management and in-VM command execution over the shared unix socket
// exposed by the incus container (see meta/incusDocker/entrypoint.sh, which
// proxies /var/lib/incus/unix.socket to /shared-socket/incus.sock).
package incus

import (
	"context"
	"fmt"
	"time"

	incusclient "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

// Client wraps a connection to an Incus daemon.
type Client struct {
	server incusclient.InstanceServer
}

// New connects to the Incus daemon over a unix socket at socketPath. If
// socketPath is empty, the SDK falls back to $INCUS_SOCKET, then
// $INCUS_DIR/unix.socket, then the standard system socket paths.
func New(socketPath string) (*Client, error) {
	server, err := incusclient.ConnectIncusUnix(socketPath, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to incus at %q: %w", socketPath, err)
	}

	return &Client{server: server}, nil
}

// Launch creates a new instance from the given image alias and starts it,
// blocking until the operation completes.
func (c *Client) Launch(ctx context.Context, name, imageAlias string, vm bool) error {
	instanceType := api.InstanceTypeContainer
	if vm {
		instanceType = api.InstanceTypeVM
	}

	req := api.InstancesPost{
		Name:  name,
		Type:  instanceType,
		Start: true,
		Source: api.InstanceSource{
			Type:  "image",
			Alias: imageAlias,
		},
	}

	op, err := c.server.CreateInstance(req)
	if err != nil {
		return fmt.Errorf("create instance %q: %w", name, err)
	}

	if err := op.WaitContext(ctx); err != nil {
		return fmt.Errorf("launch instance %q: %w", name, err)
	}

	return nil
}

// Start starts a stopped instance.
func (c *Client) Start(ctx context.Context, name string) error {
	return c.setState(ctx, name, "start", 30, false)
}

// Stop stops a running instance. If force is false, Incus asks the guest
// OS to shut down cleanly (via ACPI/agent) before the timeout elapses.
func (c *Client) Stop(ctx context.Context, name string, force bool) error {
	return c.setState(ctx, name, "stop", 30, force)
}

func (c *Client) setState(ctx context.Context, name, action string, timeoutSeconds int, force bool) error {
	op, err := c.server.UpdateInstanceState(name, api.InstanceStatePut{
		Action:  action,
		Timeout: timeoutSeconds,
		Force:   force,
	}, "")
	if err != nil {
		return fmt.Errorf("%s instance %q: %w", action, name, err)
	}

	if err := op.WaitContext(ctx); err != nil {
		return fmt.Errorf("%s instance %q: %w", action, name, err)
	}

	return nil
}

// Delete stops (forcefully, if needed) and deletes an instance.
func (c *Client) Delete(ctx context.Context, name string) error {
	instance, _, err := c.server.GetInstance(name)
	if err != nil {
		return fmt.Errorf("get instance %q: %w", name, err)
	}

	if instance.StatusCode != api.Stopped {
		if err := c.Stop(ctx, name, true); err != nil {
			return err
		}
	}

	op, err := c.server.DeleteInstance(name)
	if err != nil {
		return fmt.Errorf("delete instance %q: %w", name, err)
	}

	if err := op.WaitContext(ctx); err != nil {
		return fmt.Errorf("delete instance %q: %w", name, err)
	}

	return nil
}

// Get returns the current state of a single instance.
func (c *Client) Get(name string) (*api.Instance, error) {
	instance, _, err := c.server.GetInstance(name)
	if err != nil {
		return nil, fmt.Errorf("get instance %q: %w", name, err)
	}

	return instance, nil
}

// List returns all instances known to the daemon.
func (c *Client) List() ([]api.Instance, error) {
	instances, err := c.server.GetInstances(api.InstanceTypeAny)
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", err)
	}

	return instances, nil
}

// WaitForIPv4 polls the instance state until the Incus agent reports a
// global-scope IPv4 address (excluding loopback), or the context is done.
// VM images must include incus-agent for this to work (the k8s image does).
func (c *Client) WaitForIPv4(ctx context.Context, name string) (string, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		state, _, err := c.server.GetInstanceState(name)
		if err == nil {
			for iface, net := range state.Network {
				if iface == "lo" {
					continue
				}

				for _, addr := range net.Addresses {
					if addr.Family == "inet" && addr.Scope == "global" {
						return addr.Address, nil
					}
				}
			}
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("waiting for IPv4 address of instance %q: %w", name, ctx.Err())
		case <-ticker.C:
		}
	}
}
