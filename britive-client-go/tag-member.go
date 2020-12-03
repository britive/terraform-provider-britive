package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// GetAssignedTagMembers - Returns all members assigned to tag
func (c *Client) GetAssignedTagMembers(tagID string) (*[]User, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/user-tags/%s/users?filter=assigned", c.APIBaseURL, tagID), nil)
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

// GetTagMember - Returns a specifc member assigned to tag
func (c *Client) GetTagMember(tagID string, userID string) (*User, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/user-tags/%s/users/%s?filter=assigned", c.APIBaseURL, tagID, userID), nil)
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

// CreateTagMember - Add member to tag
func (c *Client) CreateTagMember(tagID string, userID string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/user-tags/%s/users/%s", c.APIBaseURL, tagID, userID), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)

	return err
}

// DeleteTagMember - Delete member from the tag
func (c *Client) DeleteTagMember(tagID string, userID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/user-tags/%s/users/%s", c.APIBaseURL, tagID, userID), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)

	return err
}
