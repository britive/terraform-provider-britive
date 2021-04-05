package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// GetUser - Returns user by user id
func (c *Client) GetUser(userID string) (*User, error) {
	resourceURL := fmt.Sprintf("%s/users/%s", c.APIBaseURL, userID)
	return c.getUser(resourceURL)
}

// GetUserByName - Returns user by username
func (c *Client) GetUserByName(username string) (*User, error) {
	filter := fmt.Sprintf(`username eq "%s"`, username)
	resourceURL := fmt.Sprintf(`%s/users?filter=%s`, c.APIBaseURL, url.QueryEscape(filter))
	return c.getUser(resourceURL)
}

func (c *Client) getUser(resourceURL string) (*User, error) {
	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if string(body) == emptyString {
		return nil, ErrNotFound
	}

	user := &User{}
	err = json.Unmarshal(body, user)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrNotFound
	}

	return user, nil
}
