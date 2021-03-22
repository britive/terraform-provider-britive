package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// GetProfileIdentities - Returns all identities assigned to profile
func (c *Client) GetProfileIdentities(profileID string) (*[]ProfileIdentity, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/paps/%s/users?filter=assigned", c.APIBaseURL, profileID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileID)
	if err != nil {
		return nil, err
	}

	profileIdentities := make([]ProfileIdentity, 0)
	err = json.Unmarshal(body, &profileIdentities)
	if err != nil {
		return nil, err
	}

	return &profileIdentities, nil
}

// GetProfileIdentity - Returns a specifc identity from profile
func (c *Client) GetProfileIdentity(profileID string, userID string) (*ProfileIdentity, error) {
	requestURL := fmt.Sprintf("%s/paps/%s/users/%s", c.APIBaseURL, profileID, userID)
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

	profileIdentities := []ProfileIdentity{}
	err = json.Unmarshal(body, &profileIdentities)
	if err != nil {
		return nil, err
	}

	if len(profileIdentities) == 0 {
		return nil, ErrNotFound
	}

	return &profileIdentities[0], nil
}

func (c *Client) createOrUpdateProfileIdentity(method string, profileIdentity ProfileIdentity) (*ProfileIdentity, error) {
	var profileIdentityBody []byte
	var err error
	var emptyTime = time.Time{}
	if profileIdentity.AccessPeriod == nil || (profileIdentity.AccessPeriod.Start == emptyTime && profileIdentity.AccessPeriod.End == emptyTime) {
		profileIdentityBody = []byte("{}")
	} else {
		profileIdentityBody, err = json.Marshal(*profileIdentity.AccessPeriod)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, fmt.Sprintf("%s/paps/%s/users/%s", c.APIBaseURL, profileIdentity.ProfileID, profileIdentity.UserID), strings.NewReader(string(profileIdentityBody)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileIdentity.ProfileID)
	if err != nil {
		return nil, err
	}
	pt := &ProfileIdentity{}
	err = json.Unmarshal(body, pt)
	if err != nil {
		return nil, err
	}

	return pt, nil
}

// CreateProfileIdentity - Add identity to profile
func (c *Client) CreateProfileIdentity(profileIdentity ProfileIdentity) (*ProfileIdentity, error) {
	return c.createOrUpdateProfileIdentity("POST", profileIdentity)
}

// UpdateProfileIdentity - Update profile identity properties
func (c *Client) UpdateProfileIdentity(profileIdentity ProfileIdentity) (*ProfileIdentity, error) {
	return c.createOrUpdateProfileIdentity("PATCH", profileIdentity)
}

// DeleteProfileIdentity - Delete identity from the profile
func (c *Client) DeleteProfileIdentity(profileID string, userID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/paps/%s/users/%s", c.APIBaseURL, profileID, userID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, profileID)

	return err
}
