package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetTagWithOwners - Returns tag details including owner relationships
func (c *Client) GetTagWithOwners(tagID string) (*TagWithOwners, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/user-tags/%s", c.APIBaseURL, tagID), nil)
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

	var tag TagWithOwners
	if err := json.Unmarshal(body, &tag); err != nil {
		return nil, err
	}

	return &tag, nil
}

// UpdateTagOwners - Updates tag owner relationships via PATCH /api/user-tags
func (c *Client) UpdateTagOwners(tag TagWithOwners) (*TagWithOwners, error) {
	body, err := json.Marshal(tag)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/user-tags", c.APIBaseURL), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	responseBody, err := c.DoWithLock(req, tagLockName)
	if err != nil {
		return nil, err
	}

	var updatedTag TagWithOwners
	if err := json.Unmarshal(responseBody, &updatedTag); err != nil {
		return nil, err
	}

	return &updatedTag, nil
}
