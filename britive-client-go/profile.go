package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GetProfiles - Returns all profiles
func (c *Client) GetProfiles(appContainerID string) (*[]Profile, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/apps/%s/paps", c.APIBaseURL, appContainerID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileLockName)
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

// GetProfileByName - Returns a specifc profile by name
func (c *Client) GetProfileByName(appContainerID string, name string) (*Profile, error) {
	filter := fmt.Sprintf(`name eq "%s"`, name)
	resourceURL := fmt.Sprintf(`%s/apps/%s/paps?filter=%s`, c.APIBaseURL, appContainerID, url.QueryEscape(filter))
	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == emptyString {
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

// GetProfile - Returns a specifc profile
func (c *Client) GetProfile(profileID string) (*Profile, error) {
	requestURL := fmt.Sprintf("%s/paps/%s?skipIntegrityChecks=true", c.APIBaseURL, profileID)
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileLockName)
	if err != nil {
		return nil, err
	}
	if string(body) == emptyString {
		return nil, ErrNotFound
	}

	profile := &Profile{}
	err = json.Unmarshal(body, profile)
	if err != nil {
		return nil, err
	}

	if profile == nil {
		return nil, ErrNotFound
	}

	return profile, nil
}

// CreateProfile - Create new profile
func (c *Client) CreateProfile(appContainerID string, profile Profile) (*Profile, error) {
	utb, err := json.Marshal(profile)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/apps/%s/paps", c.APIBaseURL, appContainerID), strings.NewReader(string(utb)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileLockName)
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
	profileBody, err := json.Marshal(profile)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/apps/%s/paps/%s", c.APIBaseURL, appContainerID, profileID), strings.NewReader(string(profileBody)))
	if err != nil {
		return nil, err
	}
	body, err := c.DoWithLock(req, profileLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &profile)
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

// EnableOrDisableProfile - Enable or Disable tag
func (c *Client) EnableOrDisableProfile(appContainerID string, profileID string, disabled bool) (*Profile, error) {
	var endpoint string
	if disabled {
		endpoint = "disabled-statuses"
	} else {
		endpoint = "enabled-statuses"
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/apps/%s/paps/%s/%s", c.APIBaseURL, appContainerID, profileID, endpoint), strings.NewReader(string([]byte("{}"))))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileLockName)
	if err != nil {
		return nil, err
	}

	var profile Profile
	err = json.Unmarshal(body, &profile)
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

// DeleteProfile - Delete profile
func (c *Client) DeleteProfile(appContainerID string, profileID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/apps/%s/paps/%s", c.APIBaseURL, appContainerID, profileID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, profileLockName)
	if err != nil {
		return err
	}

	return nil
}
