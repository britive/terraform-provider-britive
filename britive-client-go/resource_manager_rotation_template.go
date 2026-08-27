package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// rotationTemplateLanguageContentType maps a rotation template's inline-code language
// (the API's "editorType") to the Content-Type used when uploading the script content
// to its presigned URL. Confirmed by capture. Note this differs from resource type
// permissions' own code_language map (which uses the key "node") - rotation templates
// literally send "javascript" as editorType; the two features don't share vocabulary
// even though the resulting Content-Type happens to coincide for that one entry.
var rotationTemplateLanguageContentType = map[string]string{
	"text":       "text/plain",
	"python":     "text/x-python",
	"batch":      "text/x-batch",
	"javascript": "application/octet-stream",
	"powershell": "application/x-powershell",
	"shell":      "application/x-sh",
}

// CreateRotationTemplate creates a new rotation template stub (name/description only).
// The mode (local/inline-code/file), time limit, and variables are configured afterwards
// via UpdateRotationTemplate.
func (c *Client) CreateRotationTemplate(resourceTypeID string, template RotationTemplateCreateRequest) (*RotationTemplateSummary, error) {
	body, err := json.Marshal(template)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/resource-manager/resource-types/%s/rotation-templates", c.APIBaseURL, resourceTypeID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, rotationTemplateLockName)
	if err != nil {
		return nil, err
	}

	result := &RotationTemplateSummary{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetRotationTemplate retrieves a rotation template's full detail.
func (c *Client) GetRotationTemplate(resourceTypeID string, templateID string) (*RotationTemplate, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/resource-types/%s/rotation-templates/%s", c.APIBaseURL, resourceTypeID, templateID), nil)
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

	result := &RotationTemplate{}
	if err := json.Unmarshal(body, result); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateRotationTemplate saves a rotation template's mode/time-limit/variables
// configuration. Name/Description must be left unset on the passed-in template - the API
// never accepts them here (see RotationTemplate's doc comment).
func (c *Client) UpdateRotationTemplate(resourceTypeID string, templateID string, template RotationTemplate) (*RotationTemplate, error) {
	body, err := json.Marshal(template)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/resource-manager/resource-types/%s/rotation-templates/%s", c.APIBaseURL, resourceTypeID, templateID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, rotationTemplateLockName)
	if err != nil {
		return nil, err
	}

	result := &RotationTemplate{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteRotationTemplate deletes a rotation template.
func (c *Client) DeleteRotationTemplate(resourceTypeID string, templateID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/resource-manager/resource-types/%s/rotation-templates/%s", c.APIBaseURL, resourceTypeID, templateID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, rotationTemplateLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}

// GetRotationTemplatePresignedURL retrieves the presigned S3 URL to upload a rotation
// template's script content to. Used by both InlineFile (inline code) and FilePath
// (file upload) modes; not called at all for Local mode.
func (c *Client) GetRotationTemplatePresignedURL(resourceTypeID string, templateID string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/resource-types/%s/rotation-templates/presigned-url?templateId=%s", c.APIBaseURL, resourceTypeID, templateID), nil)
	if err != nil {
		return "", err
	}

	body, err := c.Do(req)
	if err != nil {
		return "", err
	}

	result := &RotationTemplatePresignedURL{}
	if err := json.Unmarshal(body, result); err != nil {
		return "", err
	}
	return result.PresignedURL, nil
}

// UploadRotationTemplateScriptCode uploads inline code content for a rotation template
// (InlineFile mode). Content-Type is derived from the language, mirroring how the UI's
// "Select Language" dropdown drives the upload's Content-Type.
func (c *Client) UploadRotationTemplateScriptCode(resourceTypeID string, templateID string, code string, language string) error {
	contentType, ok := rotationTemplateLanguageContentType[strings.ToLower(language)]
	if !ok {
		return fmt.Errorf("script language %q is unsupported", language)
	}

	presignedURL, err := c.GetRotationTemplatePresignedURL(resourceTypeID, templateID)
	if err != nil {
		return err
	}

	return uploadRotationTemplateScript(presignedURL, []byte(code), contentType)
}

// UploadRotationTemplateScriptFile uploads a local file's content for a rotation template
// (FilePath mode). Always sent as application/octet-stream, matching the capture: the "Add
// File" flow uploads whatever bytes are on disk verbatim, with no language selector.
func (c *Client) UploadRotationTemplateScriptFile(resourceTypeID string, templateID string, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	presignedURL, err := c.GetRotationTemplatePresignedURL(resourceTypeID, templateID)
	if err != nil {
		return err
	}

	return uploadRotationTemplateScript(presignedURL, content, "application/octet-stream")
}

// DownloadRotationTemplateScript fetches a rotation template's current script content from
// its presigned download URL (the RotationTemplate.PresignedURL field returned alongside
// GetRotationTemplate's detail response). Used for InlineFile-mode drift detection: Read()
// compares this live content against the configured script_content so out-of-band edits on
// the backend show up as a plan diff instead of going unnoticed. Not called for Local mode
// (no script exists) or FilePath mode (script_content isn't the tracked source of truth there).
func (c *Client) DownloadRotationTemplateScript(presignedURL string) (string, error) {
	req, err := http.NewRequest("GET", presignedURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", fmt.Errorf("error downloading rotation template script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error downloading rotation template script: status %s", resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading rotation template script response: %w", err)
	}
	return string(content), nil
}

// uploadRotationTemplateScript performs the actual presigned-URL PUT and, unlike
// UploadFile/UploadCode above, treats a non-2xx response as a real error rather than
// only logging it - there's no existing behavior to stay compatible with here, so this
// follows AddRemoveIcon's stricter precedent instead.
func uploadRotationTemplateScript(presignedURL string, content []byte, contentType string) error {
	resp, err := putToPresignedURL(presignedURL, content, contentType)
	if err != nil {
		return fmt.Errorf("error uploading rotation template script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("error uploading rotation template script: status %s", resp.Status)
	}
	return nil
}
