package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// GetPermissionByName - Returns a specific permission by name
func (c *Client) GetPermissionByName(name string) (*Permission, error) {
	resourceURL := fmt.Sprintf(`%s/v1/policy-admin/permissions/%s`, c.APIBaseURL, name)
	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if string(body) == emptyString {
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
func (c *Client) GetPermission(permissionID string) (*Permission, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/policy-admin/permissions/%s", c.APIBaseURL, permissionID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if string(body) == emptyString {
		return nil, ErrNotFound
	}

	permission := &Permission{}
	err = json.Unmarshal(body, permission)
	if err != nil {
		return nil, err
	}

	if permission == nil {
		return nil, ErrNotFound
	}

	return permission, nil
}

// AddPermission - Add new permission
func (c *Client) AddPermission(permission Permission) (*Permission, error) {
	pb, err := json.Marshal(permission)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/policy-admin/permissions", c.APIBaseURL), strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, permissionLockName)
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
func (c *Client) UpdatePermission(permission Permission, permissionName string) (*Permission, error) {
	permissionBody, err := json.Marshal(permission)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/v1/policy-admin/permissions/%s", c.APIBaseURL, permissionName), strings.NewReader(string(permissionBody)))
	if err != nil {
		return nil, err
	}

	_, err = c.DoWithLock(req, permissionLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &permission, nil
	}
	return nil, err
}

// DeletePermission - Delete permission
func (c *Client) DeletePermission(permissionID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/policy-admin/permissions/%s", c.APIBaseURL, permissionID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, permissionLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
