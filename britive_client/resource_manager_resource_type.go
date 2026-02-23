package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
)

// GetResourceTypeByName - Returns a specific resource type by name
func (c *Client) GetResourceTypeByName(ctx context.Context, name string) (*ResourceType, error) {
	resourceURL := fmt.Sprintf(`%s/resource-manager/resource-types/%s?compactResponse=true`, c.APIBaseURL, name)
	body, err := c.Get(ctx, resourceURL, ResourceManagerResourceTypeLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
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
func (c *Client) GetResourceType(ctx context.Context, resourceTypeID string) (*ResourceType, error) {
	url := fmt.Sprintf(`%s/resource-manager/resource-types/%s?compactResponse=true`, c.APIBaseURL, resourceTypeID)
	body, err := c.Get(ctx, url, ResourceManagerResourceTypeLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	resourceType := &ResourceType{}
	err = json.Unmarshal(body, resourceType)
	if err != nil {
		return nil, err
	}

	return resourceType, nil
}

// CreateResourceType - Create new resource type
func (c *Client) CreateResourceType(ctx context.Context, resourceType ResourceType) (*ResourceType, error) {
	url := fmt.Sprintf("%s/resource-manager/resource-types", c.APIBaseURL)
	body, err := c.Post(ctx, url, resourceType, ResourceManagerResourceTypeLockName)
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
func (c *Client) UpdateResourceType(ctx context.Context, resourceType ResourceType, resourceTypeID string) (*ResourceType, error) {
	url := fmt.Sprintf("%s/resource-manager/resource-types/%s", c.APIBaseURL, resourceTypeID)
	_, err := c.Put(ctx, url, resourceType, ResourceManagerResourceTypeLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &resourceType, nil
	}
	return nil, err
}

// DeleteResourceType - Delete resource type
func (c *Client) DeleteResourceType(ctx context.Context, resourceTypeID string) error {
	url := fmt.Sprintf("%s/resource-manager/resource-types/%s", c.APIBaseURL, resourceTypeID)
	err := c.Delete(ctx, url, ResourceManagerResourceTypeLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}

// Updating Icon - for resource type
func (c *Client) AddRemoveIcon(ctx context.Context, resourceTypeID string, uploadFilePath string) error {
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
