package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// GetPermissionByName - Returns a specific permission by name
func (c *Client) GetPermissionByName(ctx context.Context, name string) (*Permission, error) {
	resourceURL := fmt.Sprintf(`%s/v1/policy-admin/permissions/%s`, c.APIBaseURL, name)
	body, err := c.Get(ctx, resourceURL, PermissionLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	permission := &Permission{}
	err = json.Unmarshal(body, permission)
	if err != nil {
		return nil, err
	}

	return permission, nil
}

// GetPermission - Returns a specific permission by id
func (c *Client) GetPermission(ctx context.Context, permissionID string) (*Permission, error) {
	resourceUrl := fmt.Sprintf("%s/v1/policy-admin/permissions/%s", c.APIBaseURL, permissionID)
	body, err := c.Get(ctx, resourceUrl, PermissionLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	permission := &Permission{}
	err = json.Unmarshal(body, permission)
	if err != nil {
		return nil, err
	}

	return permission, nil
}

// AddPermission - Add new permission
func (c *Client) AddPermission(ctx context.Context, permission Permission) (*Permission, error) {
	resourceUrl := fmt.Sprintf("%s/v1/policy-admin/permissions", c.APIBaseURL)
	body, err := c.Post(ctx, resourceUrl, permission, PermissionLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &permission)
	if err != nil {
		return nil, err
	}

	return &permission, nil
}

// UpdatePermission - Update permission
func (c *Client) UpdatePermission(ctx context.Context, permission Permission, permissionName string) (*Permission, error) {
	resourceUrl := fmt.Sprintf("%s/v1/policy-admin/permissions/%s", c.APIBaseURL, permissionName)
	_, err := c.Patch(ctx, resourceUrl, permission, PermissionLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &permission, nil
	}
	return nil, err
}

// DeletePermission - Delete permission
func (c *Client) DeletePermission(ctx context.Context, permissionID string) error {
	resourceUrl := fmt.Sprintf("%s/v1/policy-admin/permissions/%s", c.APIBaseURL, permissionID)
	err := c.Delete(ctx, resourceUrl, PermissionLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
