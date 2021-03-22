package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetProfileTag - Returns a specifc tag from profile
func (c *Client) GetProfileTag(profileID string, tagID string) (*ProfileTag, error) {
	endpoint := fmt.Sprintf("paps/%s/user-tags", profileID)
	filter := fmt.Sprintf(`id eq %s`, tagID)

	profileTags := make([]ProfileTag, 0)

	err := client.NewQueryRequest().
		WithLock(profileID).
		WithFilter(filter).
		WithResult(&profileTags).
		Query(endpoint)

	if err != nil {
		return nil, err
	}
	if len(profileTags) == 0 {
		return nil, ErrNotFound
	}
	return &profileTags[0], nil
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

	body, err := c.DoWithLock(req, profileTag.ProfileID)
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

	_, err = c.DoWithLock(req, profileID)

	return err
}
