package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// GetProfilePolicy - Returns a specific policy from profile
func (c *Client) GetProfilePolicy(profileID string, policyID string) (*ProfilePolicy, error) {

	requestURL := fmt.Sprintf("%s/paps/%s/policies/%s?compactResponse=true", c.APIBaseURL, profileID, policyID)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileID)
	if err != nil {
		return nil, err
	}
	if string(body) == emptyString {
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
func (c *Client) GetProfilePolicyByName(profileID string, policyName string) (*ProfilePolicy, error) {

	requestURL := fmt.Sprintf("%s/paps/%s/policies/%s?compactResponse=true", c.APIBaseURL, profileID, policyName)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileID)
	if err != nil {
		return nil, err
	}
	if string(body) == emptyString {
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
func (c *Client) CreateProfilePolicy(profilePolicy ProfilePolicy) (*ProfilePolicy, error) {
	var profilePolicyBody []byte
	var err error
	profilePolicyBody, err = json.Marshal(profilePolicy)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/paps/%s/policies/%s", c.APIBaseURL, profilePolicy.ProfileID, profilePolicy.Name), strings.NewReader(string(profilePolicyBody)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profilePolicy.ProfileID)

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
func (c *Client) UpdateProfilePolicy(profilePolicy ProfilePolicy, policyName string) (*ProfilePolicy, error) {
	var profilePolicyBody []byte
	var err error
	profilePolicyBody, err = json.Marshal(profilePolicy)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/paps/%s/policies/%s", c.APIBaseURL, profilePolicy.ProfileID, policyName), strings.NewReader(string(profilePolicyBody)))
	if err != nil {
		return nil, err
	}

	_, err = c.DoWithLock(req, profilePolicy.ProfileID)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &profilePolicy, nil
	}
	return nil, err
}

// DeleteProfilePolicy - Delete policy from the profile
func (c *Client) DeleteProfilePolicy(profileID string, policyID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/paps/%s/policies/%s", c.APIBaseURL, profileID, policyID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, profileID)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}

	return err
}
