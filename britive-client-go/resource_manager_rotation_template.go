package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

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

	result := &PresignedURLResponse{}
	if err := json.Unmarshal(body, result); err != nil {
		return "", err
	}
	return result.PresignedURL, nil
}

// UploadRotationTemplateScriptCode uploads inline code content for a rotation template
// (InlineFile mode). Content-Type is derived from the language, mirroring how the UI's
// "Select Language" dropdown drives the upload's Content-Type.
func (c *Client) UploadRotationTemplateScriptCode(resourceTypeID string, templateID string, code string, language string) error {
	contentType, ok := scriptLanguageContentType[strings.ToLower(language)]
	if !ok {
		return fmt.Errorf("script language %q is unsupported", language)
	}

	presignedURL, err := c.GetRotationTemplatePresignedURL(resourceTypeID, templateID)
	if err != nil {
		return err
	}

	return uploadPresignedScript(presignedURL, []byte(code), contentType)
}

// UploadRotationTemplateScriptFile uploads a local file's content for a rotation template
// (FilePath mode). Always sent as application/octet-stream, matching the capture: the "Add
// File" flow uploads whatever bytes are on disk verbatim, with no language selector.
//
// Note: scan settings' equivalent upload derives Content-Type from the file's own extension
// instead (confirmed by a separate capture using a .txt file). This function intentionally
// keeps the original hardcoded behavior rather than silently changing rotation templates'
// established upload behavior as a side effect of adding scan settings.
func (c *Client) UploadRotationTemplateScriptFile(resourceTypeID string, templateID string, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	presignedURL, err := c.GetRotationTemplatePresignedURL(resourceTypeID, templateID)
	if err != nil {
		return err
	}

	return uploadPresignedScript(presignedURL, content, "application/octet-stream")
}
