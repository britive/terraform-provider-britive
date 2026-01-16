package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) CreateUpdateAdvancedSettings(ctx context.Context, resourceID, resourceType string, advancedSettings AdvancedSettings, isUpdate bool) error {
	profileID := ""
	resourceIDArr := strings.Split(resourceID, "/")
	resIdArrLen := len(resourceIDArr)
	if (resIdArrLen > 1) && (strings.EqualFold(resourceIDArr[resIdArrLen-2], "policy") || strings.EqualFold(resourceIDArr[resIdArrLen-2], "policies")) {
		profileID = resourceIDArr[resIdArrLen-3]
		resourceID = resourceIDArr[resIdArrLen-1]
	} else {
		resourceID = resourceIDArr[resIdArrLen-1]
	}

	apiMethod := ""
	advancedSettingURL := ""

	switch resourceType {
	case "application":
		advancedSettingURL = fmt.Sprintf("%s/apps/%s/advanced-settings", c.APIBaseURL, resourceID)
		if isUpdate {
			apiMethod = "PUT"
		} else {
			apiMethod = "POST"
		}
	case "profile":
		advancedSettingURL = fmt.Sprintf("%s/paps/%s/advanced-settings", c.APIBaseURL, resourceID)
		apiMethod = "PUT"
	case "profile_policy":
		if profileID == "" {
			return ErrNotFound
		}

		_, err := c.UpdateProfilePolicyAdvancedSettings(ctx, advancedSettings, profileID, resourceID, resourceType)
		if err != nil {
			return err
		}
		return nil
	case "resource_manager_profile":
		advancedSettingURL = fmt.Sprintf("%s/resource-manager/profile/%s/advanced-settings", c.APIBaseURL, resourceID)
		if isUpdate {
			apiMethod = "PUT"
		} else {
			apiMethod = "POST"
		}
	case "resource_manager_profile_policy":
		if profileID == "" {
			return ErrNotFound
		}

		_, err := c.UpdateProfilePolicyAdvancedSettings(ctx, advancedSettings, profileID, resourceID, resourceType)
		if err != nil {
			return err
		}
		return nil
	default:
		return ErrNotSupported
	}

	pb, err := json.Marshal(advancedSettings)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(apiMethod, advancedSettingURL, strings.NewReader(string(pb)))
	if err != nil {
		return err
	}

	body, err := c.DoWithLock(ctx, req, AdvancedSettingLockName)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	return nil
}

func (c *Client) UpdateProfilePolicyAdvancedSettings(ctx context.Context, profilePolicyAdvancedSettings AdvancedSettings, profileID, policyID, resourceType string) (*AdvancedSettings, error) {
	var err error
	advSettingUrl := ""
	resourceTypeArr := strings.Split(resourceType, "_")
	if strings.EqualFold(resourceTypeArr[0], "resource") {
		advSettingUrl = fmt.Sprintf("%s/resource-manager/profiles/%s/policies/%s", c.APIBaseURL, profileID, policyID)
	} else {
		advSettingUrl = fmt.Sprintf("%s/paps/%s/policies/%s", c.APIBaseURL, profileID, policyID)
	}

	_, err = c.Patch(ctx, advSettingUrl, profilePolicyAdvancedSettings, AdvancedSettingLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &profilePolicyAdvancedSettings, nil
	}

	return nil, err
}

func (c *Client) GetAdvancedSettings(ctx context.Context, resourceID, resourceType string) (*AdvancedSettings, error) {
	profileID := ""
	resourceIDArr := strings.Split(resourceID, "/")
	resIdArrLen := len(resourceIDArr)
	if (resIdArrLen > 1) && (strings.EqualFold(resourceIDArr[resIdArrLen-2], "policy") || strings.EqualFold(resourceIDArr[resIdArrLen-2], "policies")) {
		profileID = resourceIDArr[resIdArrLen-3]
		resourceID = resourceIDArr[resIdArrLen-1]
	} else {
		resourceID = resourceIDArr[resIdArrLen-1]
	}
	getAppSettingUrl := ""
	switch resourceType {
	case "application":
		getAppSettingUrl = fmt.Sprintf("%s/apps/%s/advanced-settings", c.APIBaseURL, resourceID)
	case "profile":
		getAppSettingUrl = fmt.Sprintf("%s/paps/%s/advanced-settings", c.APIBaseURL, resourceID)
	case "profile_policy":
		if profileID == "" {
			return nil, ErrNotFound
		}
		profilepolicy, err := c.GetProfilePolicyAdvancedSettings(ctx, profileID, resourceID, resourceType)
		if err != nil {
			return nil, err
		}
		advancedSettings := AdvancedSettings{
			Settings: profilepolicy.Settings,
		}
		return &advancedSettings, nil
	case "resource_manager_profile":
		getAppSettingUrl = fmt.Sprintf("%s/resource-manager/profile/%s/advanced-settings", c.APIBaseURL, resourceID)
	case "resource_manager_profile_policy":
		if profileID == "" {
			return nil, ErrNotFound
		}
		profilepolicy, err := c.GetProfilePolicyAdvancedSettings(ctx, profileID, resourceID, resourceType)
		if err != nil {
			return nil, err
		}
		advancedSettings := AdvancedSettings{
			Settings: profilepolicy.Settings,
		}
		return &advancedSettings, nil
	default:
		return nil, ErrNotSupported
	}

	body, err := c.Get(ctx, getAppSettingUrl, AdvancedSettingLockName)
	if err != nil {
		return nil, err
	}

	advancedSettingsResponse := AdvancedSettings{}
	err = json.Unmarshal(body, &advancedSettingsResponse)
	if err != nil {
		return nil, err
	}

	return &advancedSettingsResponse, nil
}

func (c *Client) GetProfilePolicyAdvancedSettings(ctx context.Context, profileID, policyID, resourceType string) (*AdvancedSettings, error) {

	advSettingUrl := ""
	resourceTypeArr := strings.Split(resourceType, "_")
	if strings.EqualFold(resourceTypeArr[0], "resource") {
		advSettingUrl = fmt.Sprintf("%s/resource-manager/profiles/%s/policies/%s", c.APIBaseURL, profileID, policyID)
	} else {
		advSettingUrl = fmt.Sprintf("%s/paps/%s/policies/%s?compactResponse=true", c.APIBaseURL, profileID, policyID)
	}

	body, err := c.Get(ctx, advSettingUrl, AdvancedSettingLockName)
	if err != nil {
		return nil, err
	}
	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	profilePolicy := &AdvancedSettings{}

	err = json.Unmarshal(body, profilePolicy)
	if err != nil {
		return nil, err
	}

	return profilePolicy, nil
}

// Get all Connections
func (c *Client) GetAllConnections(ctx context.Context, settingType string) ([]Connection, error) {
	var connectionsURL string
	if strings.EqualFold(settingType, "ITSM") {
		connectionsURL = fmt.Sprintf("%s/itsm-manager/connections", c.APIBaseURL)
	} else if strings.EqualFold(settingType, "IM") {
		connectionsURL = fmt.Sprintf("%s/im-manager/connections", c.APIBaseURL)
	} else {
		return nil, ErrNotSupported
	}

	body, err := c.Get(ctx, connectionsURL, AdvancedSettingLockName)
	if err != nil {
		return nil, err
	}

	connectionsResponse := []Connection{}
	err = json.Unmarshal(body, &connectionsResponse)
	if err != nil {
		return nil, err
	}
	return connectionsResponse, nil
}
