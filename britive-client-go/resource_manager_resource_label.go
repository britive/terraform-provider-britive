package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) CreateUpdateResourceLabel(resourceLabel ResourceLabel, isUpdate bool) (*ResourceLabel, error) {
	pb, err := json.Marshal(resourceLabel)
	if err != nil {
		return nil, err
	}

	var apiMethod, url string
	if isUpdate {
		apiMethod = "PUT"
		url = fmt.Sprintf("%s/resource-manager/labels/%s", c.APIBaseURL, resourceLabel.LabelId)
	} else {
		apiMethod = "POST"
		url = fmt.Sprintf("%s/resource-manager/labels", c.APIBaseURL)
	}

	req, err := http.NewRequest(apiMethod, url, strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, resourceLabelLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &resourceLabel)
	if err != nil {
		return nil, err
	}
	return &resourceLabel, nil

}

func (c *Client) GetResourceLabel(labelId string) (*ResourceLabel, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/labels/%s", c.APIBaseURL, labelId), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, resourceLabelLockName)
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

func (c *Client) DeleteResourceLabel(labelId string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/resource-manager/labels/%s", c.APIBaseURL, labelId), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, responseTemplateLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
