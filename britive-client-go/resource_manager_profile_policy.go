package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) CreateUpdateResourceManagerProfilePolicy(resourceManagerProfilePolicy ResourceManagerProfilePolicy, oldName string, isUpdate bool) (*ResourceManagerProfilePolicy, error) {
	pb, err := json.Marshal(resourceManagerProfilePolicy)
	if err != nil {
		return nil, err
	}

	var apiMethod, url string
	if isUpdate {
		apiMethod = "PATCH"
		url = fmt.Sprintf("%s/resource-manager/profiles/%s/policies/%s", c.APIBaseURL, resourceManagerProfilePolicy.ProfileID, oldName)
	} else {
		apiMethod = "POST"
		url = fmt.Sprintf("%s/resource-manager/profiles/%s/policies", c.APIBaseURL, resourceManagerProfilePolicy.ProfileID)
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
		err = json.Unmarshal(resp, &resourceManagerProfilePolicy)
		if err != nil {
			return nil, err
		}
	}

	return &resourceManagerProfilePolicy, nil
}

func (c *Client) GetResourceManagerProfilePolicy(profileID, policyName string) (*ResourceManagerProfilePolicy, error) {
	apiMethod := "GET"
	url := fmt.Sprintf("%s/resource-manager/profiles/%s/policies/%s?compactResponse=true", c.APIBaseURL, profileID, policyName)
	req, err := http.NewRequest(apiMethod, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(req, resourceManagerProfilePolicyLock)
	if err != nil {
		return nil, err
	}

	var resourceManagerProfilePolicy ResourceManagerProfilePolicy
	err = json.Unmarshal(resp, &resourceManagerProfilePolicy)
	if err != nil {
		return nil, err
	}

	return &resourceManagerProfilePolicy, nil
}

func (c *Client) DeleteResourceManagerProfilePolicy(profileID, policyID string) error {
	url := fmt.Sprintf("%s/resource-manager/profiles/%s/policies/%s", c.APIBaseURL, profileID, policyID)
	apiMethod := "DELETE"

	req, err := http.NewRequest(apiMethod, url, nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, resourceManagerProfilePolicyLock)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return err
	}

	return nil
}
