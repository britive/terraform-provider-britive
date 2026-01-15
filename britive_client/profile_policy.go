package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// GetProfilePolicy - Returns a specific policy from profile
func (c *Client) GetProfilePolicy(ctx context.Context, profileID string, policyID string) (*ProfilePolicy, error) {

	requestURL := fmt.Sprintf("%s/paps/%s/policies/%s?compactResponse=true", c.APIBaseURL, profileID, policyID)
	body, err := c.Get(ctx, requestURL, ProfileLockName)
	if err != nil {
		return nil, err
	}
	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	profilePolicy := &ProfilePolicy{}

	err = json.Unmarshal(body, profilePolicy)
	if err != nil {
		return nil, err
	}

	return profilePolicy, nil
}

// GetProfilePolicyByName - Returns a specific policy by name from profile
func (c *Client) GetProfilePolicyByName(ctx context.Context, profileID string, policyName string) (*ProfilePolicy, error) {

	requestURL := fmt.Sprintf("%s/paps/%s/policies/%s?compactResponse=true", c.APIBaseURL, profileID, policyName)
	body, err := c.Get(ctx, requestURL, ProfileLockName)
	if err != nil {
		return nil, err
	}
	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	profilePolicy := &ProfilePolicy{}

	err = json.Unmarshal(body, profilePolicy)
	if err != nil {
		return nil, err
	}

	return profilePolicy, nil
}

// CreateProfilePolicy - Add policy to profile
func (c *Client) CreateProfilePolicy(ctx context.Context, profilePolicy ProfilePolicy) (*ProfilePolicy, error) {
	url := fmt.Sprintf("%s/paps/%s/policies/%s", c.APIBaseURL, profilePolicy.ProfileID, profilePolicy.Name)
	body, err := c.Post(ctx, url, profilePolicy, ProfileLockName)
	if err != nil {
		return nil, err
	}

	pp := &ProfilePolicy{}

	err = json.Unmarshal(body, pp)
	if err != nil {
		return nil, err
	}

	return pp, nil
}

// UpdateProfilePolicy - Update profile policy
func (c *Client) UpdateProfilePolicy(ctx context.Context, profilePolicy ProfilePolicy, policyName string) (*ProfilePolicy, error) {
	url := fmt.Sprintf("%s/paps/%s/policies/%s", c.APIBaseURL, profilePolicy.ProfileID, policyName)
	_, err := c.Patch(ctx, url, profilePolicy, ProfileLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &profilePolicy, nil
	}
	return nil, err
}

// DeleteProfilePolicy - Delete policy from the profile
func (c *Client) DeleteProfilePolicy(ctx context.Context, profileID string, policyID string) error {
	url := fmt.Sprintf("%s/paps/%s/policies/%s", c.APIBaseURL, profileID, policyID)
	err := c.Delete(ctx, url, ProfileLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}

	return err
}

// RetrieveAppIdGivenProfileId - Fetch the app Id for a given profile ID
func (c *Client) RetrieveAppIdGivenProfileId(ctx context.Context, profileID string) (string, error) {
	requestURL := fmt.Sprintf("%s/paps/%s", c.APIBaseURL, profileID)
	body, err := c.Get(ctx, requestURL, ProfileLockName)
	if err != nil {
		return EmptyString, err
	}
	if string(body) == EmptyString {
		return EmptyString, ErrNotFound
	}

	application := &Application{}
	err = json.Unmarshal(body, application)
	if err != nil {
		return EmptyString, err
	}

	if application.AppContainerID == EmptyString {
		return EmptyString, ErrNotFound
	}

	return application.AppContainerID, nil
}
