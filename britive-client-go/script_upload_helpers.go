package britive

import (
	"fmt"
	"io"
	"net/http"
)

// scriptLanguageContentType maps an inline-code language (the API's "editorType") to the
// Content-Type used when uploading script content to a presigned URL. Confirmed by capture
// for rotation templates; scan settings only exercised "text" but shares the same field
// shape and is assumed to accept the same enum. Note this differs from resource type
// permissions' own code_language map (which uses the key "node" instead of "javascript") -
// that feature doesn't share vocabulary with these two, even though the resulting
// Content-Type happens to coincide for that one entry.
var scriptLanguageContentType = map[string]string{
	"text":       "text/plain",
	"python":     "text/x-python",
	"batch":      "text/x-batch",
	"javascript": "application/octet-stream",
	"powershell": "application/x-powershell",
	"shell":      "application/x-sh",
}

// uploadPresignedScript performs a presigned-URL PUT of script/file content and treats a
// non-2xx response as a real error (unlike UploadFile/UploadCode in
// resource_manager_resource_type_permissions.go, which only log on failure - there's no
// existing behavior to stay compatible with for rotation templates/scan settings, so this
// follows AddRemoveIcon's stricter precedent instead). Shared by rotation templates' and
// scan settings' script/file upload functions.
func uploadPresignedScript(presignedURL string, content []byte, contentType string) error {
	resp, err := putToPresignedURL(presignedURL, content, contentType)
	if err != nil {
		return fmt.Errorf("error uploading script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("error uploading script: status %s", resp.Status)
	}
	return nil
}

// DownloadPresignedContent fetches content from a presigned download URL (the
// "presignedUrl" field returned alongside a rotation template's or scan settings' detail
// response). Used for InlineFile-mode drift detection: Read() compares this live content
// against the configured script_content so an out-of-band edit on the backend shows up as
// a plan diff instead of going unnoticed, and for FilePath-mode drift detection by hashing
// the result. Not called for Local mode (no script exists).
func (c *Client) DownloadPresignedContent(presignedURL string) (string, error) {
	req, err := http.NewRequest("GET", presignedURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", fmt.Errorf("error downloading script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error downloading script: status %s", resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading script download response: %w", err)
	}
	return string(content), nil
}
