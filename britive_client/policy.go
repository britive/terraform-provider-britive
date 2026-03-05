package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// GetPolicyByName - Returns a specific policy by name
func (c *Client) GetPolicyByName(ctx context.Context, name string) (*Policy, error) {

	requestURL := fmt.Sprintf("%s/v1/policy-admin/policies/%s?compactResponse=true", c.APIBaseURL, name)
	body, err := c.Get(ctx, requestURL, PolicyLockName)
	if err != nil {
		return nil, err
	}
	if string(body) == EmptyString {
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
func (c *Client) GetPolicy(ctx context.Context, policyID string) (*Policy, error) {

	requestURL := fmt.Sprintf("%s/v1/policy-admin/policies/%s?compactResponse=true", c.APIBaseURL, policyID)
	body, err := c.Get(ctx, requestURL, PolicyLockName)
	if err != nil {
		return nil, err
	}
	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	policy := &Policy{}

	err = json.Unmarshal(body, policy)
	if err != nil {
		return nil, err
	}

	return policy, nil
}

// CreatePolicy - Add new policy
func (c *Client) CreatePolicy(ctx context.Context, policy Policy) (*Policy, error) {
	url := fmt.Sprintf("%s/v1/policy-admin/policies", c.APIBaseURL)
	body, err := c.Post(ctx, url, policy, PolicyLockName)
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
func (c *Client) UpdatePolicy(ctx context.Context, policy Policy, policyName string) (*Policy, error) {
	url := fmt.Sprintf("%s/v1/policy-admin/policies/%s", c.APIBaseURL, policyName)
	_, err := c.Patch(ctx, url, policy, PolicyLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &policy, nil
	}
	return nil, err
}

// DeletePolicy - Delete policy
func (c *Client) DeletePolicy(ctx context.Context, policyID string) error {
	url := fmt.Sprintf("%s/v1/policy-admin/policies/%s", c.APIBaseURL, policyID)
	err := c.Delete(ctx, url, PolicyLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}

	return err
}
