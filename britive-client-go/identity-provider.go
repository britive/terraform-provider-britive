package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// GetIdentityProviders - Returns all identity providers
func (c *Client) GetIdentityProviders() (*[]IdentityProvider, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/identity-providers", c.APIBaseURL), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	identityProviders := make([]IdentityProvider, 0)
	err = json.Unmarshal(body, &identityProviders)
	if err != nil {
		return nil, err
	}

	return &identityProviders, nil
}

// GetIdentityProvider - Returns identity provider
func (c *Client) GetIdentityProvider(identityProviderID string) (*IdentityProvider, error) {
	resourceURL := fmt.Sprintf("%s/identity-providers/%s", c.APIBaseURL, identityProviderID)
	return c.getIdentityProvider(resourceURL)
}

// GetIdentityProviderByName - Returns identity provider by name
func (c *Client) GetIdentityProviderByName(name string) (*IdentityProvider, error) {
	resourceURL := fmt.Sprintf("%s/identity-providers?metadata=false&name=%s", c.APIBaseURL, url.QueryEscape(name))
	return c.getIdentityProvider(resourceURL)
}

func (c *Client) getIdentityProvider(resourceURL string) (*IdentityProvider, error) {
	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	if string(body) == emptyString {
		return nil, ErrNotFound
	}

	identityProvider := &IdentityProvider{}
	err = json.Unmarshal(body, identityProvider)
	if err != nil {
		return nil, err
	}

	if identityProvider == nil {
		return nil, ErrNotFound
	}

	return identityProvider, nil
}
