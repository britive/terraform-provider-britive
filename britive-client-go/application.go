package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// GetApplications - Returns all applications
func (c *Client) GetApplications() (*[]Application, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/apps", c.HostURL), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
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
	resourceURL := fmt.Sprintf("%s/apps/%s", c.HostURL, appContainerID)
	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	application := Application{}
	err = json.Unmarshal(body, &application)
	if err != nil {
		return nil, err
	}

	return &application, nil
}

// GetApplicationByName - Returns application by name
func (c *Client) GetApplicationByName(name string) (*Application, error) {
	filter := fmt.Sprintf(`name eq "%s"`, name)
	resourceURL := fmt.Sprintf(`%s/apps?filter=%s`, c.HostURL, url.QueryEscape(filter))
	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	if string(body) == "" {
		return nil, fmt.Errorf("No application matching with the name %s", name)
	}

	applications := make([]Application, 0)
	err = json.Unmarshal(body, &applications)
	if err != nil {
		return nil, err
	}

	if len(applications) == 0 {
		return nil, fmt.Errorf("No application matching with the name %s", name)
	}

	return &applications[0], nil
}
