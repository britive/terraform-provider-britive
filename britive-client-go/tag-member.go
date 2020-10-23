package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// GetTagMembers - Returns all users assigned to user tag
func (c *Client) GetTagMembers(tagID string) (*[]User, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/user-tags/%s/users?filter=assigned", c.HostURL, tagID), nil)
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

// GetTagMember - Returns a specifc user from user tag
func (c *Client) GetTagMember(tagID string, userID string) (*User, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/user-tags/%s/users/%s?filter=assigned", c.HostURL, tagID, userID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	user := &User{}
	err = json.Unmarshal(body, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// CreateTagMember - Add member to user tag
func (c *Client) CreateTagMember(tagID string, userID string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/user-tags/%s/users/%s", c.HostURL, tagID, userID), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)

	return err
}

// DeleteTagMember - Delete member from the tag
func (c *Client) DeleteTagMember(tagID string, userID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/user-tags/%s/users/%s", c.HostURL, tagID, userID), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)

	return err
}
