package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// GetProfileAdditionalSettings - Returns the additional settings from profile
func (c *Client) GetProfileAdditionalSettings(profileID string) (*ProfileAdditionalSettings, error) {

	requestURL := fmt.Sprintf("%s/paps/%s/additional-settings?propertiesOnly=true", c.APIBaseURL, profileID)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileID)
	if err != nil {
		return nil, err
	}

	if string(body) == emptyString {
		return nil, ErrNotFound
	}

	profileAdditionalSettings := &ProfileAdditionalSettings{}

	err = json.Unmarshal(body, profileAdditionalSettings)
	if err != nil {
		return nil, err
	}

	return profileAdditionalSettings, nil
}

// UpdateProfileAdditionalSettings - Update profile additional settings
func (c *Client) UpdateProfileAdditionalSettings(profileAdditionalSettings ProfileAdditionalSettings) (*ProfileAdditionalSettings, error) {

	var profileAdditionalSettingsBody []byte
	var err error
	profileAdditionalSettingsBody, err = json.Marshal(profileAdditionalSettings)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/paps/%s/additional-settings", c.APIBaseURL, profileAdditionalSettings.ProfileID), strings.NewReader(string(profileAdditionalSettingsBody)))
	if err != nil {
		return nil, err
	}

	_, err = c.DoWithLock(req, profileAdditionalSettings.ProfileID)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &profileAdditionalSettings, nil
	}
	return nil, err
}
