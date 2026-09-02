package britive

import (
	"bytes"
	"net/http"
)

// putToPresignedURL performs a raw PUT of content to a presigned URL (e.g. an S3 upload
// URL issued by the Britive API), setting the given Content-Type. It intentionally does
// not go through c.Do/c.DoWithLock: a presigned URL is pre-authorized via its own
// query-string signature and must not carry the Britive Authorization/User-Agent headers
// those helpers add.
//
// Shared by every feature that uploads a file or inline code via a Britive-issued presigned
// URL (resource type permission checkin/checkout files and codes, rotation template scripts,
// and future resources with the same shape). Callers decide how to interpret the response
// status, since existing callers differ in how strictly they treat a non-2xx response.
func putToPresignedURL(presignedURL string, content []byte, contentType string) (*http.Response, error) {
	req, err := http.NewRequest("PUT", presignedURL, bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	return (&http.Client{}).Do(req)
}
