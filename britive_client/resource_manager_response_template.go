package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// CreateResponseTemplate creates a new response template.
func (c *Client) CreateResponseTemplate(ctx context.Context, template ResponseTemplate) (*ResponseTemplate, error) {
	url := fmt.Sprintf("%s/resource-manager/response-templates", c.APIBaseURL)
	respBody, err := c.Post(ctx, url, template, ResourceManagerResponseTemplateLockName)
	if err != nil {
		return nil, err
	}

	var result ResponseTemplate
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetResponseTemplate retrieves a response template by its ID.
func (c *Client) GetResponseTemplate(ctx context.Context, templateID string) (*ResponseTemplate, error) {
	url := fmt.Sprintf("%s/resource-manager/response-templates/%s", c.APIBaseURL, templateID)
	respBody, err := c.Get(ctx, url, ResourceManagerResponseTemplateLockName)
	if err != nil {
		return nil, err
	}

	var result ResponseTemplate
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateResponseTemplate updates an existing response template.
func (c *Client) UpdateResponseTemplate(ctx context.Context, templateID string, template ResponseTemplate) (*ResponseTemplate, error) {
	url := fmt.Sprintf("%s/resource-manager/response-templates/%s", c.APIBaseURL, templateID)
	respBody, err := c.Put(ctx, url, template, ResourceManagerResponseTemplateLockName)
	if err != nil {
		return nil, err
	}

	var result ResponseTemplate
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteResponseTemplate deletes a response template by its ID.
func (c *Client) DeleteResponseTemplate(ctx context.Context, templateID string) error {
	url := fmt.Sprintf("%s/resource-manager/response-templates/%s", c.APIBaseURL, templateID)
	err := c.Delete(ctx, url, ResourceManagerResponseTemplateLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}

// GetResponseTemplate retrieves a response template by its ID.
func (c *Client) GetAllResponseTemplate(ctx context.Context) ([]ResponseTemplate, error) {
	url := fmt.Sprintf("%s/resource-manager/response-templates", c.APIBaseURL)
	respBody, err := c.Get(ctx, url, ResourceManagerResponseTemplateLockName)
	if err != nil {
		return nil, err
	}

	var allResponseTemplates AllResponseTemplates
	err = json.Unmarshal(respBody, &allResponseTemplates)
	if err != nil {
		return nil, err
	}

	return allResponseTemplates.ResponseTemplates, nil
}
