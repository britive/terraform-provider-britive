package britive

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"time"
)

// Client - Britive API client
type Client struct {
	APIBaseURL string
	HTTPClient *http.Client
	Token      string
	Version    string
}

// NewClient - Initialises new Britive API client
func NewClient(apiBaseURL, token, version string) (*Client, error) {
	c := Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		APIBaseURL: apiBaseURL,
		Token:      token,
		Version:    version,
	}
	return &c, nil
}

//DoRequest - Perform Britive API call
func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", fmt.Sprintf("TOKEN %s", c.Token))
	req.Header.Set("Content-Type", "application/json")
	userAgent := fmt.Sprintf("britive-client-go/%s golang/%s %s/%s britive-terraform/%s", c.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH, c.Version)
	req.Header.Add("User-Agent", userAgent)

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("An error occured while processing the request\nrequestUrl: %s\nrequestMethod: %s\nresponseStatus: %d\nresponseBody: %s", req.URL, req.Method, res.StatusCode, body)
	}

	return body, err
}
