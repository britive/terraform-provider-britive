package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
)

// GetResourceTypeByName - Returns a specific resource type by name
func (c *Client) GetResourceTypeByName(name string) (*ResourceType, error) {
	resourceURL := fmt.Sprintf(`%s/resource-manager/resource-types/%s?compactResponse=true`, c.APIBaseURL, name)
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

	resourceType := &ResourceType{}
	err = json.Unmarshal(body, resourceType)
	if err != nil {
		return nil, err
	}

	return resourceType, nil
}

// GetResourceType - Returns a specific resource type by id
func (c *Client) GetResourceType(resourceTypeID string) (*ResourceType, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(`%s/resource-manager/resource-types/%s?compactResponse=true`, c.APIBaseURL, resourceTypeID), nil)

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

	resourceType := &ResourceType{}
	err = json.Unmarshal(body, resourceType)
	if err != nil {
		return nil, err
	}

	if resourceType == nil {
		return nil, ErrNotFound
	}

	return resourceType, nil
}

// CreateResourceType - Create new resource type
func (c *Client) CreateResourceType(resourceType ResourceType) (*ResourceType, error) {
	pb, err := json.Marshal(resourceType)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/resource-manager/resource-types", c.APIBaseURL), strings.NewReader(string(pb)))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, resourceTypeLockName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &resourceType)
	if err != nil {
		return nil, err
	}
	return &resourceType, nil
}

// UpdateResourceType - Update resource type
func (c *Client) UpdateResourceType(resourceType ResourceType, resourceTypeID string) (*ResourceType, error) {
	var resourceTypeBody []byte
	var err error
	resourceTypeBody, err = json.Marshal(resourceType)
	if err != nil {
		return nil, err
	}

	// ToDo: Check for patch/put
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/resource-manager/resource-types/%s", c.APIBaseURL, resourceTypeID), strings.NewReader(string(resourceTypeBody)))
	if err != nil {
		return nil, err
	}

	_, err = c.DoWithLock(req, resourceTypeLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &resourceType, nil
	}
	return nil, err
}

// DeleteResourceType - Delete resource type
func (c *Client) DeleteResourceType(resourceTypeID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/resource-manager/resource-types/%s", c.APIBaseURL, resourceTypeID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, resourceTypeLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}

// Updating Icon - for resource type
func (c *Client) AddRemoveIcon(resourceTypeID string, uploadFilePath string) error {
	presignedURL := fmt.Sprintf("%s/resource-manager/resource-types/%s/icon-data", c.APIBaseURL, resourceTypeID)

	var err error
	var req *http.Request
	if len(uploadFilePath) != 0 {
		req, err = http.NewRequest("PUT", presignedURL, strings.NewReader(uploadFilePath))
	} else {
		req, err = http.NewRequest("DELETE", presignedURL, nil)
	}
	if err != nil {
		fmt.Println("Failed to create request:", err)
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("TOKEN %s", c.Token))
	req.Header.Set("Content-Type", "text/xml")
	userAgent := fmt.Sprintf("britive-client-go/%s golang/%s %s/%s britive-terraform/%s", c.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH, c.Version)
	req.Header.Add("User-Agent", userAgent)

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil && !errors.Is(err, ErrNoContent) {
		fmt.Println("Upload failed:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		log.Println("Upload successful!")
	} else {
		fmt.Printf("Upload failed. Status: %s\n", resp.Status)
		return fmt.Errorf("Upload failed. Status: %s\n", resp.Status)
	}
	return nil
}
