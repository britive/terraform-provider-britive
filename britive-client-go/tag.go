package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetTags - Returns all user tags
func (c *Client) GetTags() (*[]Tag, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/user-tags", c.HostURL), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	tags := make([]Tag, 0)
	err = json.Unmarshal(body, &tags)
	if err != nil {
		return nil, err
	}

	return &tags, nil
}

// GetTag - Returns a specifc user tag
func (c *Client) GetTag(tagID string) (*Tag, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/user-tags/%s", c.HostURL, tagID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	tag := Tag{}
	err = json.Unmarshal(body, &tag)
	if err != nil {
		return nil, err
	}

	return &tag, nil
}

// CreateTag - Create new tag
func (c *Client) CreateTag(tag Tag) (*Tag, error) {
	utb, err := json.Marshal(tag)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/user-tags", c.HostURL), strings.NewReader(string(utb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &tag)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// UpdateTag - Updates an tag
func (c *Client) UpdateTag(tagID string, tag Tag) (*Tag, error) {
	utsb, err := json.Marshal(tag)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/user-tags/%s", c.HostURL, tagID), strings.NewReader(string(utsb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &tag)
	if err != nil {
		return nil, err
	}

	return &tag, nil
}

// DeleteTag - Deletes an tag
func (c *Client) DeleteTag(tagID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/user-tags/%s", c.HostURL, tagID), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	if err != nil {
		return err
	}

	return nil
}
