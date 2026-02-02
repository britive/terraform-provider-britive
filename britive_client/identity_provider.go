package britive_client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// GetIdentityProviders - Returns all identity providers
func (c *Client) GetIdentityProviders(ctx context.Context) (*[]IdentityProvider, error) {
	url := fmt.Sprintf("%s/identity-providers", c.APIBaseURL)
	body, err := c.Get(ctx, url, IdentityProviderLockName)
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
func (c *Client) GetIdentityProvider(ctx context.Context, identityProviderID string) (*IdentityProvider, error) {
	resourceURL := fmt.Sprintf("%s/identity-providers/%s", c.APIBaseURL, identityProviderID)
	return c.getIdentityProvider(ctx, resourceURL)
}

// GetIdentityProviderByName - Returns identity provider by name
func (c *Client) GetIdentityProviderByName(ctx context.Context, name string) (*IdentityProvider, error) {
	resourceURL := fmt.Sprintf("%s/identity-providers?metadata=false&name=%s", c.APIBaseURL, url.QueryEscape(name))
	return c.getIdentityProvider(ctx, resourceURL)
}

func (c *Client) getIdentityProvider(ctx context.Context, resourceURL string) (*IdentityProvider, error) {
	body, err := c.Get(ctx, resourceURL, IdentityProviderLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
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
