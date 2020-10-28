package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetProfileTags - Returns all tags assigned to profile
func (c *Client) GetProfileTags(profileID string) (*[]Tag, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/paps/%s/user-tags?filter=assigned", c.HostURL, profileID), nil)
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

// GetProfileTag - Returns a specifc tag from profile
func (c *Client) GetProfileTag(profileID string, tagID string) (*Tag, error) {
	//TODO: Warning Recursion - Get single instead of array
	tags, err := c.GetProfileTags(profileID)
	if err != nil {
		return nil, err
	}

	var tag *Tag
	for _, t := range *tags {
		if strings.ToLower(t.ID) == strings.ToLower(tagID) {
			tag = &t
			break
		}
	}

	return tag, nil
}

// CreateProfileTag - Add tag to profile
func (c *Client) CreateProfileTag(profileID string, tagID string, timePeriod *TimePeriod) (err error) {
	var utsb []byte
	if timePeriod == nil {
		utsb = []byte("{}")
	} else {
		utsb, err = json.Marshal(timePeriod)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/paps/%s/user-tags/%s", c.HostURL, profileID, tagID), strings.NewReader(string(utsb)))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)

	return err
}

// DeleteProfileTag - Delete member from the profile
func (c *Client) DeleteProfileTag(profileID string, tagID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/paps/%s/user-tags/%s", c.HostURL, profileID, tagID), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)

	return err
}
