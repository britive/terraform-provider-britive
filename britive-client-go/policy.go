package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// GetPolicyByName - Returns a specific policy by name
func (c *Client) GetPolicyByName(name string) (*Policy, error) {

	requestURL := fmt.Sprintf("%s/v1/policy-admin/policies/%s?compactResponse=true", c.APIBaseURL, name)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, policyLockName)
	if err != nil {
		return nil, err
	}
	if string(body) == emptyString {
		return nil, ErrNotFound
	}

	policy := &Policy{}

	err = json.Unmarshal(body, policy)
	if err != nil {
		return nil, err
	}

	return policy, nil
}

// GetPolicy - Returns a specific policy by id
func (c *Client) GetPolicy(policyID string) (*Policy, error) {

	requestURL := fmt.Sprintf("%s/v1/policy-admin/policies/%s?compactResponse=true", c.APIBaseURL, policyID)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, policyLockName)
	if err != nil {
		return nil, err
	}
	if string(body) == emptyString {
		return nil, ErrNotFound
	}

	policy := &Policy{}

	err = json.Unmarshal(body, policy)
	if err != nil {
		return nil, err
	}

	if policy == nil {
		return nil, ErrNotFound
	}

	return policy, nil
}

// CreatePolicy - Add new policy
func (c *Client) CreatePolicy(policy Policy) (*Policy, error) {
	policyBody, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/policy-admin/policies", c.APIBaseURL), strings.NewReader(string(policyBody)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, policyLockName)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &policy)
	if err != nil {
		return nil, err
	}

	return &policy, nil
}

// UpdatePolicy - Update policy
func (c *Client) UpdatePolicy(policy Policy, policyName string) (*Policy, error) {
	var policyBody []byte
	var err error
	policyBody, err = json.Marshal(policy)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/v1/policy-admin/policies/%s", c.APIBaseURL, policyName), strings.NewReader(string(policyBody)))
	if err != nil {
		return nil, err
	}

	_, err = c.DoWithLock(req, policyLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &policy, nil
	}
	return nil, err
}

// DeletePolicy - Delete policy
func (c *Client) DeletePolicy(policyID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/policy-admin/policies/%s", c.APIBaseURL, policyID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, policyLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}

	return err
}
