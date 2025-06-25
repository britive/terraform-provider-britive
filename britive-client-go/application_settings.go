package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// CreateApplication - Create new application
func (c *Client) CreateApplicationSettings(applicationSettings AdvancedSettings) (*AdvancedSettings, error) {
	appID := applicationSettings.Settings[0].EntityID
	applicationSettingURL := fmt.Sprintf("%s/apps/%s/advanced-settings", c.APIBaseURL, appID)
	pb, err := json.Marshal(applicationSettings)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", applicationSettingURL, strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, applicationLockName)
	if err != nil {
		return nil, err
	}

	applicationSettingsResponse := AdvancedSettings{}
	err = json.Unmarshal(body, &applicationSettingsResponse)
	if err != nil {
		return nil, err
	}
	return &applicationSettingsResponse, nil
}
