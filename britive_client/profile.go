package britive_client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *Client) CreateProfile(ctx context.Context, appContainerID string, profile Profile) (*Profile, error) {
	url := fmt.Sprintf("%s/apps/%s/paps", c.APIBaseURL, appContainerID)
	resp, err := c.Post(ctx, url, profile, ProfileLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, &profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (c *Client) GetProfile(ctx context.Context, profileID string) (*Profile, error) {
	url := fmt.Sprintf("%s/paps/%s?skipIntegrityChecks=true", c.APIBaseURL, profileID)
	resp, err := c.Get(ctx, url, ProfileLockName)
	if err != nil {
		return nil, err
	}
	if string(resp) == "" {
		return nil, ErrNotFound
	}

	profile := &Profile{}
	err = json.Unmarshal(resp, profile)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

// GetProfileByName - Returns a specifc profile by name
func (c *Client) GetProfileByName(ctx context.Context, appContainerID string, name string) (*Profile, error) {
	filter := fmt.Sprintf(`name eq "%s"`, name)
	resourceURL := fmt.Sprintf(`%s/apps/%s/paps?filter=%s`, c.APIBaseURL, appContainerID, url.QueryEscape(filter))
	body, err := c.Get(ctx, resourceURL, ProfileLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	profiles := make([]Profile, 0)
	err = json.Unmarshal(body, &profiles)
	if err != nil {
		return nil, err
	}

	if len(profiles) == 0 {
		return nil, ErrNotFound
	}

	return &profiles[0], nil
}

func (c *Client) UpdateProfile(ctx context.Context, appContainerID string, profileID string, profile Profile) (*Profile, error) {
	url := fmt.Sprintf("%s/apps/%s/paps/%s", c.APIBaseURL, appContainerID, profileID)
	resp, err := c.Patch(ctx, url, profile, ProfileLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, &profile)
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

// EnableOrDisableProfile - Enable or Disable tag
func (c *Client) EnableOrDisableProfile(ctx context.Context, appContainerID string, profileID string, disabled bool) (*Profile, error) {
	var endpoint string
	if disabled {
		endpoint = "disabled-statuses"
	} else {
		endpoint = "enabled-statuses"
	}
	url := fmt.Sprintf("%s/apps/%s/paps/%s/%s", c.APIBaseURL, appContainerID, profileID, endpoint)
	resp, err := c.Post(ctx, url, struct{}{}, ProfileLockName)
	if err != nil {
		return nil, err
	}

	var profile Profile
	err = json.Unmarshal(resp, &profile)
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

func (c *Client) DeleteProfile(ctx context.Context, appContainerID string, profileID string) error {
	url := fmt.Sprintf("%s/apps/%s/paps/%s", c.APIBaseURL, appContainerID, profileID)
	err := c.Delete(ctx, url, ProfileLockName)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GetProfilePolicies(ctx context.Context, profileId string) ([]ProfilePolicy, error) {
	url := fmt.Sprintf("%s/paps/%s/policies", c.APIBaseURL, profileId)
	body, err := c.Get(ctx, url, ProfileLockName)
	if err != nil {
		return nil, err
	}

	var policies []ProfilePolicy
	err = json.Unmarshal(body, &policies)
	if err != nil {
		return nil, err
	}

	return policies, nil
}

func (c *Client) GetProfileSummary(ctx context.Context, profileID string) (*ProfileSummary, error) {
	requestURL := fmt.Sprintf("%s/paps/%s?view=summary", c.APIBaseURL, profileID)
	body, err := c.Get(ctx, requestURL, ProfileLockName)
	if err != nil {
		return nil, err
	}
	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	profileSummary := &ProfileSummary{}
	err = json.Unmarshal(body, profileSummary)
	if err != nil {
		return nil, err
	}

	return profileSummary, nil
}

// EnablePolicyOrdering - Enable Policy Order
func (c *Client) EnableDisablePolicyPrioritization(ctx context.Context, profile ProfileSummary) (*ProfileSummary, error) {
	url := fmt.Sprintf("%s/paps/%s", c.APIBaseURL, profile.PapId)
	resp, err := c.Patch(ctx, url, profile, ProfileLockName)
	if err != nil {
		return nil, err
	}

	var profileSummary ProfileSummary

	err = json.Unmarshal(resp, &profileSummary)
	if err != nil {
		return nil, err
	}

	return &profileSummary, nil
}

// PrioritizePolicies - Order Policy
func (c *Client) PrioritizePolicies(ctx context.Context, resourcePolicyPriority ProfilePolicyPriority) (*ProfilePolicyPriority, error) {
	url := fmt.Sprintf("%s/paps/%s/policies/order", c.APIBaseURL, resourcePolicyPriority.ProfileID)
	body, err := c.Post(ctx, url, resourcePolicyPriority.PolicyOrder, ProfileLockName)
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
