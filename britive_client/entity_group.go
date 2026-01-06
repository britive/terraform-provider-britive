package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// CreateEntityGroup - Create entity group for an application
func (c *Client) CreateEntityGroup(ctx context.Context, applicationEntity ApplicationEntityGroup, applicationID string) (*ApplicationEntityGroup, error) {
	url := fmt.Sprintf("%s/apps/%s/root-environment-group/groups", c.APIBaseURL, applicationID)
	body, err := c.Post(ctx, url, applicationEntity, ApplicationLockName)
	if err != nil {
		return nil, err
	}

	ae := &ApplicationEntityGroup{}

	err = json.Unmarshal(body, ae)
	if err != nil {
		return nil, err
	}

	return ae, nil
}

// UpdateEntityGroup - Update the entity group for an application
func (c *Client) UpdateEntityGroup(ctx context.Context, applicationEntity ApplicationEntityGroup, applicationID string) (*ApplicationEntityGroup, error) {
	url := fmt.Sprintf("%s/apps/%s/root-environment-group/groups/%s", c.APIBaseURL, applicationID, applicationEntity.EntityID)

	body, err := c.Patch(ctx, url, applicationEntity, ApplicationLockName)
	if err != nil {
		return nil, err
	}

	ae := &ApplicationEntityGroup{}

	err = json.Unmarshal(body, ae)
	if err != nil {
		return nil, err
	}

	return ae, nil
}

// DeleteEntityGroup - Delete entity group from the application
func (c *Client) DeleteEntityGroup(ctx context.Context, applicationID, entityID string) error {
	url := fmt.Sprintf("%s/apps/%s/environment-groups/%s", c.APIBaseURL, applicationID, entityID)

	err := c.Delete(ctx, url, ApplicationLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}

	return err
}
