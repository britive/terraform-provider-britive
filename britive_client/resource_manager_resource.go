package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Get Server Access Resource By Name - Returns a specific server access resource by name
func (c *Client) GetServerAccessResourceByName(ctx context.Context, name string) (*ServerAccessResource, error) {
	resourceURL := fmt.Sprintf(`%s/resource-manager/resources/%s`, c.APIBaseURL, name)
	body, err := c.Get(ctx, resourceURL, ResourceManagerResourceLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	serverAccessResource := &ServerAccessResource{}
	err = json.Unmarshal(body, serverAccessResource)
	if err != nil {
		return nil, err
	}

	return serverAccessResource, nil
}

// Get Server Access Resource - Returns a specific server access resource by id
func (c *Client) GetServerAccessResource(ctx context.Context, serverAccessResourceID string) (*ServerAccessResource, error) {
	url := fmt.Sprintf("%s/resource-manager/resources/%s", c.APIBaseURL, serverAccessResourceID)
	body, err := c.Get(ctx, url, ResourceManagerResourceLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	serverAccessResource := &ServerAccessResource{}
	err = json.Unmarshal(body, serverAccessResource)
	if err != nil {
		return nil, err
	}

	return serverAccessResource, nil
}

// AddServerAccessResource - Add new server access resource
func (c *Client) AddServerAccessResource(ctx context.Context, serverAccessResource ServerAccessResource) (*ServerAccessResource, error) {
	url := fmt.Sprintf("%s/resource-manager/resources", c.APIBaseURL)
	body, err := c.Post(ctx, url, serverAccessResource, ResourceManagerResourceLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &serverAccessResource)
	if err != nil {
		return nil, err
	}

	return &serverAccessResource, nil
}

// UpdateServerAccessResource - Update Server Access Resource
func (c *Client) UpdateServerAccessResource(ctx context.Context, serverAccessResource ServerAccessResource, serverAccessResourceID string) (*ServerAccessResource, error) {
	url := fmt.Sprintf("%s/resource-manager/resources/%s", c.APIBaseURL, serverAccessResourceID)
	_, err := c.Put(ctx, url, serverAccessResource, ResourceManagerResourceLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &serverAccessResource, nil
	}
	return nil, err
}

// DeleteServerAccessResource - Delete server access resource
func (c *Client) DeleteServerAccessResource(ctx context.Context, serverAccessResourceID string) error {
	url := fmt.Sprintf("%s/resource-manager/resources/%s", c.APIBaseURL, serverAccessResourceID)
	err := c.Delete(ctx, url, ResourceManagerResourceLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
