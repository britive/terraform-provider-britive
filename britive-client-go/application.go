package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
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

// GetApplication - Returns application
func (c *Client) GetApplication(appContainerID string) (*Application, error) {
	resourceURL := fmt.Sprintf("%s/apps/%s", c.HostURL, appContainerID)
	return c.getApplication(resourceURL)
}

// GetApplicationByName - Returns application
func (c *Client) GetApplicationByName(name string) (*Application, error) {
	resourceURL := fmt.Sprintf("%s/apps?metadata=false&name=%s", c.HostURL, name)
	return c.getApplication(resourceURL)
}

func (c *Client) getApplication(resourceURL string) (*Application, error) {
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
