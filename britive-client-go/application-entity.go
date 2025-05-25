package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// CreateApplicationEntity - Create entity for an application
func (c *Client) CreateApplicationEntity(applicationEntity ApplicationEntity, applicationID string) (*ApplicationEntity, error) {

	log.Printf("[INFO] Inside CreateApplicationEntity: %#v", applicationEntity)

	applicationEntityBody, err := json.Marshal(applicationEntity)
	if err != nil {
		return nil, err
	}

	var entType string

	if strings.EqualFold(applicationEntity.Type, environment) {
		entType = "environments"
	} else if strings.EqualFold(applicationEntity.Type, environmentGroup) {
		entType = "groups"
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/apps/%s/root-environment-group/%s", c.APIBaseURL, applicationID, entType), strings.NewReader(string(applicationEntityBody)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, applicationID)

	if err != nil {
		return nil, err
	}

	ae := &ApplicationEntity{}

	err = json.Unmarshal(body, ae)
	if err != nil {
		return nil, err
	}

	return ae, nil
}

// UpdateApplicationEntity - Update the entity for an application
func (c *Client) UpdateApplicationEntity(applicationEntity ApplicationEntity, applicationID string) (*ApplicationEntity, error) {

	applicationEntityBody, err := json.Marshal(applicationEntity)
	if err != nil {
		return nil, err
	}

	var entType string

	if strings.EqualFold(applicationEntity.Type, environment) {
		entType = "environments"
	} else if strings.EqualFold(applicationEntity.Type, environmentGroup) {
		entType = "groups"
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/apps/%s/root-environment-group/%s/%s", c.APIBaseURL, applicationID, entType, applicationEntity.EntityID), strings.NewReader(string(applicationEntityBody)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, applicationID)

	if err != nil {
		return nil, err
	}

	ae := &ApplicationEntity{}

	err = json.Unmarshal(body, ae)
	if err != nil {
		return nil, err
	}

	return ae, nil
}

// DeleteApplicationEntity - Delete entity from the application
func (c *Client) DeleteApplicationEntity(applicationID, entityType, entityID string) error {
	var delType string
	if strings.EqualFold(entityType, environment) {
		delType = "environments"
	} else if strings.EqualFold(entityType, environmentGroup) {
		delType = "environment-groups"
	}

	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/apps/%s/%s/%s", c.APIBaseURL, applicationID, delType, entityID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, applicationID)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}

	return err
}
