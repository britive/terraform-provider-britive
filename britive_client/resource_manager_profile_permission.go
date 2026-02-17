package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

func (c *Client) GetAvailablePermissions(ctx context.Context, profileID string) (*ResourceManagerPermissions, error) {
	url := fmt.Sprintf("%s/resource-manager/profiles/%s/available-permissions", c.APIBaseURL, profileID)
	resourceManagerPermissions := &ResourceManagerPermissions{}
	body, err := c.Get(ctx, url, ResourceManagerProfileLockName)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return nil, err
	}

	if len(body) != 0 {
		if err := json.Unmarshal(body, resourceManagerPermissions); err != nil {
			return nil, err
		}
	}

	return resourceManagerPermissions, nil
}

func (c *Client) GetPermissionVersions(ctx context.Context, permissionID string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/resource-manager/permissions/%s", c.APIBaseURL, permissionID)
	var permissionVersions []map[string]interface{}
	resp, err := c.Get(ctx, url, ResourceManagerProfileLockName)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return nil, err
	}

	if len(resp) != 0 {
		if err := json.Unmarshal(resp, &permissionVersions); err != nil {
			return nil, err
		}
	}

	return permissionVersions, nil
}
