package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// GetProfileIdentities - Returns all users assigned to profile
func (c *Client) GetProfileIdentities(profileID string) (*[]User, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/paps/%s/users?filter=assigned", c.HostURL, profileID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	users := make([]User, 0)
	err = json.Unmarshal(body, &users)
	if err != nil {
		return nil, err
	}

	return &users, nil
}

// GetProfileIdentity - Returns a specifc user from profile
func (c *Client) GetProfileIdentity(profileID string, userID string) (*User, error) {
	requestURL := fmt.Sprintf("%s/paps/%s/users/%s?filter=assigned", c.HostURL, profileID, userID)
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	//TODO: Warning Recursion - Get single instead of array
	users := []User{}
	err = json.Unmarshal(body, &users)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("No identities associated with profile for the resource %s", requestURL)
	}

	return &users[0], nil
}

// CreateProfileIdentity - Add member to profile
func (c *Client) CreateProfileIdentity(profileID string, userID string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/paps/%s/users/%s", c.HostURL, profileID, userID), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)

	return err
}

// DeleteProfileIdentity - Delete member from the profile
func (c *Client) DeleteProfileIdentity(profileID string, userID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/paps/%s/users/%s", c.HostURL, profileID, userID), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)

	return err
}
