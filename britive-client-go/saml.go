package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// GetSAMLSettings - Returns the tenant's SAML identity provider configurations
func (c *Client) GetSAMLSettings() ([]SAMLSettings, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/saml/settings", c.APIBaseURL), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	settings := make([]SAMLSettings, 0)
	if string(body) == emptyString {
		return settings, nil
	}
	if err := json.Unmarshal(body, &settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// GetSAMLMetadata - Returns the SAML identity provider metadata XML for a configuration, unchanged
func (c *Client) GetSAMLMetadata(configurationID int64) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/saml/metadata/%d", c.APIBaseURL, configurationID), nil)
	if err != nil {
		return emptyString, err
	}

	body, err := c.Do(req)
	if err != nil {
		return emptyString, err
	}

	return string(body), nil
}

// GetSAMLCertificate - Returns the PEM signing certificate of a SAML configuration, unchanged
func (c *Client) GetSAMLCertificate(configurationID int64) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/saml/certificate/%d", c.APIBaseURL, configurationID), nil)
	if err != nil {
		return emptyString, err
	}

	body, err := c.Do(req)
	if err != nil {
		return emptyString, err
	}

	return string(body), nil
}

// GetOAuth2Certificate - Returns the tenant's PEM OAuth2 certificate, unchanged
func (c *Client) GetOAuth2Certificate() (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/sso/certificate", c.APIBaseURL), nil)
	if err != nil {
		return emptyString, err
	}

	body, err := c.Do(req)
	if err != nil {
		return emptyString, err
	}

	return string(body), nil
}
