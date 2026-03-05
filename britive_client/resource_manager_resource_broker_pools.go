package britive_client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Get Resource Name By Id - Returns the name for a specific resource given the id
func (c *Client) GetResourceName(ctx context.Context, serverAccessResourceID string) (string, error) {
	url := fmt.Sprintf("%s/resource-manager/resources/%s", c.APIBaseURL, serverAccessResourceID)

	body, err := c.Get(ctx, url, ResourceManagerResourceLockName)
	if err != nil {
		return EmptyString, err
	}

	if string(body) == EmptyString {
		return EmptyString, ErrNotFound
	}

	resource := &ServerAccessResource{}
	err = json.Unmarshal(body, resource)
	if err != nil {
		return EmptyString, err
	}

	return resource.Name, nil
}

// Get Broker Pools Resource By Name - Returns the broker pools for a specific resource by name
func (c *Client) GetBrokerPoolsResourceByName(ctx context.Context, serverAccessResourceName string) (*[]BrokerPool, error) {
	url := fmt.Sprintf("%s/resource-manager/resources/%s/broker-pools", c.APIBaseURL, serverAccessResourceName)

	body, err := c.Get(ctx, url, ResourceManagerResourceLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	brokerPoolsResource := &[]BrokerPool{}
	err = json.Unmarshal(body, brokerPoolsResource)
	if err != nil {
		return nil, err
	}

	return brokerPoolsResource, nil
}

// Get Broker Pools Resource - Returns the broker pools for a specific resource
func (c *Client) GetBrokerPoolsResource(ctx context.Context, serverAccessResourceName string) (*[]BrokerPool, error) {
	url := fmt.Sprintf("%s/resource-manager/resources/%s/broker-pools", c.APIBaseURL, serverAccessResourceName)

	body, err := c.Get(ctx, url, ResourceManagerResourceLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	brokerPoolsResource := &[]BrokerPool{}
	err = json.Unmarshal(body, brokerPoolsResource)
	if err != nil {
		return nil, err
	}

	return brokerPoolsResource, nil
}

// AddBrokerPoolsResource - Add broker pools to a given resource
func (c *Client) AddBrokerPoolsResource(ctx context.Context, brokerPoolNamesString []string, serverAccessResourceName string) error {
	url := fmt.Sprintf("%s/resource-manager/resources/%s/broker-pools", c.APIBaseURL, serverAccessResourceName)

	_, err := c.Post(ctx, url, brokerPoolNamesString, ResourceManagerResourceLockName)
	if err != nil {
		return err
	}

	return nil
}

// DeleteBrokerPoolsResource - Delete broker pools resource
func (c *Client) DeleteBrokerPoolsResource(ctx context.Context, serverAccessResourceID string) error {

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/resource-manager/resources/%s/broker-pools", c.APIBaseURL, serverAccessResourceID), bytes.NewBuffer([]byte("[]")))
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(ctx, req, ResourceManagerResourcePlicyLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
