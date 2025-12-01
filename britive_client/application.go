package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

func (c *Client) GetApplicationType(ctx context.Context, appContainerID string) (*ApplicationType, error) {
	url := fmt.Sprintf("%s/apps/%s", c.APIBaseURL, appContainerID)
	resp, err := c.Get(ctx, url, ApplicationLockName)
	if err != nil {
		return nil, err
	}

	if string(resp) == "" {
		return nil, ErrNotFound
	}

	applicationType := &ApplicationType{}
	err = json.Unmarshal(resp, applicationType)
	if err != nil {
		return nil, err
	}

	return applicationType, nil
}

func (c *Client) GetApplicationByName(ctx context.Context, appName string) (*Application, error) {
	filter := fmt.Sprintf(`name eq "%s"`, appName)
	resourceURL := fmt.Sprintf(`%s/apps?filter=%s`, c.APIBaseURL, url.QueryEscape(filter))
	resp, err := c.Get(ctx, resourceURL, ApplicationLockName)
	if err != nil {
		return nil, err
	}

	if string(resp) == EmptyString {
		return nil, ErrNotFound
	}

	applications := make([]Application, 0)
	err = json.Unmarshal(resp, &applications)
	if err != nil {
		return nil, err
	}

	if len(applications) == 0 {
		return nil, ErrNotFound
	}

	return &applications[0], nil
}

// GetSystemApps fetches the list of system apps and their propertyTypes
func (c *Client) GetSystemApps(ctx context.Context) ([]SystemApp, error) {
	url := fmt.Sprintf("%s/system/apps", c.APIBaseURL)
	body, err := c.Get(ctx, url, ApplicationLockName)
	if err != nil {
		return nil, err
	}
	var apps []SystemApp
	err = json.Unmarshal(body, &apps)
	if err != nil {
		return nil, err
	}
	return apps, nil
}

// CreateApplication - Create new application
func (c *Client) CreateApplication(ctx context.Context, application ApplicationRequest) (*ApplicationResponse, error) {
	url := fmt.Sprintf("%s/apps", c.APIBaseURL)
	body, err := c.Post(ctx, url, application, ApplicationLockName)
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
func (c *Client) PatchApplicationPropertyTypes(ctx context.Context, applicationID string, properties Properties) (*ApplicationResponse, error) {
	url := fmt.Sprintf("%s/apps/%s/properties", c.APIBaseURL, applicationID)
	body, err := c.Patch(ctx, url, properties, ApplicationLockName)
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
func (c *Client) ConfigureUserMappings(ctx context.Context, applicationID string, userMappings UserMappings) error {
	url := fmt.Sprintf("%s/apps/%s/user-account-mappings", c.APIBaseURL, applicationID)
	_, err := c.Post(ctx, url, userMappings, ApplicationLockName)
	if err != nil {
		return err
	}
	return nil
}

// Create root environment group
func (c *Client) CreateRootEnvironmentGroup(ctx context.Context, applicationID string, catalogAppId int64) error {
	appEnvGroups, err := c.GetAppEnvs(ctx, applicationID, "environmentGroups")
	if err != nil {
		return err
	}

	if len(appEnvGroups) == 0 {
		var rootAppEntity ApplicationEntityGroup

		rootAppEntity.Name = "root"
		rootAppEntity.ParentID = ""

		url := fmt.Sprintf("%s/apps/%s/root-environment-group/groups", c.APIBaseURL, applicationID)
		body, err := c.Post(ctx, url, rootAppEntity, ApplicationLockName)
		if err != nil {
			return err
		}
		ae := &ApplicationEntityGroup{}

		err = json.Unmarshal(body, ae)
		if err != nil {
			return err
		}

	}
	return nil
}

func (c *Client) GetAppEnvs(ctx context.Context, appId string, envType string) ([]ApplicationEnvironment, error) {
	url := fmt.Sprintf("%s/apps/%s/root-environment-group?view=summary&type=%s", c.APIBaseURL, appId, envType)
	body, err := c.Get(ctx, url, ApplicationLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
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

func (c *Client) GetRootEnvID(ctx context.Context, applicationID string) (string, error) {
	appEnvGroups, err := c.GetAppEnvs(ctx, applicationID, "environmentGroups")
	if err != nil {
		return "", err
	}
	envGrpIdNameList, err := c.GetEnvFullDetails(appEnvGroups)
	if err != nil {
		return "", err
	}
	for _, envGrp := range envGrpIdNameList {
		if envGrp["name"] == "root" {
			return envGrp["id"], err
		}
	}
	return "", errors.New("No root Environment Group ia available for application" + applicationID)
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

// GetApplication - Returns application by id
func (c *Client) GetApplication(ctx context.Context, appContainerID string) (*ApplicationResponse, error) {
	url := fmt.Sprintf("%s/apps/%s", c.APIBaseURL, appContainerID)
	body, err := c.Get(ctx, url, ApplicationLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	application := &ApplicationResponse{}
	err = json.Unmarshal(body, application)
	if err != nil {
		return nil, err
	}

	return application, nil
}

// DeleteApplication - Delete application
func (c *Client) DeleteApplication(ctx context.Context, applicationID string) error {
	url := fmt.Sprintf("%s/apps?appContainerId=%s", c.APIBaseURL, applicationID)
	err := c.Delete(ctx, url, ApplicationLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
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
