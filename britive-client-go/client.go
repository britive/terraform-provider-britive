package britive

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Client - godoc
type Client struct {
	HostURL    string
	HTTPClient *http.Client
	Token      string
}

// NewClient - godoc
func NewClient(host, token *string) (*Client, error) {
	if token == nil {
		return nil, fmt.Errorf("Token must not be empty")
	}
	c := Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		HostURL:    *host,
		Token:      *token,
	}
	return &c, nil
}

//DoRequest - godoc
func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", fmt.Sprintf("TOKEN %s", c.Token))
	req.Header.Set("Content-Type", "application/json")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("\nrequest: %+v\nstatus: %d\nbody: %s", *req, res.StatusCode, body)
	}

	return body, err
}
