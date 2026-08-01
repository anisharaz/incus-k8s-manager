package incus

import (
	"fmt"

	"github.com/lxc/incus/v7/shared/api"
)

// ListNetworks returns every network known to the Incus daemon, including
// ones this application didn't create (e.g. the appliance's own bridge).
func (c *Client) ListNetworks() ([]api.Network, error) {
	networks, err := c.server.GetNetworks()
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}

	return networks, nil
}

// GetNetwork returns a single network by name.
func (c *Client) GetNetwork(name string) (*api.Network, error) {
	network, _, err := c.server.GetNetwork(name)
	if err != nil {
		return nil, fmt.Errorf("get network %q: %w", name, err)
	}

	return network, nil
}

// CreateNetwork creates a new bridge network with the given config (e.g.
// "ipv4.address": "10.10.0.1/24").
func (c *Client) CreateNetwork(name string, config map[string]string) error {
	req := api.NetworksPost{
		Name: name,
		Type: "bridge",
		NetworkPut: api.NetworkPut{
			Config: config,
		},
	}

	if err := c.server.CreateNetwork(req); err != nil {
		return fmt.Errorf("create network %q: %w", name, err)
	}

	return nil
}

// DeleteNetwork deletes a network by name.
func (c *Client) DeleteNetwork(name string) error {
	if err := c.server.DeleteNetwork(name); err != nil {
		return fmt.Errorf("delete network %q: %w", name, err)
	}

	return nil
}
