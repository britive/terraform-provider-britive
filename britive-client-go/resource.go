package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Get Server Access Resource By Name - Returns a specific server access resource by name
func (c *Client) GetServerAccessResourceByName(name string) (*ServerAccessResource, error) {
	resourceURL := fmt.Sprintf(`%s/resource-manager/resources/%s`, c.APIBaseURL, name)
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

	serverAccessResource := &ServerAccessResource{}
	err = json.Unmarshal(body, serverAccessResource)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Server access resource returned in GetServerAccessResourceByName: %#v", serverAccessResource)

	return serverAccessResource, nil
}

// Get Server Access Resource - Returns a specific server access resource by id
func (c *Client) GetServerAccessResource(serverAccessResourceID string) (*ServerAccessResource, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/resources/%s", c.APIBaseURL, serverAccessResourceID), nil)
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

	serverAccessResource := &ServerAccessResource{}
	err = json.Unmarshal(body, serverAccessResource)
	if err != nil {
		return nil, err
	}

	if serverAccessResource == nil {
		return nil, ErrNotFound
	}

	return serverAccessResource, nil
}

// AddServerAccessResource - Add new server access resource
func (c *Client) AddServerAccessResource(serverAccessResource ServerAccessResource) (*ServerAccessResource, error) {
	pb, err := json.Marshal(serverAccessResource)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/resource-manager/resources", c.APIBaseURL), strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, serverAccessLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &serverAccessResource)
	if err != nil {
		return nil, err
	}

	return &serverAccessResource, nil
}

// UpdateServerAccessResource - Update Server Access Resource
func (c *Client) UpdateServerAccessResource(serverAccessResource ServerAccessResource, serverAccessResourceID string) (*ServerAccessResource, error) {
	var serverAccessResourceBody []byte
	var err error
	serverAccessResourceBody, err = json.Marshal(serverAccessResource)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Update request resource body passed: %s", serverAccessResourceBody)
	log.Printf("[INFO] Update request resource id passed: %s", serverAccessResourceID)

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/resource-manager/resources/%s", c.APIBaseURL, serverAccessResourceID), strings.NewReader(string(serverAccessResourceBody)))
	if err != nil {
		return nil, err
	}

	_, err = c.DoWithLock(req, serverAccessLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &serverAccessResource, nil
	}
	return nil, err
}

// DeleteServerAccessResource - Delete server access resource
func (c *Client) DeleteServerAccessResource(serverAccessResourceID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/resource-manager/resources/%s", c.APIBaseURL, serverAccessResourceID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, serverAccessLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
