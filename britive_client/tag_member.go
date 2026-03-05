package britive_client

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetTagMember - Returns a specifc member assigned to tag
func (c *Client) GetTagMember(ctx context.Context, tagID string, userID string) (*User, error) {
	url := fmt.Sprintf("%s/user-tags/%s/users/%s?filter=assigned", c.APIBaseURL, tagID, userID)
	body, err := c.Get(ctx, url, TagLockName)
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

// CreateTagMember - Add member to tag
func (c *Client) CreateTagMember(ctx context.Context, tagID string, userID string) error {
	url := fmt.Sprintf("%s/user-tags/%s/users/%s", c.APIBaseURL, tagID, userID)
	_, err := c.Post(ctx, url, nil, TagLockName)

	return err
}

// DeleteTagMember - Delete member from the tag
func (c *Client) DeleteTagMember(ctx context.Context, tagID string, userID string) error {
	url := fmt.Sprintf("%s/user-tags/%s/users/%s", c.APIBaseURL, tagID, userID)
	err := c.Delete(ctx, url, TagLockName)

	return err
}
