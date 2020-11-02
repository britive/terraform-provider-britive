package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// GetUsers - Returns all users
func (c *Client) GetUsers() (*[]User, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/users", c.HostURL), nil)
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

// GetUser - Returns user by user id
func (c *Client) GetUser(userID string) (*User, error) {
	resourceURL := fmt.Sprintf("%s/users/%s", c.HostURL, userID)
	return c.getUser(resourceURL)
}

// GetUserByName - Returns user by username
func (c *Client) GetUserByName(username string) (*User, error) {
	filter := fmt.Sprintf(`username eq "%s"`, username)
	resourceURL := fmt.Sprintf(`%s/users?filter=%s`, c.HostURL, url.QueryEscape(filter))
	return c.getUser(resourceURL)
}

func (c *Client) getUser(resourceURL string) (*User, error) {
	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	user := User{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
