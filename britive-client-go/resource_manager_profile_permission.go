package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) GetAvailablePermissions(profileID string) (*ResourceManagerPermissions, error) {
	url := fmt.Sprintf("%s/resource-manager/profiles/%s/available-permissions", c.APIBaseURL, profileID)
	apiMethod := "GET"

	req, err := http.NewRequest(apiMethod, url, nil)
	if err != nil {
		return nil, err
	}

	resourceManagerPermissions := &ResourceManagerPermissions{}
	resp, err := c.DoWithLock(req, resourceManagerProfilePermission)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return nil, err
	}

	if len(resp) != 0 {
		if err := json.Unmarshal(resp, resourceManagerPermissions); err != nil {
			return nil, err
		}
	}

	return resourceManagerPermissions, nil
}

func (c *Client) GetPermissionVersions(permissionID string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/resource-manager/permissions/%s", c.APIBaseURL, permissionID)
	apiMethod := "GET"

	req, err := http.NewRequest(apiMethod, url, nil)
	if err != nil {
		return nil, err
	}

	var permissionVersions []map[string]interface{}
	resp, err := c.DoWithLock(req, resourceManagerProfilePermission)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return nil, err
	}

	if len(resp) != 0 {
		if err := json.Unmarshal(resp, &permissionVersions); err != nil {
			return nil, err
		}
	}

	return permissionVersions, nil
}

func (c *Client) GetSpecifiedVersionPermission(permissionID, version string) (*ResourceTypePermission, error) {
	url := fmt.Sprintf("%s/resource-manager/permissions/%s/%s", c.APIBaseURL, permissionID, version)
	apiMethod := "GET"

	req, err := http.NewRequest(apiMethod, url, nil)
	if err != nil {
		return nil, err
	}

	resourceTypePermissions := &ResourceTypePermission{}
	resp, err := c.DoWithLock(req, resourceManagerProfilePermission)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return nil, err
	}

	if len(resp) != 0 {
		if err := json.Unmarshal(resp, resourceTypePermissions); err != nil {
			return nil, err
		}
	}

	return resourceTypePermissions, nil
}

func (c *Client) CreateUpdateResourceManagerProfilePermission(resourceManagerProfilePermission ResourceManagerProfilePermission, isUpdate bool) (*ResourceManagerProfilePermission, error) {
	var url, apiMethod string
	if isUpdate {
		url = fmt.Sprintf("%s/resource-manager/profiles/%s/permissions/%s", c.APIBaseURL, resourceManagerProfilePermission.ProfilID, resourceManagerProfilePermission.PermissionID)
		apiMethod = "PATCH"
	} else {
		url = fmt.Sprintf("%s/resource-manager/profiles/%s/permissions", c.APIBaseURL, resourceManagerProfilePermission.ProfilID)
		apiMethod = "POST"
	}

	pb, err := json.Marshal(resourceManagerProfilePermission)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(apiMethod, url, strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(req, resourceManagerProfilePolicyLock)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return nil, err
	}

	if len(resp) != 0 {
		err = json.Unmarshal(resp, &resourceManagerProfilePermission)
		if err != nil {
			return nil, err
		}
	}

	return &resourceManagerProfilePermission, nil
}

func (c *Client) GetResourceManagerProfilePermission(profileID string) (*ResourceManagerPermissions, error) {
	url := fmt.Sprintf("%s/resource-manager/profiles/%s/permissions", c.APIBaseURL, profileID)
	apiMethod := "GET"

	req, err := http.NewRequest(apiMethod, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(req, resourceManagerProfilePermission)
	if err != nil {
		return nil, err
	}

	var resourceManagerPermissions ResourceManagerPermissions
	if len(resp) != 0 {
		err = json.Unmarshal(resp, &resourceManagerPermissions)
		if err != nil {
			return nil, err
		}
	}

	return &resourceManagerPermissions, nil

}

func (c *Client) DeleteResourceManagerProfilePermission(profileID, permissionID string) error {
	url := fmt.Sprintf("%s/resource-manager/profiles/%s/permissions/%s", c.APIBaseURL, profileID, permissionID)
	apiMethod := "DELETE"

	req, err := http.NewRequest(apiMethod, url, nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, resourceManagerProfilePermission)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return err
	}

	return nil
}
