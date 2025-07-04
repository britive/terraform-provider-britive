package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Create Advanced Setting
func (c *Client) CreateUpdateAdvancedSettings(d *schema.ResourceData, advancedSettings AdvancedSettings) error {
	resourceID := ""
	resourceType := d.Get("resource_type").(string)
	resourceType = strings.ToUpper(resourceType)
	policyID := ""

	rawResourceID := d.Get("resource_id")
	resourceID = rawResourceID.(string)
	resourceIDArr := strings.Split(resourceID, "/")
	if len(resourceIDArr) > 1 {
		policyID = resourceIDArr[3]
		resourceID = resourceIDArr[1]
	}

	apiMethod := ""
	advancedSettingURL := ""

	switch resourceType {
	case "APPLICATION":
		advancedSettingURL = fmt.Sprintf("%s/apps/%s/advanced-settings", c.APIBaseURL, resourceID)
		if d.Id() != "" {
			apiMethod = "PUT"
		} else {
			apiMethod = "POST"
		}
	case "PROFILE":
		advancedSettingURL = fmt.Sprintf("%s/paps/%s/advanced-settings", c.APIBaseURL, resourceID)
		apiMethod = "PUT"
	case "PROFILE_POLICY":
		if policyID == "" {
			return fmt.Errorf("unable to fetch profile policy, policyId is empty")
		}

		_, err := c.UpdateProfilePolicyAdvancedSettings(advancedSettings, resourceID, policyID, resourceType)
		if err != nil {
			return fmt.Errorf("Error : %v", err)
		}
		return nil
	case "RESOURCE_MANAGER_PROFILE":
		advancedSettingURL = fmt.Sprintf("%s/resource-manager/profile/%s/advanced-settings", c.APIBaseURL, resourceID)
		if d.Id() != "" {
			apiMethod = "PUT"
		} else {
			apiMethod = "POST"
		}
	case "RESOURCE_MANAGER_PROFILE_POLICY":
		if policyID == "" {
			return fmt.Errorf("unable to fetch profile policy, policyId is empty")
		}

		_, err := c.UpdateProfilePolicyAdvancedSettings(advancedSettings, resourceID, policyID, resourceType)
		if err != nil {
			return fmt.Errorf("Error : %v", err)
		}
		return nil
	default:
		return fmt.Errorf("Resource Type '%s' not supported", resourceType)
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

func (c *Client) GetAdvancedSettings(d *schema.ResourceData, resourceID string, resourceType string) (*AdvancedSettings, error) {
	getAppSettingUrl := ""
	switch resourceType {
	case "APPLICATION":
		getAppSettingUrl = fmt.Sprintf("%s/apps/%s/advanced-settings", c.APIBaseURL, resourceID)
	case "PROFILE":
		getAppSettingUrl = fmt.Sprintf("%s/paps/%s/advanced-settings", c.APIBaseURL, resourceID)
	case "PROFILE_POLICY":
		rawID := d.Get("resource_id")
		rawIdArr := strings.Split(rawID.(string), "/")
		profileID := rawIdArr[1]
		profilepolicy, err := c.GetProfilePolicyAdvancedSettings(profileID, resourceID, resourceType)
		if err != nil {
			return nil, fmt.Errorf("Unable to get profile policy details : '%v'", err)
		}
		advancedSettings := AdvancedSettings{
			Settings: profilepolicy.Settings,
		}
		return &advancedSettings, nil
	case "RESOURCE_MANAGER_PROFILE":
		getAppSettingUrl = fmt.Sprintf("%s/resource-manager/profile/%s/advanced-settings", c.APIBaseURL, resourceID)
	case "RESOURCE_MANAGER_PROFILE_POLICY":
		rawID := d.Get("resource_id")
		rawIdArr := strings.Split(rawID.(string), "/")
		profileID := rawIdArr[1]
		profilepolicy, err := c.GetProfilePolicyAdvancedSettings(profileID, resourceID, resourceType)
		if err != nil {
			return nil, fmt.Errorf("Unable to get profile policy details : '%v'", err)
		}
		advancedSettings := AdvancedSettings{
			Settings: profilepolicy.Settings,
		}
		return &advancedSettings, nil
	default:
		return nil, fmt.Errorf("ResourceType '%s' is not supported", resourceType)
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
	if resourceTypeArr[0] == "RESOURCE" {
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
	if resourceTypeArr[0] == "RESOURCE" {
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
func (c *Client) GetAllConnections() ([]Connection, error) {
	connectionsURL := fmt.Sprintf("%s/itsm-manager/connections", c.APIBaseURL)

	req, err := http.NewRequest("GET", connectionsURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, applicationLockName)
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
