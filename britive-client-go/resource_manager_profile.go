package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) CreateUpdateResourceManagerProfile(resourceManagerProfile ResourceManagerProfile, isUpdate bool) (*ResourceManagerProfile, error) {
	pb, err := json.Marshal(resourceManagerProfile)
	if err != nil {
		return nil, err
	}

	var apiMethod, url string
	if isUpdate {
		apiMethod = "PATCH"
		url = fmt.Sprintf("%s/resource-manager/profiles/%s", c.APIBaseURL, resourceManagerProfile.ProfileId)
	} else {
		apiMethod = "POST"
		url = fmt.Sprintf("%s/resource-manager/profiles", c.APIBaseURL)
	}

	req, err := http.NewRequest(apiMethod, url, strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(req, resourceManagerProfileLock)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return nil, err
	}

	if len(resp) != 0 {
		err = json.Unmarshal(resp, &resourceManagerProfile)
		if err != nil {
			return nil, err
		}
	}

	return &resourceManagerProfile, nil
}

func (c *Client) CreateUpdateResourceManagerProfileAssociations(resourceManagerProfile ResourceManagerProfile) (*ResourceManagerProfile, error) {
	pb, err := json.Marshal(resourceManagerProfile)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/resource-manager/profiles/%s/associations", c.APIBaseURL, resourceManagerProfile.ProfileId)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(req, resourceManagerProfileLock)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return nil, err
	}

	if len(resp) != 0 {
		err = json.Unmarshal(resp, &resourceManagerProfile)
		if err != nil {
			return nil, err
		}
	}

	return &resourceManagerProfile, nil
}

func (c *Client) GetResourceManagerProfile(profileId string) (*ResourceManagerProfile, error) {
	apiMethod := "GET"
	url := fmt.Sprintf("%s/resource-manager/profiles/%s", c.APIBaseURL, profileId)
	req, err := http.NewRequest(apiMethod, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(req, resourceManagerProfileLock)
	if err != nil {
		return nil, err
	}

	var resourceManagerProfile ResourceManagerProfile
	err = json.Unmarshal(resp, &resourceManagerProfile)
	if err != nil {
		return nil, err
	}

	return &resourceManagerProfile, nil
}

func (c *Client) GetResourceManagerProfileAssociations(profileId string) (*ResourceManagerProfile, error) {
	apiMethod := "GET"
	url := fmt.Sprintf("%s/resource-manager/profiles/%s/associations", c.APIBaseURL, profileId)
	req, err := http.NewRequest(apiMethod, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(req, resourceManagerProfileLock)
	if err != nil {
		return nil, err
	}

	var resourceManagerProfile ResourceManagerProfile
	err = json.Unmarshal(resp, &resourceManagerProfile)
	if err != nil {
		return nil, err
	}

	return &resourceManagerProfile, nil
}

func (c *Client) DeleteResourceManagerProfile(profileId string) error {
	url := fmt.Sprintf("%s/resource-manager/profiles/%s", c.APIBaseURL, profileId)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, resourceManagerProfileLock)
	if err != nil && !errors.Is(err, ErrNoContent) {
		return err
	}

	return nil
}
