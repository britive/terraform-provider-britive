package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GetAttributeByName - Returns a specifc user attribute by name
func (c *Client) GetAttributeByName(name string) (*UserAttribute, error) {
	filter := fmt.Sprintf(`name eq "%s"`, name)
	resourceURL := fmt.Sprintf(`%s/users/attributes?filter=%s`, c.APIBaseURL, url.QueryEscape(filter))
	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if string(body) == emptyString {
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
func (c *Client) GetAttribute(attributeID string) (*UserAttribute, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/users/attributes/%s", c.APIBaseURL, attributeID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if string(body) == emptyString {
		return nil, ErrNotFound
	}

	attribute := &UserAttribute{}
	err = json.Unmarshal(body, attribute)
	if err != nil {
		return nil, err
	}

	if attribute == nil {
		return nil, ErrNotFound
	}

	return attribute, nil
}

// GetProfileSessionAttributes - Returns profile session attributes
func (c *Client) GetProfileSessionAttribute(profileID string, sessionAttributeID string) (*SessionAttribute, error) {
	sessionAttributes, err := c.GetProfileSessionAttributes(profileID)
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
func (c *Client) GetProfileSessionAttributeByTypeAndMappingName(profileID, attributeType, mappingName string) (*SessionAttribute, error) {
	sessionAttributes, err := c.GetProfileSessionAttributes(profileID)
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
func (c *Client) GetProfileSessionAttributes(profileID string) (*[]SessionAttribute, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/paps/%s/session-attributes", c.APIBaseURL, profileID), nil)
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

	attributes := &[]SessionAttribute{}
	err = json.Unmarshal(body, attributes)
	if err != nil {
		return nil, err
	}

	if attributes == nil || len(*attributes) == 0 {
		return nil, ErrNotFound
	}

	return attributes, nil
}

// CreateProfileSessionAttribute - Create new profile session attribute
func (c *Client) CreateProfileSessionAttribute(profileID string, sessionAttribute SessionAttribute) (*SessionAttribute, error) {
	sessionAttributeBytes, err := json.Marshal(sessionAttribute)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/paps/%s/session-attributes", c.APIBaseURL, profileID), strings.NewReader(string(sessionAttributeBytes)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, profileID)
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
func (c *Client) UpdateProfileSessionAttribute(profileID string, sessionAttribute SessionAttribute) (*SessionAttribute, error) {
	sessionAttributeBytes, err := json.Marshal(sessionAttribute)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/paps/%s/session-attributes", c.APIBaseURL, profileID), strings.NewReader(string(sessionAttributeBytes)))
	if err != nil {
		return nil, err
	}

	_, err = c.DoWithLock(req, profileID)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &sessionAttribute, nil
	}
	return nil, err
}

// DeleteTag - Delete tag
func (c *Client) DeleteProfileSessionAttribute(profileID string, sessionAttributeID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/paps/%s/session-attributes/%s", c.APIBaseURL, profileID, sessionAttributeID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, profileID)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
