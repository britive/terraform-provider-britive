package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GetPermissionByName - Returns a specific permission by name
func (c *Client) GetPermissionByName(name string) (*Permissions, error) {
	filter := fmt.Sprintf(`name eq "%s"`, name)
	resourceURL := fmt.Sprintf(`%s/v1/policy-admin/permissions?filter=%s`, c.APIBaseURL, url.QueryEscape(filter))
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

	permissions := make([]Permissions, 0)
	err = json.Unmarshal(body, &permissions)
	if err != nil {
		return nil, err
	}

	if len(permissions) == 0 {
		return nil, ErrNotFound
	}

	return &permissions[0], nil
}

// GetTag - Returns a specific permission by id
func (c *Client) GetPermission(permissionID string) (*Permissions, error) {
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

	permissions := &Permissions{}
	err = json.Unmarshal(body, permissions)
	if err != nil {
		return nil, err
	}

	if permissions == nil {
		return nil, ErrNotFound
	}

	return permissions, nil
}

// AddPermission - Add new permission
func (c *Client) AddPermission(permissions Permissions) (*Permissions, error) {
	pb, err := json.Marshal(permissions)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/policy-admin/permissions", c.APIBaseURL), strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, permissionsLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &permissions)
	if err != nil {
		return nil, err
	}

	return &permissions, nil
}

// UpdatePermission - Update permission
func (c *Client) UpdatePermission(permissionID string, permissions Permissions) (*Permissions, error) {
	permissionsBody, err := json.Marshal(permissions)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/v1/policy-admin/permissions/%s", c.APIBaseURL, permissionID), strings.NewReader(string(permissionsBody)))
	if err != nil {
		return nil, err
	}

	_, err = c.DoWithLock(req, permissionsLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &permissions, nil
	}
	return nil, err
}

// DeletePermission - Delete permission
func (c *Client) DeletePermission(permissionID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/policy-admin/permissions/%s", c.APIBaseURL, permissionID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, permissionsLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
