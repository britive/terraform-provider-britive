package britive

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Get Resource Name By Id - Returns the name for a specific resource given the id
func (c *Client) GetResourceName(serverAccessResourceID string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/resources/%s", c.APIBaseURL, serverAccessResourceID), nil)
	if err != nil {
		return emptyString, err
	}

	body, err := c.Do(req)
	if err != nil {
		return emptyString, err
	}

	if string(body) == emptyString {
		return emptyString, ErrNotFound
	}

	resource := &ServerAccessResource{}
	err = json.Unmarshal(body, resource)
	if err != nil {
		return emptyString, err
	}

	if resource == nil {
		return emptyString, ErrNotFound
	}

	return resource.Name, nil
}

// Get Broker Pools Resource By Name - Returns the broker pools for a specific resource by name
func (c *Client) GetBrokerPoolsResourceByName(serverAccessResourceName string) (*[]BrokerPool, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/resources/%s/broker-pools", c.APIBaseURL, serverAccessResourceName), nil)
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

	brokerPoolsResource := &[]BrokerPool{}
	err = json.Unmarshal(body, brokerPoolsResource)
	if err != nil {
		return nil, err
	}

	if brokerPoolsResource == nil {
		return nil, ErrNotFound
	}

	return brokerPoolsResource, nil
}

// Get Broker Pools Resource - Returns the broker pools for a specific resource
func (c *Client) GetBrokerPoolsResource(serverAccessResourceName string) (*[]BrokerPool, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/resources/%s/broker-pools", c.APIBaseURL, serverAccessResourceName), nil)
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

	brokerPoolsResource := &[]BrokerPool{}
	err = json.Unmarshal(body, brokerPoolsResource)
	if err != nil {
		return nil, err
	}

	if brokerPoolsResource == nil {
		return nil, ErrNotFound
	}

	return brokerPoolsResource, nil
}

// AddBrokerPoolsResource - Add broker pools to a given resource
func (c *Client) AddBrokerPoolsResource(brokerPoolNamesString []string, serverAccessResourceName string) error {
	bp, err := json.Marshal(brokerPoolNamesString)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/resource-manager/resources/%s/broker-pools", c.APIBaseURL, serverAccessResourceName), strings.NewReader(string(bp)))
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, serverAccessLockName)
	if err != nil {
		return err
	}

	return nil
}

// DeleteBrokerPoolsResource - Delete broker pools resource
func (c *Client) DeleteBrokerPoolsResource(serverAccessResourceID string) error {

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/resource-manager/resources/%s/broker-pools", c.APIBaseURL, serverAccessResourceID), bytes.NewBuffer([]byte("[]")))
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, serverAccessLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
