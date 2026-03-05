package britive_client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// GetTagByName - Returns a specifc tag by name
func (c *Client) GetTagByName(ctx context.Context, name string) (*Tag, error) {
	filter := fmt.Sprintf(`name eq "%s"`, name)
	resourceURL := fmt.Sprintf(`%s/user-tags?filter=%s`, c.APIBaseURL, url.QueryEscape(filter))
	body, err := c.Get(ctx, resourceURL, TagLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	tags := make([]Tag, 0)
	err = json.Unmarshal(body, &tags)
	if err != nil {
		return nil, err
	}

	if len(tags) == 0 {
		return nil, ErrNotFound
	}

	return &tags[0], nil
}

// GetTag - Returns a specifc tag by id
func (c *Client) GetTag(ctx context.Context, tagID string) (*Tag, error) {
	url := fmt.Sprintf("%s/user-tags/%s", c.APIBaseURL, tagID)
	body, err := c.Get(ctx, url, TagLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	tag := &Tag{}
	err = json.Unmarshal(body, tag)
	if err != nil {
		return nil, err
	}

	return tag, nil
}

// CreateTag - Create new tag
func (c *Client) CreateTag(ctx context.Context, tag Tag) (*Tag, error) {
	url := fmt.Sprintf("%s/user-tags", c.APIBaseURL)
	body, err := c.Post(ctx, url, tag, TagLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &tag)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// UpdateTag - Update tag
func (c *Client) UpdateTag(ctx context.Context, tagID string, tag Tag) (*Tag, error) {
	url := fmt.Sprintf("%s/user-tags/%s", c.APIBaseURL, tagID)
	body, err := c.Patch(ctx, url, tag, TagLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &tag)
	if err != nil {
		return nil, err
	}

	return &tag, nil
}

// EnableOrDisableTag - Enable or Disable tag
func (c *Client) EnableOrDisableTag(ctx context.Context, tagID string, disabled bool) (*Tag, error) {
	var endpoint string
	if disabled {
		endpoint = "disabled-statuses"
	} else {
		endpoint = "enabled-statuses"
	}
	var emptyInterface interface{}
	url := fmt.Sprintf("%s/user-tags/%s/%s", c.APIBaseURL, tagID, endpoint)
	body, err := c.Post(ctx, url, emptyInterface, TagLockName)
	if err != nil {
		return nil, err
	}

	var tag Tag
	err = json.Unmarshal(body, &tag)
	if err != nil {
		return nil, err
	}

	return &tag, nil
}

// DeleteTag - Delete tag
func (c *Client) DeleteTag(ctx context.Context, tagID string) error {
	url := fmt.Sprintf("%s/user-tags/%s", c.APIBaseURL, tagID)
	err := c.Delete(ctx, url, TagLockName)
	if err != nil {
		return err
	}

	return nil
}
