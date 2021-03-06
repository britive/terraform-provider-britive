package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// GetApplicationRootEnvironmentGroup - Returns root environment group
func (c *Client) GetApplicationRootEnvironmentGroup(appContainerID string) (*ApplicationRootEnvironmentGroup, error) {
	//TODO: Warning Recursion - Get by Filter
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/apps/%s/root-environment-group", c.APIBaseURL, appContainerID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	applicationRootEnvironmentGroup := &ApplicationRootEnvironmentGroup{}
	err = json.Unmarshal(body, applicationRootEnvironmentGroup)
	if err != nil {
		return nil, err
	}

	return applicationRootEnvironmentGroup, nil
}
