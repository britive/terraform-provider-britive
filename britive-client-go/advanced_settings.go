package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Create Advanced Setting
func (c *Client) CreateUpdateAdvancedSettings(resourceID, resourceType string, advancedSettings AdvancedSettings, isUpdate bool) error {
	profileID := ""
	resourceIDArr := strings.Split(resourceID, "/")
	if len(resourceIDArr) > 1 {
		profileID = resourceIDArr[1]
		resourceID = resourceIDArr[3]
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

		_, err := c.UpdateProfilePolicyAdvancedSettings(advancedSettings, profileID, resourceID, resourceType)
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

		_, err := c.UpdateProfilePolicyAdvancedSettings(advancedSettings, profileID, resourceID, resourceType)
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

	body, err := c.DoWithLock(req, advancedSettingLockName)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	return nil
}

func (c *Client) GetAdvancedSettings(resourceID, resourceType string) (*AdvancedSettings, error) {
	profileID := ""
	resourceIDArr := strings.Split(resourceID, "/")
	if len(resourceIDArr) > 1 {
		profileID = resourceIDArr[1]
		resourceID = resourceIDArr[3]
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
		profilepolicy, err := c.GetProfilePolicyAdvancedSettings(profileID, resourceID, resourceType)
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
		profilepolicy, err := c.GetProfilePolicyAdvancedSettings(profileID, resourceID, resourceType)
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

	req, err := http.NewRequest("GET", getAppSettingUrl, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, advancedSettingLockName)
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

func (c *Client) GetProfilePolicyAdvancedSettings(profileID, policyID, resourceType string) (*AdvancedSettings, error) {

	advSettingUrl := ""
	resourceTypeArr := strings.Split(resourceType, "_")
	if strings.EqualFold(resourceTypeArr[0], "resource") {
		advSettingUrl = fmt.Sprintf("%s/resource-manager/profiles/%s/policies/%s", c.APIBaseURL, profileID, policyID)
	} else {
		advSettingUrl = fmt.Sprintf("%s/paps/%s/policies/%s?compactResponse=true", c.APIBaseURL, profileID, policyID)
	}

	req, err := http.NewRequest("GET", advSettingUrl, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, advancedSettingLockName)
	if err != nil {
		return nil, err
	}
	if string(body) == emptyString {
		return nil, ErrNotFound
	}

	profilePolicy := &AdvancedSettings{}

	err = json.Unmarshal(body, profilePolicy)
	if err != nil {
		return nil, err
	}

	return profilePolicy, nil
}

func (c *Client) UpdateProfilePolicyAdvancedSettings(profilePolicyAdvancedSettings AdvancedSettings, profileID, policyID, resourceType string) (*AdvancedSettings, error) {
	var profilePolicyBody []byte
	var err error
	profilePolicyBody, err = json.Marshal(profilePolicyAdvancedSettings)
	if err != nil {
		return nil, err
	}

	advSettingUrl := ""
	resourceTypeArr := strings.Split(resourceType, "_")
	if strings.EqualFold(resourceTypeArr[0], "resource") {
		advSettingUrl = fmt.Sprintf("%s/resource-manager/profiles/%s/policies/%s", c.APIBaseURL, profileID, policyID)
	} else {
		advSettingUrl = fmt.Sprintf("%s/paps/%s/policies/%s", c.APIBaseURL, profileID, policyID)
	}

	req, err := http.NewRequest("PATCH", advSettingUrl, strings.NewReader(string(profilePolicyBody)))
	if err != nil {
		return nil, err
	}

	_, err = c.DoWithLock(req, advancedSettingLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &profilePolicyAdvancedSettings, nil
	}

	return nil, err
}

// Get all Connections
func (c *Client) GetAllConnections(settingType string) ([]Connection, error) {
	var connectionsURL string
	if strings.EqualFold(settingType, "ITSM") {
		connectionsURL = fmt.Sprintf("%s/itsm-manager/connections", c.APIBaseURL)
	} else if strings.EqualFold(settingType, "IM") {
		connectionsURL = fmt.Sprintf("%s/im-manager/connections", c.APIBaseURL)
	} else {
		return nil, ErrNotSupported
	}

	req, err := http.NewRequest("GET", connectionsURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, advancedSettingLockName)
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

func (c *Client) GetEscalationPolicies(page int, imConnectionId, policyName string) (*EscalationPolicies, error) {
	url := fmt.Sprintf("%s/im-integration/%s/escalation-policies/search?page=%d&size=20&searchText=%s", c.APIBaseURL, imConnectionId, page, policyName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, advancedSettingLockName)
	if err != nil {
		return nil, err
	}

	var policies EscalationPolicies
	err = json.Unmarshal(body, &policies)
	if err != nil {
		return nil, err
	}
	return &policies, nil
}
