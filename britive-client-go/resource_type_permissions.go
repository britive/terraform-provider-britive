package britive

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
)

// GetResourceTypePermission - Returns a specific resource type permission by ID
func (c *Client) GetResourceTypePermission(permissionID string) (*ResourceTypePermission, error) {
	// Step 1: Fetch the list of permissions
	// req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/permissions/%s", c.APIBaseURL, permissionID), nil)
	// if err != nil {
	// 	return nil, err
	// }

	// body, err := c.Do(req)
	// if err != nil {
	// 	return nil, err
	// }

	// if string(body) == emptyString {
	// 	return nil, ErrNotFound
	// }

	// var permissions []ResourceTypePermission
	// err = json.Unmarshal(body, &permissions)
	// if err != nil {
	// 	return nil, err
	// }

	// if len(permissions) == 0 {
	// 	return nil, ErrNotFound
	// }

	// // Find the latest version explicitly
	// latestPermission := permissions[0]
	// for _, perm := range permissions {
	// 	if perm.Version > latestPermission.Version {
	// 		latestPermission = perm
	// 	}
	// }

	// Step 2: Fetch the latest version's details
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/permissions/%s/latest", c.APIBaseURL, permissionID), nil)
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

	permission := &ResourceTypePermission{}
	err = json.Unmarshal(body, permission)
	if err != nil {
		return nil, err
	}

	return permission, nil
}

func (c *Client) GetPermissionUploadUrls(permissionID string) (*ResourceTypePermissiosUploadUrls, error) {
	// Step 1: Fetch the list of permissions
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/permissions/get-urls/%s", c.APIBaseURL, permissionID), nil)
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
		fmt.Println("Upload successful!")
	} else {
		fmt.Printf("Upload failed. Status: %s\n", resp.Status)
	}
	return nil
}

func (c *Client) UploadPermissionFiles(permissionId string, checkInFilePath string, checkOutFilePath string) error {
	permissionUploadUrls, err := c.GetPermissionUploadUrls(permissionId)
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
		fmt.Println("Upload successful!")
	} else {
		fmt.Printf("Upload failed. Status: %s\n", resp.Status)
	}
	return nil
}

func (c *Client) UploadPermissionCodes(permissionId string, checkInCode string, checkOutCode string, codeLanguage string) error {
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

	permissionUploadUrls, err := c.GetPermissionUploadUrls(permissionId)
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
func (c *Client) CreateResourceTypePermission(permission ResourceTypePermission) (*ResourceTypePermission, error) {
	body, err := json.Marshal(permission)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/resource-manager/permissions", c.APIBaseURL), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, resourceTypePermissions) // Updated to include lock name
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
func (c *Client) UpdateResourceTypePermission(permission ResourceTypePermission) (*ResourceTypePermission, error) {
	body, err := json.Marshal(permission)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/resource-manager/permissions/%s", c.APIBaseURL, permission.PermissionID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, resourceTypePermissions) // Updated to include lock name
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
func (c *Client) DeleteResourceTypePermission(permissionID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/resource-manager/permissions/%s", c.APIBaseURL, permissionID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, resourceTypePermissions) // Updated to include lock name
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}
