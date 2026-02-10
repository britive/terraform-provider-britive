package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// GetRoleByName - Returns a specific role by name
func (c *Client) GetRoleByName(ctx context.Context, name string) (*Role, error) {
	resourceURL := fmt.Sprintf(`%s/v1/policy-admin/roles/%s?compactResponse=true`, c.APIBaseURL, name)
	body, err := c.Get(ctx, resourceURL, RoleLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	role := &Role{}
	err = json.Unmarshal(body, role)
	if err != nil {
		return nil, err
	}

	return role, nil
}

// GetRole - Returns a specific role by id
func (c *Client) GetRole(ctx context.Context, roleID string) (*Role, error) {
	url := fmt.Sprintf("%s/v1/policy-admin/roles/%s?compactResponse=true", c.APIBaseURL, roleID)
	body, err := c.Get(ctx, url, RoleLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	role := &Role{}
	err = json.Unmarshal(body, role)
	if err != nil {
		return nil, err
	}

	if role == nil {
		return nil, ErrNotFound
	}

	return role, nil
}

// AddRole - Add new role
func (c *Client) AddRole(ctx context.Context, role Role) (*Role, error) {
	url := fmt.Sprintf("%s/v1/policy-admin/roles", c.APIBaseURL)
	body, err := c.Post(ctx, url, role, RoleLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &role)
	if err != nil {
		return nil, err
	}

	return &role, nil
}

// UpdateRole - Update role
func (c *Client) UpdateRole(ctx context.Context, role Role, roleName string) (*Role, error) {
	url := fmt.Sprintf("%s/v1/policy-admin/roles/%s", c.APIBaseURL, roleName)
	_, err := c.Patch(ctx, url, role, RoleLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &role, nil
	}
	return nil, err
}

// DeleteRole - Delete role
func (c *Client) DeleteRole(ctx context.Context, roleID string) error {
	url := fmt.Sprintf("%s/v1/policy-admin/roles/%s", c.APIBaseURL, roleID)
	err := c.Delete(ctx, url, RoleLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
