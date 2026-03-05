package britive_client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// GetUser - Returns user by user id
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	resourceURL := fmt.Sprintf("%s/users/%s", c.APIBaseURL, userID)
	return c.getUser(ctx, resourceURL)
}

// GetUserByName - Returns user by username
func (c *Client) GetUserByName(ctx context.Context, username string) (*User, error) {
	filter := fmt.Sprintf(`username eq "%s"`, username)
	resourceURL := fmt.Sprintf(`%s/users?filter=%s`, c.APIBaseURL, url.QueryEscape(filter))
	return c.getUser(ctx, resourceURL)
}

func (c *Client) getUser(ctx context.Context, resourceURL string) (*User, error) {
	body, err := c.Get(ctx, resourceURL, UserLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	user := &User{}
	err = json.Unmarshal(body, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}
