package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// GetAttributeByName - Returns a specifc user attribute by name
func (c *Client) GetAttributeByName(ctx context.Context, name string) (*UserAttribute, error) {
	filter := fmt.Sprintf(`name eq "%s"`, name)
	resourceURL := fmt.Sprintf(`%s/users/attributes?filter=%s`, c.APIBaseURL, url.QueryEscape(filter))
	body, err := c.Get(ctx, resourceURL, ProfileLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	attributes := make([]UserAttribute, 0)
	err = json.Unmarshal(body, &attributes)
	if err != nil {
		return nil, err
	}

	if len(attributes) == 0 {
		return nil, ErrNotFound
	}

	return &attributes[0], nil
}

// GetAttribute - Returns a specifc attribute by id
func (c *Client) GetAttribute(ctx context.Context, attributeID string) (*UserAttribute, error) {
	url := fmt.Sprintf("%s/users/attributes/%s", c.APIBaseURL, attributeID)
	body, err := c.Get(ctx, url, ProfileLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	attribute := &UserAttribute{}
	err = json.Unmarshal(body, attribute)
	if err != nil {
		return nil, err
	}

	return attribute, nil
}

// GetProfileSessionAttributes - Returns profile session attributes
func (c *Client) GetProfileSessionAttribute(ctx context.Context, profileID string, sessionAttributeID string) (*SessionAttribute, error) {
	sessionAttributes, err := c.GetProfileSessionAttributes(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if sessionAttributes == nil || len(*sessionAttributes) == 0 {
		return nil, ErrNotFound
	}
	var sessionAttribute SessionAttribute
	for _, sa := range *sessionAttributes {
		if sa.ID == sessionAttributeID {
			sessionAttribute = sa
			break
		}
	}
	return &sessionAttribute, nil
}

// GetProfileSessionAttributeByTypeAndMappingName - Returns profile session attributes
func (c *Client) GetProfileSessionAttributeByTypeAndMappingName(ctx context.Context, profileID, attributeType, mappingName string) (*SessionAttribute, error) {
	sessionAttributes, err := c.GetProfileSessionAttributes(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if sessionAttributes == nil || len(*sessionAttributes) == 0 {
		return nil, ErrNotFound
	}
	var sessionAttribute SessionAttribute
	for _, sa := range *sessionAttributes {
		if strings.EqualFold(sa.SessionAttributeType, attributeType) && strings.EqualFold(sa.MappingName, mappingName) {
			sessionAttribute = sa
			break
		}
	}
	return &sessionAttribute, nil
}

// GetProfileSessionAttributes - Returns profile session attributes
func (c *Client) GetProfileSessionAttributes(ctx context.Context, profileID string) (*[]SessionAttribute, error) {
	url := fmt.Sprintf("%s/paps/%s/session-attributes", c.APIBaseURL, profileID)
	body, err := c.Get(ctx, url, ProfileLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	attributes := &[]SessionAttribute{}
	err = json.Unmarshal(body, attributes)
	if err != nil {
		return nil, err
	}

	if len(*attributes) == 0 {
		return nil, ErrNotFound
	}

	return attributes, nil
}

// CreateProfileSessionAttribute - Create new profile session attribute
func (c *Client) CreateProfileSessionAttribute(ctx context.Context, profileID string, sessionAttribute SessionAttribute) (*SessionAttribute, error) {
	url := fmt.Sprintf("%s/paps/%s/session-attributes", c.APIBaseURL, profileID)
	body, err := c.Post(ctx, url, sessionAttribute, ProfileLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &sessionAttribute)
	if err != nil {
		return nil, err
	}
	return &sessionAttribute, nil
}

// UpdateProfileSessionAttribute - Update profile session attribute
func (c *Client) UpdateProfileSessionAttribute(ctx context.Context, profileID string, sessionAttribute SessionAttribute) (*SessionAttribute, error) {
	url := fmt.Sprintf("%s/paps/%s/session-attributes", c.APIBaseURL, profileID)
	_, err := c.Put(ctx, url, sessionAttribute, ProfileLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &sessionAttribute, nil
	}
	return nil, err
}

// DeleteTag - Delete tag
func (c *Client) DeleteProfileSessionAttribute(ctx context.Context, profileID string, sessionAttributeID string) error {
	url := fmt.Sprintf("%s/paps/%s/session-attributes/%s", c.APIBaseURL, profileID, sessionAttributeID)
	err := c.Delete(ctx, url, ProfileLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
