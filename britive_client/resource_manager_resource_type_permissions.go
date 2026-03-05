package britive_client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

// GetResourceTypePermission - Returns a specific resource type permission by ID
func (c *Client) GetResourceTypePermission(ctx context.Context, permissionID string) (*ResourceTypePermission, error) {
	// Fetch the latest version's details
	url := fmt.Sprintf("%s/resource-manager/permissions/%s/latest", c.APIBaseURL, permissionID)
	body, err := c.Get(ctx, url, ResourceManagerResourceTypeLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	permission := &ResourceTypePermission{}
	err = json.Unmarshal(body, permission)
	if err != nil {
		return nil, err
	}

	return permission, nil
}

func (c *Client) GetPermissionUploadUrls(ctx context.Context, permissionID string) (*ResourceTypePermissiosUploadUrls, error) {
	// Step 1: Fetch the list of permissions
	url := fmt.Sprintf("%s/resource-manager/permissions/get-urls/%s", c.APIBaseURL, permissionID)
	body, err := c.Get(ctx, url, ResourceManagerResourceTypeLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
		return nil, ErrNotFound
	}

	permissionUrls := &ResourceTypePermissiosUploadUrls{}
	err = json.Unmarshal(body, &permissionUrls)
	if err != nil {
		return nil, err
	}

	return permissionUrls, nil
}

func (c *Client) UploadFile(presignedURL string, uploadFilePath string) error {
	filePath, err := filepath.Abs(uploadFilePath)
	if err != nil {
		return err
	}

	// Read the file
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("Failed to read file:", err)
		return err
	}

	// Create the PUT request
	req, err := http.NewRequest("PUT", presignedURL, bytes.NewReader(fileData))
	if err != nil {
		fmt.Println("Failed to create request:", err)
		return err
	}

	// Optionally set the content type if needed
	req.Header.Set("Content-Type", "text/plain") // or "application/octet-stream"

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Upload failed:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Println("Upload successful!")
	} else {
		fmt.Printf("Upload failed. Status: %s\n", resp.Status)
	}
	return nil
}

func (c *Client) UploadPermissionFiles(ctx context.Context, permissionId string, checkInFilePath string, checkOutFilePath string) error {
	permissionUploadUrls, err := c.GetPermissionUploadUrls(ctx, permissionId)
	if err != nil {
		return err
	}

	checkInUrl := permissionUploadUrls.CheckInUrl
	err = c.UploadFile(checkInUrl, checkInFilePath)
	if err != nil {
		return err
	}

	checkOutUrl := permissionUploadUrls.CheckOutUrl
	err = c.UploadFile(checkOutUrl, checkOutFilePath)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) UploadCode(presignedURL string, code string, contentType string) error {

	codePayload := []byte(code)

	// Create the PUT request
	req, err := http.NewRequest("PUT", presignedURL, bytes.NewBuffer(codePayload))
	if err != nil {
		fmt.Println("Failed to create request:", err)
		return err
	}

	req.Header.Set("Content-Type", contentType)

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Upload code failed:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Println("Upload successful!")
	} else {
		fmt.Printf("Upload failed. Status: %s\n", resp.Status)
	}
	return nil
}

func (c *Client) UploadPermissionCodes(ctx context.Context, permissionId string, checkInCode string, checkOutCode string, codeLanguage string) error {
	permissionCodeLanguageMap := map[string]string{
		"text":       "text/plain",
		"batch":      "text/x-batch",
		"node":       "application/octet-stream",
		"powershell": "application/x-powershell",
		"python":     "text/x-python",
		"shell":      "application/x-sh",
	}

	contentType, exists := permissionCodeLanguageMap[strings.ToLower(codeLanguage)]
	if !exists {
		return errors.New("Code Language of type " + codeLanguage + " is unsupported.")
	}

	permissionUploadUrls, err := c.GetPermissionUploadUrls(ctx, permissionId)
	if err != nil {
		return err
	}

	checkInUrl := permissionUploadUrls.CheckInUrl
	err = c.UploadCode(checkInUrl, checkInCode, contentType)
	if err != nil {
		return err
	}

	checkOutUrl := permissionUploadUrls.CheckOutUrl
	err = c.UploadCode(checkOutUrl, checkOutCode, contentType)
	if err != nil {
		return err
	}

	return nil
}

// CreateResourceTypePermission - Creates a new resource type permission
func (c *Client) CreateResourceTypePermission(ctx context.Context, permission ResourceTypePermission) (*ResourceTypePermission, error) {
	url := fmt.Sprintf("%s/resource-manager/permissions", c.APIBaseURL)
	respBody, err := c.Post(ctx, url, permission, ResourceManagerResourceTypeLockName)
	if err != nil {
		return nil, err
	}

	createdPermission := &ResourceTypePermission{}
	err = json.Unmarshal(respBody, createdPermission)
	if err != nil {
		return nil, err
	}

	return createdPermission, nil
}

// UpdateResourceTypePermission - Updates an existing resource type permission
func (c *Client) UpdateResourceTypePermission(ctx context.Context, permission ResourceTypePermission) (*ResourceTypePermission, error) {
	url := fmt.Sprintf("%s/resource-manager/permissions/%s", c.APIBaseURL, permission.PermissionID)
	respBody, err := c.Put(ctx, url, permission, ResourceManagerResourceTypeLockName)
	if err != nil {
		return nil, err
	}

	updatedPermission := &ResourceTypePermission{}
	err = json.Unmarshal(respBody, updatedPermission)
	if err != nil {
		return nil, err
	}

	return updatedPermission, nil
}

// DeleteResourceTypePermission - Deletes a resource type permission by ID
func (c *Client) DeleteResourceTypePermission(ctx context.Context, permissionID string) error {
	url := fmt.Sprintf("%s/resource-manager/permissions/%s", c.APIBaseURL, permissionID)
	err := c.Delete(ctx, url, ResourceManagerResourceTypeLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
