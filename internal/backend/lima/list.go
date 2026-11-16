package lima

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Instance describes a Lima VM without consulting its guest or VBX state.
type Instance struct {
	Name, Status, MountType string
	Mounts                  []Mount
}
type Mount struct {
	Location, MountPoint string
	Writable             bool
}

// List reads Lima's stream of JSON instance objects, including captured mounts.
func (c *Client) List(ctx context.Context) ([]Instance, error) {
	output, err := c.capture(ctx, "list", "--json", "--all-fields")
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(output))
	var instances []Instance
	for {
		var record struct {
			Name, Status string
			Config       struct {
				MountType string
				Mounts    []Mount
			}
		}
		if err := decoder.Decode(&record); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decode Lima instance list: %w", err)
		}
		if record.Name == "" || record.Status == "" {
			return nil, fmt.Errorf("Lima instance list contains an unnamed VM or missing status")
		}
		instances = append(instances, Instance{Name: record.Name, Status: record.Status, MountType: record.Config.MountType, Mounts: record.Config.Mounts})
	}
	return instances, nil
}
