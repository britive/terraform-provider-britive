package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetProfilePermissions - Returns all permissions assigned to user profile
func (c *Client) GetProfilePermissions(profileID string) (*[]ProfilePermission, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/paps/%s/permissions?filter=assigned", c.HostURL, profileID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	profilePermissions := make([]ProfilePermission, 0)
	err = json.Unmarshal(body, &profilePermissions)
	if err != nil {
		return nil, err
	}

	return &profilePermissions, nil
}

// GetProfilePermission - Returns a specifc user from user profile
func (c *Client) GetProfilePermission(profileID string, profilePermission ProfilePermission) (*ProfilePermission, error) {
	//TODO: Warning Recursion - Get by Name
	profilePermissions, err := c.GetProfilePermissions(profileID)
	if err != nil {
		return nil, err
	}
	if profilePermissions == nil || len(*profilePermissions) == 0 {
		return nil, fmt.Errorf("No profiles permissions matching for the resource /paps/%s/permissions/%s", profileID, profilePermission.Name)
	}

	var pp *ProfilePermission
	for _, p := range *profilePermissions {
		if strings.ToLower(p.Name) == strings.ToLower(profilePermission.Name) && strings.ToLower(p.Type) == strings.ToLower(profilePermission.Type) {
			pp = &p
			break
		}
	}

	return pp, nil
}

// ExecuteProfilePermissionRequest - Add/delete permission from profile
func (c *Client) ExecuteProfilePermissionRequest(profileID string, ppr ProfilePermissionRequest) error {
	pprb, err := json.Marshal(ppr)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/paps/%s/permissions", c.HostURL, profileID), strings.NewReader(string(pprb)))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	if err != nil {
		return err
	}

	return nil
}
