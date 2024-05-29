package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// GetApplications - Returns all applications
func (c *Client) GetApplications() (*[]Application, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/apps", c.APIBaseURL), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	applications := make([]Application, 0)
	err = json.Unmarshal(body, &applications)
	if err != nil {
		return nil, err
	}

	return &applications, nil
}

// GetApplication - Returns application by id
func (c *Client) GetApplication(appContainerID string) (*Application, error) {
	resourceURL := fmt.Sprintf("%s/apps/%s", c.APIBaseURL, appContainerID)
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

	application := &Application{}
	err = json.Unmarshal(body, application)
	if err != nil {
		return nil, err
	}

	if application == nil {
		return nil, ErrNotFound
	}

	return application, nil
}

// GetApplicationByName - Returns application by name
func (c *Client) GetApplicationByName(name string) (*Application, error) {
	filter := fmt.Sprintf(`name eq "%s"`, name)
	resourceURL := fmt.Sprintf(`%s/apps?filter=%s`, c.APIBaseURL, url.QueryEscape(filter))
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

	applications := make([]Application, 0)
	err = json.Unmarshal(body, &applications)
	if err != nil {
		return nil, err
	}

	if len(applications) == 0 {
		return nil, ErrNotFound
	}

	return &applications[0], nil
}
func (c *Client) GetEnvDetails(appId string, envType string, field string) ([]string, error) {
	var envList []string
	var envValue string

	resourceURL := fmt.Sprintf("%s/apps/%s/root-environment-group?view=summary&type=%s", c.APIBaseURL, appId, envType)
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

	appEnvs := make([]ApplicationEnvironment, 0)

	err = json.Unmarshal(body, &appEnvs)
	if err != nil {
		return nil, err
	}

	if appEnvs == nil {
		return nil, ErrNotFound
	}

	for _, appEnv := range appEnvs {
		switch field {
		case "id":
			envValue = appEnv.EnvironmentID
		case "name":
			envValue = appEnv.EnvironmentName
		default:
			return nil, ErrNotFound
		}
		envList = append(envList, envValue)
	}

	return envList, nil
}
