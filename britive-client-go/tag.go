package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GetTags - Returns all tags
func (c *Client) GetTags() (*[]Tag, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/user-tags", c.APIBaseURL), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
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

// GetTagByName - Returns a specifc tag by name
func (c *Client) GetTagByName(name string) (*Tag, error) {
	filter := fmt.Sprintf(`name eq "%s"`, name)
	resourceURL := fmt.Sprintf(`%s/user-tags?filter=%s`, c.APIBaseURL, url.QueryEscape(filter))
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
func (c *Client) GetTag(tagID string) (*Tag, error) {
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

	tag := &Tag{}
	err = json.Unmarshal(body, tag)
	if err != nil {
		return nil, err
	}

	if tag == nil {
		return nil, ErrNotFound
	}

	return tag, nil
}

// CreateTag - Create new tag
func (c *Client) CreateTag(tag Tag) (*Tag, error) {
	utb, err := json.Marshal(tag)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/user-tags", c.APIBaseURL), strings.NewReader(string(utb)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, tagLockName)
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
func (c *Client) UpdateTag(tagID string, tag Tag) (*Tag, error) {
	tagBody, err := json.Marshal(tag)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/user-tags/%s", c.APIBaseURL, tagID), strings.NewReader(string(tagBody)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, tagLockName)
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
func (c *Client) EnableOrDisableTag(tagID string, disabled bool) (*Tag, error) {
	var endpoint string
	if disabled {
		endpoint = "disabled-statuses"
	} else {
		endpoint = "enabled-statuses"
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/user-tags/%s/%s", c.APIBaseURL, tagID, endpoint), strings.NewReader(string([]byte("{}"))))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, tagLockName)
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
func (c *Client) DeleteTag(tagID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/user-tags/%s", c.APIBaseURL, tagID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, tagLockName)
	if err != nil {
		return err
	}

	return nil
}
