package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// GetResourceTypeByName - Returns a specific resource type by name
func (c *Client) GetResourceTypeByName(name string) (*ResourceType, error) {
	resourceURL := fmt.Sprintf(`%s/resource-manager/resource-types/%s?compactResponse=true`, c.APIBaseURL, name)
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

	resourceType := &ResourceType{}
	err = json.Unmarshal(body, resourceType)
	if err != nil {
		return nil, err
	}

	return resourceType, nil
}

// GetRole - Returns a specific role by id
func (c *Client) GetResourceType(resourceTypeID string) (*ResourceType, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(`%s/resource-manager/resource-types/%s?compactResponse=true`, c.APIBaseURL, resourceTypeID), nil)

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

	resourceType := &ResourceType{}
	err = json.Unmarshal(body, resourceType)
	if err != nil {
		return nil, err
	}

	if resourceType == nil {
		return nil, ErrNotFound
	}

	return resourceType, nil
}

// CreateResourceType - Create new resource type
func (c *Client) CreateResourceType(resourceType ResourceType) (*ResourceType, error) {
	pb, err := json.Marshal(resourceType)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/resource-manager/resource-types", c.APIBaseURL), strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, resourceTypeLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &resourceType)
	if err != nil {
		return nil, err
	}
	return &resourceType, nil
}

// UpdateRole - Update role
func (c *Client) UpdateResourceType(resourceType ResourceType, resourceTypeID string) (*ResourceType, error) {
	var resourceTypeBody []byte
	var err error
	resourceTypeBody, err = json.Marshal(resourceType)
	if err != nil {
		return nil, err
	}

	// ToDo: Check for patch/put
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/resource-manager/resource-types/%s", c.APIBaseURL, resourceTypeID), strings.NewReader(string(resourceTypeBody)))
	if err != nil {
		return nil, err
	}

	_, err = c.DoWithLock(req, resourceTypeLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &resourceType, nil
	}
	return nil, err
}

// DeleteRole - Delete role
func (c *Client) DeleteResourceType(resourceTypeID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/resource-manager/resource-types/%s", c.APIBaseURL, resourceTypeID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, resourceTypeLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
