package britive

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Scan settings are a resource-type-scoped singleton, not a named collection like rotation
// templates: there's exactly one per resource type, no name/description, and it's created
// via an idempotent PUT (201 the first time, 200 after) rather than a POST-then-PUT
// two-step. GetScanSettings returns {} (200, not 404) when never configured.

// GetScanSettings retrieves a resource type's scan settings.
func (c *Client) GetScanSettings(resourceTypeID string) (*ScanSettings, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/resource-types/%s/scan-settings", c.APIBaseURL, resourceTypeID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	result := &ScanSettings{}
	if err := json.Unmarshal(body, result); err != nil {
		return nil, err
	}
	return result, nil
}

// UpsertScanSettings creates or updates a resource type's scan settings - the API doesn't
// distinguish the two, so this single function backs both Create and Update.
func (c *Client) UpsertScanSettings(resourceTypeID string, settings ScanSettings) (*ScanSettings, error) {
	body, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/resource-manager/resource-types/%s/scan-settings", c.APIBaseURL, resourceTypeID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, scanSettingsLockName)
	if err != nil {
		return nil, err
	}

	result := &ScanSettings{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetScanSettingsPresignedURL retrieves the presigned S3 URL to upload a resource type's
// scan settings script content to. Unlike rotation templates' equivalent, no query
// parameter is needed - scan settings is a singleton per resource type, so resourceTypeID
// alone identifies it.
func (c *Client) GetScanSettingsPresignedURL(resourceTypeID string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/resource-types/%s/scan-settings/presigned-url", c.APIBaseURL, resourceTypeID), nil)
	if err != nil {
		return "", err
	}

	body, err := c.Do(req)
	if err != nil {
		return "", err
	}

	result := &PresignedURLResponse{}
	if err := json.Unmarshal(body, result); err != nil {
		return "", err
	}
	return result.PresignedURL, nil
}

// UploadScanSettingsScriptCode uploads inline code content for a resource type's scan
// settings (InlineFile mode). Content-Type is derived from the language, same as rotation
// templates' equivalent.
func (c *Client) UploadScanSettingsScriptCode(resourceTypeID string, code string, language string) error {
	contentType, ok := scriptLanguageContentType[strings.ToLower(language)]
	if !ok {
		return fmt.Errorf("script language %q is unsupported", language)
	}

	presignedURL, err := c.GetScanSettingsPresignedURL(resourceTypeID)
	if err != nil {
		return err
	}

	return uploadPresignedScript(presignedURL, []byte(code), contentType)
}

// UploadScanSettingsScriptFile uploads a local file's content for a resource type's scan
// settings (FilePath mode). Content-Type is derived from the file's own extension (a
// separate capture using a .txt file showed Content-Type: text/plain, not the hardcoded
// application/octet-stream rotation templates' FilePath mode uses), falling back to
// application/octet-stream for an unrecognized or absent extension - matching what a
// browser's File API would report.
func (c *Client) UploadScanSettingsScriptFile(resourceTypeID string, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	presignedURL, err := c.GetScanSettingsPresignedURL(resourceTypeID)
	if err != nil {
		return err
	}

	return uploadPresignedScript(presignedURL, content, contentType)
}
