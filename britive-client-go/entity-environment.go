package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// CreateEntityEnvironment - Create entity environment for an application
func (c *Client) CreateEntityEnvironment(applicationEntity ApplicationEntityEnvironment, applicationID string) (*ApplicationEntityEnvironment, error) {

	applicationEntityBody, err := json.Marshal(applicationEntity)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/apps/%s/root-environment-group/environments", c.APIBaseURL, applicationID), strings.NewReader(string(applicationEntityBody)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, applicationID)

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
func (c *Client) DeleteEntityEnvironment(applicationID, entityID string) error {

	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/apps/%s/environments/%s", c.APIBaseURL, applicationID, entityID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, applicationID)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}

	return err
}

func (c *Client) GetApplicationEnvironment(appContainerID string, entityID string) (*ApplicationResponse, error) {

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/apps/%s/environments/%s", c.APIBaseURL, appContainerID, entityID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
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
func (c *Client) PatchApplicationEnvPropertyTypes(applicationID string, entityID string, properties Properties) (*ApplicationResponse, error) {

	propertiesURL := fmt.Sprintf("%s/apps/%s/environments/%s/properties", c.APIBaseURL, applicationID, entityID)
	pb, err := json.Marshal(properties)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", propertiesURL, strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, applicationLockName)
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
