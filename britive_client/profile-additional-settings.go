package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// UpdateProfileAdditionalSettings - Update profile additional settings
func (c *Client) UpdateProfileAdditionalSettings(ctx context.Context, profileAdditionalSettings ProfileAdditionalSettings) (*ProfileAdditionalSettings, error) {
	url := fmt.Sprintf("%s/paps/%s/additional-settings", c.APIBaseURL, profileAdditionalSettings.ProfileID)

	_, err := c.Patch(ctx, url, profileAdditionalSettings, ProfileLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &profileAdditionalSettings, nil
	}
	return nil, err
}

// GetProfileAdditionalSettings - Returns the additional settings from profile
func (c *Client) GetProfileAdditionalSettings(ctx context.Context, profileID string) (*ProfileAdditionalSettings, error) {

	requestURL := fmt.Sprintf("%s/paps/%s/additional-settings?propertiesOnly=true", c.APIBaseURL, profileID)

	body, err := c.Get(ctx, requestURL, ProfileLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	profileAdditionalSettings := &ProfileAdditionalSettings{}

	err = json.Unmarshal(body, profileAdditionalSettings)
	if err != nil {
		return nil, err
	}

	return profileAdditionalSettings, nil
}
