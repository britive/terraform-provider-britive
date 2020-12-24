package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetAssignedProfileTags - Returns all tags assigned to profile
func (c *Client) GetAssignedProfileTags(profileID string) (*[]ProfileTag, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/paps/%s/user-tags?filter=assigned", c.APIBaseURL, profileID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequestWithLock(req, profileID)
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
	profileTags, err := c.GetAssignedProfileTags(profileID)
	if err != nil {
		return nil, err
	}

	if profileTags == nil || len(*profileTags) == 0 {
		return nil, ErrNotFound
	}

	var profileTag *ProfileTag
	for _, t := range *profileTags {
		if strings.ToLower(t.TagID) == strings.ToLower(tagID) {
			profileTag = &t
			break
		}
	}

	if profileTag == nil {
		return nil, ErrNotFound
	}

	return profileTag, nil
}

func (c *Client) createOrUpdateProfileTag(method string, profileTag ProfileTag) (*ProfileTag, error) {
	var profileTagBody []byte
	var err error
	if profileTag.AccessPeriod == nil {
		profileTagBody = []byte("{}")
	} else {
		profileTagBody, err = json.Marshal(*profileTag.AccessPeriod)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, fmt.Sprintf("%s/paps/%s/user-tags/%s", c.APIBaseURL, profileTag.ProfileID, profileTag.TagID), strings.NewReader(string(profileTagBody)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequestWithLock(req, profileTag.ProfileID)
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

// UpdateProfileTag - Update profile tag attributes
func (c *Client) UpdateProfileTag(profileTag ProfileTag) (*ProfileTag, error) {
	return c.createOrUpdateProfileTag("PATCH", profileTag)
}

// DeleteProfileTag - Delete tag from the profile
func (c *Client) DeleteProfileTag(profileID string, tagID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/paps/%s/user-tags/%s", c.APIBaseURL, profileID, tagID), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequestWithLock(req, profileID)

	return err
}
