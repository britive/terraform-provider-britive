package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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
func (c *Client) GetApplication(appContainerID string) (*ApplicationResponse, error) {
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

	application := &ApplicationResponse{}
	err = json.Unmarshal(body, application)
	if err != nil {
		return nil, err
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

func (c *Client) GetAppEnvs(appId string, envType string) ([]ApplicationEnvironment, error) {
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

	return appEnvs, nil
}

func (c *Client) GetEnvDetails(appEnvs []ApplicationEnvironment, field string) ([]string, error) {
	var envList []string
	var envValue string

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

// CreateApplication - Create new application
func (c *Client) CreateApplication(application ApplicationRequest) (*ApplicationResponse, error) {
	applicationURL := fmt.Sprintf("%s/apps", c.APIBaseURL)
	pb, err := json.Marshal(application)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", applicationURL, strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, applicationLockName)
	if err != nil {
		return nil, err
	}

	applicationResponse := ApplicationResponse{}
	err = json.Unmarshal(body, &applicationResponse)
	if err != nil {
		return nil, err
	}
	return &applicationResponse, nil
}

// Patch Application property types
func (c *Client) PatchApplicationPropertyTypes(applicationID string, properties Properties) (*ApplicationResponse, error) {
	propertiesURL := fmt.Sprintf("%s/apps/%s/properties", c.APIBaseURL, applicationID)
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

	applicationResponse := ApplicationResponse{}
	err = json.Unmarshal(body, &applicationResponse)
	if err != nil {
		return nil, err
	}
	return &applicationResponse, nil
}

// Configure User Mappings
func (c *Client) ConfigureUserMappings(applicationID string, userMappings UserMappings) error {
	userMappingURL := fmt.Sprintf("%s/apps/%s/user-account-mappings", c.APIBaseURL, applicationID)
	pb, err := json.Marshal(userMappings)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", userMappingURL, strings.NewReader(string(pb)))
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, applicationLockName)
	if err != nil {
		return err
	}
	return nil
}

// DeleteApplication - Delete application
func (c *Client) DeleteApplication(applicationID string) error {
	applicationURL := fmt.Sprintf("%s/apps?appContainerId=%s", c.APIBaseURL, applicationID)
	req, err := http.NewRequest("DELETE", applicationURL, nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, applicationLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}

func (c *Client) GetEnvFullDetails(appEnvs []ApplicationEnvironment) ([]map[string]string, error) {
	envList := make([]map[string]string, len(appEnvs))

	for i, appEnv := range appEnvs {
		envValue := make(map[string]string)
		envValue["id"] = appEnv.EnvironmentID
		envValue["name"] = appEnv.EnvironmentName
		envList[i] = envValue
	}

	return envList, nil
}
