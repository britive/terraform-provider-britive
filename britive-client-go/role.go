package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// GetRoleByName - Returns a specific role by name
func (c *Client) GetRoleByName(name string) (*Role, error) {
	resourceURL := fmt.Sprintf(`%s/v1/policy-admin/roles/%s?compactResponse=true`, c.APIBaseURL, name)
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

	role := &Role{}
	err = json.Unmarshal(body, role)
	if err != nil {
		return nil, err
	}

	return role, nil
}

// GetRole - Returns a specific role by id
func (c *Client) GetRole(roleID string) (*Role, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/policy-admin/roles/%s?compactResponse=true", c.APIBaseURL, roleID), nil)
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
func (c *Client) AddRole(role Role) (*Role, error) {
	pb, err := json.Marshal(role)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/policy-admin/roles", c.APIBaseURL), strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, roleLockName)
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
func (c *Client) UpdateRole(role Role, roleName string) (*Role, error) {
	var roleBody []byte
	var err error
	roleBody, err = json.Marshal(role)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/v1/policy-admin/roles/%s", c.APIBaseURL, roleName), strings.NewReader(string(roleBody)))
	if err != nil {
		return nil, err
	}

	_, err = c.DoWithLock(req, roleLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &role, nil
	}
	return nil, err
}

// DeleteRole - Delete role
func (c *Client) DeleteRole(roleID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/policy-admin/roles/%s", c.APIBaseURL, roleID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, roleLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
