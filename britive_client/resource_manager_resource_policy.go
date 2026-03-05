package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

func (c *Client) CreateUpdateResourceManagerResourcePolicy(ctx context.Context, resourcePolicy ResourceManagerResourcePolicy, oldName string, isUpdate bool) (*ResourceManagerResourcePolicy, error) {
	var resp []byte
	var err error
	if isUpdate {
		url := fmt.Sprintf("%s/resource-manager/policies/%s", c.APIBaseURL, oldName)
		resp, err = c.Patch(ctx, url, resourcePolicy, ResourceManagerResourcePlicyLockName)
	} else {
		url := fmt.Sprintf("%s/resource-manager/policies", c.APIBaseURL)
		resp, err = c.Post(ctx, url, resourcePolicy, ResourceManagerResourcePlicyLockName)
	}

	if err != nil && !errors.Is(err, ErrNoContent) {
		return nil, err
	}

	if len(resp) != 0 {
		err = json.Unmarshal(resp, &resourcePolicy)
		if err != nil {
			return nil, err
		}
	}

	return &resourcePolicy, nil
}

func (c *Client) GetResourceManagerResourcePolicy(ctx context.Context, policyName string) (*ResourceManagerResourcePolicy, error) {
	url := fmt.Sprintf("%s/resource-manager/policies/%s?compactResponse=true", c.APIBaseURL, policyName)

	resp, err := c.Get(ctx, url, ResourceManagerResourcePlicyLockName)
	if err != nil {
		return nil, err
	}

	var resourcePolicy ResourceManagerResourcePolicy
	err = json.Unmarshal(resp, &resourcePolicy)
	if err != nil {
		return nil, err
	}

	return &resourcePolicy, nil
}

func (c *Client) DeleteResourceManagerResourcePolicy(ctx context.Context, policyID string) error {
	url := fmt.Sprintf("%s/resource-manager/policies/%s", c.APIBaseURL, policyID)
	err := c.Delete(ctx, url, ResourceManagerResourcePlicyLockName)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return err
	}

	return nil
}
