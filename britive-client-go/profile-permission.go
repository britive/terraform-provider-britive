package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetProfilePermission - Returns a specifc permission associated with profile
func (c *Client) GetProfilePermission(profileID string, profilePermission ProfilePermission) (*ProfilePermission, error) {
	filter := fmt.Sprintf("name eq %s", profilePermission.Name)
	endpoint := fmt.Sprintf("paps/%s/permissions", profileID)

	profilePermissions := make([]ProfilePermission, 0)

	err := client.NewQueryRequest().
		WithLock(profileID).
		WithFilter(filter).
		WithResult(&profilePermissions).
		Query(endpoint)

	if err != nil {
		return nil, err
	}
	if len(profilePermissions) == 0 {
		return nil, ErrNotFound
	}

	var pp *ProfilePermission
	for _, p := range profilePermissions {
		if strings.EqualFold(p.Type, profilePermission.Type) {
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
