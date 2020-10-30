package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetProfileTags - Returns all tags assigned to profile
func (c *Client) GetProfileTags(profileID string) (*[]ProfileTag, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/paps/%s/user-tags?filter=assigned", c.HostURL, profileID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	profileTags := make([]ProfileTag, 0)
	err = json.Unmarshal(body, &profileTags)
	if err != nil {
		return nil, err
	}

	return &profileTags, nil
}

// GetProfileTag - Returns a specifc tag from profile
func (c *Client) GetProfileTag(profileID string, tagID string) (*ProfileTag, error) {
	//TODO: Warning Recursion - Get single instead of array
	profileTags, err := c.GetProfileTags(profileID)
	if err != nil {
		return nil, err
	}

	if profileTags == nil || len(*profileTags) == 0 {
		return nil, fmt.Errorf("No profiles tags matching for the resource /paps/%s/user-tags/%s", profileID, tagID)
	}

	var profileTag *ProfileTag
	for _, t := range *profileTags {
		if strings.ToLower(t.TagID) == strings.ToLower(tagID) {
			profileTag = &t
			break
		}
	}

	if profileTag == nil {
		return nil, fmt.Errorf("No profiles tags matching for the resource /paps/%s/user-tags/%s", profileID, tagID)
	}

	return profileTag, nil
}

func (c *Client) createOrUpdateProfileTag(method string, profileTag ProfileTag) (*ProfileTag, error) {
	var ptapb []byte
	var err error
	if profileTag.AccessPeriod == nil {
		ptapb = []byte("{}")
	} else {
		ptapb, err = json.Marshal(*profileTag.AccessPeriod)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, fmt.Sprintf("%s/paps/%s/user-tags/%s", c.HostURL, profileTag.ProfileID, profileTag.TagID), strings.NewReader(string(ptapb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	pt := &ProfileTag{}
	err = json.Unmarshal(body, pt)
	if err != nil {
		return nil, err
	}

	return pt, nil
}

// CreateProfileTag - Add tag to profile
func (c *Client) CreateProfileTag(profileTag ProfileTag) (*ProfileTag, error) {
	return c.createOrUpdateProfileTag("POST", profileTag)
}

// UpdateProfileTag - Add tag to profile
func (c *Client) UpdateProfileTag(profileTag ProfileTag) (*ProfileTag, error) {
	return c.createOrUpdateProfileTag("PATCH", profileTag)
}

// DeleteProfileTag - Delete tag from the profile
func (c *Client) DeleteProfileTag(profileID string, tagID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/paps/%s/user-tags/%s", c.HostURL, profileID, tagID), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)

	return err
}
