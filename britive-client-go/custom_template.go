package britive

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"runtime"
)

func (c *Client) CreateCustomTemplate(fileContent, fileName string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("templateFile", fileName)
	if err != nil {
		return err
	}

	_, err = io.Copy(part, bytes.NewReader([]byte(fileContent)))
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/generic-apps/templates", c.APIBaseURL), body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("TOKEN %s", c.Token))
	req.Header.Set("Content-Type", writer.FormDataContentType())
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

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusAccepted {
		log.Println("Upload successful!")
	} else {
		fmt.Printf("Upload failed. Status: %s\n", resp.Status)
		return fmt.Errorf("Upload failed. Status: %s\n", resp.Status)
	}

	return nil

}
