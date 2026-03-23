package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) CreateUpdateResourceManagerProfile(resourceManagerProfile ResourceManagerProfile, isUpdate bool) (*ResourceManagerProfile, error) {
	pb, err := json.Marshal(resourceManagerProfile)
	if err != nil {
		return nil, err
	}

	var apiMethod, url string
	if isUpdate {
		apiMethod = "PATCH"
		url = fmt.Sprintf("%s/resource-manager/profiles/%s", c.APIBaseURL, resourceManagerProfile.ProfileId)
	} else {
		apiMethod = "POST"
		url = fmt.Sprintf("%s/resource-manager/profiles", c.APIBaseURL)
	}

	req, err := http.NewRequest(apiMethod, url, strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(req, resourceManagerProfileLock)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return nil, err
	}

	if len(resp) != 0 {
		err = json.Unmarshal(resp, &resourceManagerProfile)
		if err != nil {
			return nil, err
		}
	}

	return &resourceManagerProfile, nil
}

func (c *Client) CreateUpdateResourceManagerProfileAssociations(resourceManagerProfile ResourceManagerProfile) (*ResourceManagerProfile, error) {
	pb, err := json.Marshal(resourceManagerProfile)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/resource-manager/profiles/%s/associations", c.APIBaseURL, resourceManagerProfile.ProfileId)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(req, resourceManagerProfileLock)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return nil, err
	}

	if len(resp) != 0 {
		err = json.Unmarshal(resp, &resourceManagerProfile)
		if err != nil {
			return nil, err
		}
	}

	return &resourceManagerProfile, nil
}

func (c *Client) EnableDisableResourceManagerPolicyPrioritization(profileId string, policyOrderingEnabled bool) error {
	payload := map[string]bool{
		"policyOrderingEnabled": policyOrderingEnabled,
	}
	policyOrderingEnabledPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/resource-manager/profiles/%s", c.APIBaseURL, profileId), strings.NewReader(string(policyOrderingEnabledPayload)))
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, profileLockName)
	if !(errors.Is(err, ErrNoContent)) && err != nil {
		return err
	}

	return nil
}

func (c *Client) GetResourceManagerProfile(profileId string) (*ResourceManagerProfile, error) {
	apiMethod := "GET"
	url := fmt.Sprintf("%s/resource-manager/profiles/%s", c.APIBaseURL, profileId)
	req, err := http.NewRequest(apiMethod, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(req, resourceManagerProfileLock)
	if err != nil {
		return nil, err
	}

	var resourceManagerProfile ResourceManagerProfile
	err = json.Unmarshal(resp, &resourceManagerProfile)
	if err != nil {
		return nil, err
	}

	return &resourceManagerProfile, nil
}

func (c *Client) GetResourceManagerProfileAssociations(profileId string) (*ResourceManagerProfile, error) {
	apiMethod := "GET"
	url := fmt.Sprintf("%s/resource-manager/profiles/%s/associations", c.APIBaseURL, profileId)
	req, err := http.NewRequest(apiMethod, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(req, resourceManagerProfileLock)
	if err != nil {
		return nil, err
	}

	var resourceManagerProfile ResourceManagerProfile
	err = json.Unmarshal(resp, &resourceManagerProfile)
	if err != nil {
		return nil, err
	}

	return &resourceManagerProfile, nil
}

func (c *Client) DeleteResourceManagerProfile(profileId string) error {
	url := fmt.Sprintf("%s/resource-manager/profiles/%s", c.APIBaseURL, profileId)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, resourceManagerProfileLock)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return err
	}

	return nil
}

// PrioritizePolicies - Order Policy
func (c *Client) ResourceManagerPrioritizeProfilePolicies(resourcePolicyPriority ProfilePolicyPriority) (*ProfilePolicyPriority, error) {
	policyOrder, err := json.Marshal(resourcePolicyPriority.PolicyOrder)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/resource-manager/profiles/%s/policies/order", c.APIBaseURL, resourcePolicyPriority.ProfileID), strings.NewReader(string(policyOrder)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileLockName)
	if err != nil {
		return nil, err
	}

	var profilePolicyPriority ProfilePolicyPriority
	err = json.Unmarshal(body, &profilePolicyPriority.PolicyOrder)
	if err != nil {
		return nil, err
	}

	return &profilePolicyPriority, nil
}

func (c *Client) GetResourceManagerProfilePolicies(profileId string) ([]ResourceManagerProfilePolicy, error) {
	url := fmt.Sprintf("%s/resource-manager/profiles/%s/policies", c.APIBaseURL, profileId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileLockName)
	if err != nil {
		return nil, err
	}
	var policies []ResourceManagerProfilePolicy
	err = json.Unmarshal(body, &policies)
	if err != nil {
		return nil, err
	}

	return policies, nil
}
