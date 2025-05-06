package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetResourceTypePermission - Returns a specific resource type permission by ID
func (c *Client) GetResourceTypePermission(permissionID string) (*ResourceTypePermission, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/permissions/%s", c.APIBaseURL, permissionID), nil)
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

	permission := &ResourceTypePermission{}
	err = json.Unmarshal(body, permission)
	if err != nil {
		return nil, err
	}

	return permission, nil
}

// CreateResourceTypePermission - Creates a new resource type permission
func (c *Client) CreateResourceTypePermission(permission ResourceTypePermission) (*ResourceTypePermission, error) {
	body, err := json.Marshal(permission)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/resource-manager/permissions", c.APIBaseURL), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	createdPermission := &ResourceTypePermission{}
	err = json.Unmarshal(respBody, createdPermission)
	if err != nil {
		return nil, err
	}

	return createdPermission, nil
}

// UpdateResourceTypePermission - Updates an existing resource type permission
func (c *Client) UpdateResourceTypePermission(permission ResourceTypePermission) (*ResourceTypePermission, error) {
	body, err := json.Marshal(permission)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/resource-manager/permissions/%s", c.APIBaseURL, permission.PermissionID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	updatedPermission := &ResourceTypePermission{}
	err = json.Unmarshal(respBody, updatedPermission)
	if err != nil {
		return nil, err
	}

	return updatedPermission, nil
}

// DeleteResourceTypePermission - Deletes a resource type permission by ID
func (c *Client) DeleteResourceTypePermission(permissionID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/resource-manager/permissions/%s", c.APIBaseURL, permissionID), nil)
	if err != nil {
		return err
	}

	_, err = c.Do(req)
	return err
}
