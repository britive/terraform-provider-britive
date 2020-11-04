package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetProfiles - Returns all user profiles
func (c *Client) GetProfiles(appContainerID string) (*[]Profile, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/apps/%s/paps", c.HostURL, appContainerID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	profiles := make([]Profile, 0)
	err = json.Unmarshal(body, &profiles)
	if err != nil {
		return nil, err
	}

	return &profiles, nil
}

// GetProfile - Returns a specifc user profile
func (c *Client) GetProfile(profileID string) (*Profile, error) {
	requestURL := fmt.Sprintf("%s/paps/%s", c.HostURL, profileID)
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	profile := &Profile{}
	err = json.Unmarshal(body, profile)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

// CreateProfile - Create new profile
func (c *Client) CreateProfile(appContainerID string, profile Profile) (*Profile, error) {
	utb, err := json.Marshal(profile)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/apps/%s/paps", c.HostURL, appContainerID), strings.NewReader(string(utb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// UpdateProfile - Updates profile
func (c *Client) UpdateProfile(appContainerID string, profileID string, profile Profile) (*Profile, error) {
	utsb, err := json.Marshal(profile)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/apps/%s/paps/%s", c.HostURL, appContainerID, profileID), strings.NewReader(string(utsb)))
	if err != nil {
		return nil, err
	}
	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &profile)
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

// DeleteProfile - Deletes profile
func (c *Client) DeleteProfile(appContainerID string, profileID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/apps/%s/paps/%s", c.HostURL, appContainerID, profileID), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	if err != nil {
		return err
	}

	return nil
}
