package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

func (c *Client) CreateUpdateResourceLabel(ctx context.Context, resourceLabel ResourceLabel, isUpdate bool) (*ResourceLabel, error) {
	var body []byte
	var err error
	if isUpdate {
		url := fmt.Sprintf("%s/resource-manager/labels/%s", c.APIBaseURL, resourceLabel.LabelId)
		body, err = c.Put(ctx, url, resourceLabel, ResourceManagerResourceLabelLockName)
	} else {
		url := fmt.Sprintf("%s/resource-manager/labels", c.APIBaseURL)
		body, err = c.Post(ctx, url, resourceLabel, ResourceManagerResourceLabelLockName)
	}

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &resourceLabel)
	if err != nil {
		return nil, err
	}
	return &resourceLabel, nil

}

func (c *Client) GetResourceLabel(ctx context.Context, labelId string) (*ResourceLabel, error) {
	url := fmt.Sprintf("%s/resource-manager/labels/%s", c.APIBaseURL, labelId)
	body, err := c.Get(ctx, url, ResourceManagerResourceLabelLockName)
	if err != nil {
		return nil, err
	}

	var resourceLabel ResourceLabel
	err = json.Unmarshal(body, &resourceLabel)
	if err != nil {
		return nil, err
	}
	return &resourceLabel, nil
}

func (c *Client) DeleteResourceLabel(ctx context.Context, labelId string) error {
	url := fmt.Sprintf("%s/resource-manager/labels/%s", c.APIBaseURL, labelId)
	err := c.Delete(ctx, url, ResourceManagerResourceLabelLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
