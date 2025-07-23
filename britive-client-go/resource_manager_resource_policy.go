package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) CreateUpdateResourceManagerResourcePolicy(resourcePolicy ResourceManagerResourcePolicy, oldName string, isUpdate bool) (*ResourceManagerResourcePolicy, error) {
	pb, err := json.Marshal(resourcePolicy)
	if err != nil {
		return nil, err
	}

	var apiMethod, url string
	if isUpdate {
		apiMethod = "PATCH"
		url = fmt.Sprintf("%s/resource-manager/policies/%s", c.APIBaseURL, oldName)
	} else {
		apiMethod = "POST"
		url = fmt.Sprintf("%s/resource-manager/policies", c.APIBaseURL)
	}

	req, err := http.NewRequest(apiMethod, url, strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(req, resourceManagerResourcePolicy)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return nil, err
	}

	if len(resp) != 0 {
		err = json.Unmarshal(resp, &resourcePolicy)
		if err != nil {
			return nil, err
		}
	}

	return &resourcePolicy, nil
}

func (c *Client) GetResourceManagerResourcePolicy(policyName string) (*ResourceManagerResourcePolicy, error) {
	apiMethod := "GET"
	url := fmt.Sprintf("%s/resource-manager/policies/%s?compactResponse=true", c.APIBaseURL, policyName)
	req, err := http.NewRequest(apiMethod, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(req, resourceManagerResourcePolicy)
	if err != nil {
		return nil, err
	}

	var resourcePolicy ResourceManagerResourcePolicy
	err = json.Unmarshal(resp, &resourcePolicy)
	if err != nil {
		return nil, err
	}

	return &resourcePolicy, nil
}

func (c *Client) DeleteResourceManagerResourcePolicy(policyID string) error {
	url := fmt.Sprintf("%s/resource-manager/policies/%s", c.APIBaseURL, policyID)
	apiMethod := "DELETE"

	req, err := http.NewRequest(apiMethod, url, nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, resourceManagerResourcePolicy)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return err
	}

	return nil
}
