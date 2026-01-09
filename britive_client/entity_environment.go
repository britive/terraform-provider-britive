package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// CreateEntityEnvironment - Create entity environment for an application
func (c *Client) CreateEntityEnvironment(ctx context.Context, applicationEntity ApplicationEntityEnvironment, applicationID string) (*ApplicationEntityEnvironment, error) {
	url := fmt.Sprintf("%s/apps/%s/root-environment-group/environments", c.APIBaseURL, applicationID)
	body, err := c.Post(ctx, url, applicationEntity, ApplicationLockName)
	if err != nil {
		return nil, err
	}

	ae := &ApplicationEntityEnvironment{}

	err = json.Unmarshal(body, ae)
	if err != nil {
		return nil, err
	}

	return ae, nil
}

// DeleteEntityEnvironment - Delete entity from the application
func (c *Client) DeleteEntityEnvironment(ctx context.Context, applicationID, entityID string) error {
	url := fmt.Sprintf("%s/apps/%s/environments/%s", c.APIBaseURL, applicationID, entityID)
	err := c.Delete(ctx, url, ApplicationLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}

	return err
}

func (c *Client) GetApplicationEnvironment(ctx context.Context, appContainerID string, entityID string) (*ApplicationResponse, error) {
	url := fmt.Sprintf("%s/apps/%s/environments/%s", c.APIBaseURL, appContainerID, entityID)
	body, err := c.Get(ctx, url, ApplicationLockName)
	if err != nil {
		return nil, err
	}

	applicationEnvironmentDetails := &ApplicationResponse{}
	err = json.Unmarshal(body, applicationEnvironmentDetails)
	if err != nil {
		return nil, err
	}

	return applicationEnvironmentDetails, nil
}

// Patch Application property types
func (c *Client) PatchApplicationEnvPropertyTypes(ctx context.Context, applicationID string, entityID string, properties Properties) (*ApplicationResponse, error) {
	url := fmt.Sprintf("%s/apps/%s/environments/%s/properties", c.APIBaseURL, applicationID, entityID)
	body, err := c.Patch(ctx, url, properties, ApplicationLockName)
	if err != nil {
		return nil, err
	}

	applicationEnvResponse := ApplicationResponse{}
	err = json.Unmarshal(body, &applicationEnvResponse)
	if err != nil {
		return nil, err
	}
	return &applicationEnvResponse, nil
}
