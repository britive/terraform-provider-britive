package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// CreateEntityGroup - Create entity group for an application
func (c *Client) CreateEntityGroup(applicationEntity ApplicationEntity, applicationID string) (*ApplicationEntity, error) {

	appEnvGroups, err := c.GetAppEnvs(applicationID, "environmentGroups")
	if err != nil {
		return nil, err
	}

	if len(appEnvGroups) == 0 && !(applicationEntity.Name == "root" && applicationEntity.ParentID == "") {
		return nil, fmt.Errorf("As there is no environment group created yet for this application, please create one with only the exact same values as below\napplication_id = \"%s\"\nentity_name = \"root\"\nparent_id = \"\"", applicationID)
	}

	applicationEntityBody, err := json.Marshal(applicationEntity)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/apps/%s/root-environment-group/groups", c.APIBaseURL, applicationID), strings.NewReader(string(applicationEntityBody)))
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

// UpdateEntityGroup - Update the entity group for an application
func (c *Client) UpdateEntityGroup(applicationEntity ApplicationEntity, applicationID string) (*ApplicationEntity, error) {

	applicationEntityBody, err := json.Marshal(applicationEntity)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/apps/%s/root-environment-group/groups/%s", c.APIBaseURL, applicationID, applicationEntity.EntityID), strings.NewReader(string(applicationEntityBody)))
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

// DeleteEntityGroup - Delete entity group from the application
func (c *Client) DeleteEntityGroup(applicationID, entityID string) error {

	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/apps/%s/environment-groups/%s", c.APIBaseURL, applicationID, entityID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, applicationID)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}

	return err
}
