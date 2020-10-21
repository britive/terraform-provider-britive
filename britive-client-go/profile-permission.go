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
	profilePermissions, err := c.GetProfilePermissions(profileID)
	if err != nil {
		return nil, err
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

// PerformProfilePermissionRequest - Add/delete permission from profile
func (c *Client) PerformProfilePermissionRequest(profileID string, ppr ProfilePermissionRequest) (*ProfilePermission, error) {
	pprb, err := json.Marshal(ppr)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/paps/%s/permissions", c.HostURL, profileID), strings.NewReader(string(pprb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var profilePermission ProfilePermission
	err = json.Unmarshal(body, &profilePermission)
	if err != nil {
		return nil, err
	}
	return &profilePermission, nil
}
