package britive

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// CreateResponseTemplate creates a new response template.
func (c *Client) CreateResponseTemplate(template ResponseTemplate) (*ResponseTemplate, error) {
	body, err := json.Marshal(template)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/resource-manager/response-templates", c.APIBaseURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, responseTemplateLockName)
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
func (c *Client) GetResponseTemplate(templateID string) (*ResponseTemplate, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/response-templates/%s", c.APIBaseURL, templateID), nil)
	if err != nil {
		return nil, err
	}

	respBody, err := c.Do(req)
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
func (c *Client) UpdateResponseTemplate(templateID string, template ResponseTemplate) (*ResponseTemplate, error) {
	body, err := json.Marshal(template)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/resource-manager/response-templates/%s", c.APIBaseURL, templateID), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, responseTemplateLockName)
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
func (c *Client) DeleteResponseTemplate(templateID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/resource-manager/response-templates/%s", c.APIBaseURL, templateID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, responseTemplateLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
