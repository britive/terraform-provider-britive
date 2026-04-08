package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// GetApplicationRootEnvironmentGroup - Returns root environment group
func (c *Client) GetApplicationRootEnvironmentGroup(appContainerID string) (*ApplicationRootEnvironmentGroup, error) {
	cacheKey := fmt.Sprintf("root-env-group:%s", appContainerID)
	if cached, ok := c.cacheGet(cacheKey); ok {
		original := cached.(*ApplicationRootEnvironmentGroup)
		cp := *original
		cp.EnvironmentGroups = make([]Association, len(original.EnvironmentGroups))
		copy(cp.EnvironmentGroups, original.EnvironmentGroups)
		cp.Environments = make([]Association, len(original.Environments))
		copy(cp.Environments, original.Environments)
		return &cp, nil
	}

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

	c.cacheSet(cacheKey, applicationRootEnvironmentGroup)
	return applicationRootEnvironmentGroup, nil
}
