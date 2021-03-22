package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetAssignedProfilePermissions - Returns all permissions assigned to profile
func (c *Client) GetAssignedProfilePermissions(profileID string) (*[]ProfilePermission, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/paps/%s/permissions?filter=assigned", c.APIBaseURL, profileID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileID)
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

// GetProfilePermission - Returns a specifc permission associated with profile
func (c *Client) GetProfilePermission(profileID string, profilePermission ProfilePermission) (*ProfilePermission, error) {
	//TODO: Warning Recursion - Get by Name
	profilePermissions, err := c.GetAssignedProfilePermissions(profileID)
	if err != nil {
		return nil, err
	}
	if profilePermissions == nil || len(*profilePermissions) == 0 {
		return nil, ErrNotFound
	}

	var pp *ProfilePermission
	for _, p := range *profilePermissions {
		if strings.EqualFold(p.Name, profilePermission.Name) && strings.EqualFold(p.Type, profilePermission.Type) {
			pp = &p
			break
		}
	}

	if pp == nil {
		return nil, ErrNotFound
	}

	return pp, nil
}

// ExecuteProfilePermissionRequest - Add/delete permission from profile
func (c *Client) ExecuteProfilePermissionRequest(profileID string, ppr ProfilePermissionRequest) error {
	profilePermissionRequestBody, err := json.Marshal(ppr)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/paps/%s/permissions", c.APIBaseURL, profileID), strings.NewReader(string(profilePermissionRequestBody)))
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, profileID)
	if err != nil {
		return err
	}

	return nil
}
