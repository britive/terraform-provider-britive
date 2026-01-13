package britive_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type Client struct {
	APIBaseURL string
	HTTPClient *http.Client
	Token      string
	Version    string
	SyncMap    *sync.Map
}

func NewClient(apiBaseURL, token, version string) (*Client, error) {
	return &Client{
		HTTPClient: &http.Client{Timeout: 0},
		APIBaseURL: apiBaseURL,
		Token:      token,
		Version:    version,
		SyncMap:    &sync.Map{},
	}, nil
}

type QueryRequest struct {
	Client      *Client
	QueryParams map[string]string
	Lock        string
	Result      interface{}
}

// DoWithLock - Perform API call with lock
func (c *Client) DoWithLock(ctx context.Context, req *http.Request, key string) ([]byte, error) {
	c.lock(key)
	defer c.unlock(key)
	return c.Do(ctx, req)
}

// Do - Perform API call
func (c *Client) Do(ctx context.Context, req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	req.Header.Set("Content-Type", "application/json")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNoContent {
		return []byte(""), ErrNoContent
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusNotFound {
		return body, ErrNotFound
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusAccepted {
		var httpErrorResponse HTTPErrorResponse
		err = json.Unmarshal(body, &httpErrorResponse)
		if err == nil && httpErrorResponse.Message != "" {
			return nil, fmt.Errorf("%s: %s", httpErrorResponse.ErrorCode, httpErrorResponse.Message)
		}
		return nil, fmt.Errorf("an error occurred while processing the request\nrequest url: %s\nrequest method: %s\nresponse status: %d\nresponse body: %s", req.URL, req.Method, res.StatusCode, body)
	}

	return body, err
}

// Lock to lock based on key
func (c *Client) lock(key interface{}) {
	mutex := &sync.Mutex{}
	actual, _ := c.SyncMap.LoadOrStore(key, mutex)
	actualMutex := actual.(*sync.Mutex)
	actualMutex.Lock()
	if actualMutex != mutex {
		actualMutex.Unlock()
		c.lock(key)
		return
	}
}

// Unlock to unlock based on key
func (c *Client) unlock(key interface{}) {
	actual, exist := c.SyncMap.Load(key)
	if !exist {
		return
	}
	actualMutex := actual.(*sync.Mutex)
	c.SyncMap.Delete(key)
	actualMutex.Unlock()
}

// WithQuery - godoc
func (gpr *QueryRequest) WithQuery(query string) *QueryRequest {
	if query != "" {
		gpr.QueryParams["query"] = url.QueryEscape(query)
	}
	return gpr
}

// SortDirection - godoc
type SortDirection string

// WithFilter - godoc
func (gpr *QueryRequest) WithFilter(filter string) *QueryRequest {
	if filter != "" {
		gpr.QueryParams["filter"] = url.QueryEscape(filter)
	}
	return gpr
}

// WithSort - godoc
func (gpr *QueryRequest) WithSort(name string, direction SortDirection) *QueryRequest {
	if name != "" && direction != "" {
		gpr.QueryParams["sort"] = fmt.Sprintf("%s,%s", name, direction)
	}
	return gpr
}

// WithSize - godoc
func (gpr *QueryRequest) WithSize(size int) *QueryRequest {
	if size > 0 {
		gpr.QueryParams["size"] = strconv.Itoa(size)
	}
	return gpr
}

// WithLock - godoc
func (gpr *QueryRequest) WithLock(lock string) *QueryRequest {
	gpr.Lock = lock
	return gpr
}

// WithResult - godoc
func (gpr *QueryRequest) WithResult(result interface{}) *QueryRequest {
	gpr.Result = result
	return gpr
}

// NewQueryRequest - godoc
func (c *Client) NewQueryRequest() *QueryRequest {
	return &QueryRequest{
		Client:      c,
		QueryParams: make(map[string]string),
	}
}

// Query - godoc
func (gpr *QueryRequest) Query(endpoint string) error {
	const size = 10
	var page = 0
	result := reflect.ValueOf(gpr.Result).Elem()
	if _, ok := gpr.QueryParams["size"]; !ok {
		gpr.QueryParams["size"] = strconv.Itoa(size)
	}
	for {
		gpr.QueryParams["page"] = strconv.Itoa(page)
		queryParams := []string{}

		for k, v := range gpr.QueryParams {
			queryParams = append(queryParams, fmt.Sprintf("%s=%s", k, v))
		}
		url := fmt.Sprintf("%s/%s?%s", gpr.Client.APIBaseURL, endpoint, strings.Join(queryParams, "&"))
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		var response []byte
		var ctx context.Context
		if gpr.Lock == "" {
			response, err = gpr.Client.Do(ctx, req)
		} else {
			response, err = gpr.Client.DoWithLock(ctx, req, gpr.Lock)
		}
		if err != nil {
			return err
		}
		var pr PaginationResponse
		err = json.Unmarshal(response, &pr)
		if err != nil {
			return err
		}
		if len(pr.Data) > 0 {
			for _, d := range pr.Data {
				ds, err := json.Marshal(d.(map[string]interface{}))
				if err != nil {
					return err
				}
				fr := reflect.New(result.Type().Elem())
				fro := fr.Interface()
				err = json.Unmarshal(ds, &fro)
				if err != nil {
					return err
				}
				result.Set(reflect.Append(result, fr.Elem()))
			}
		}
		page = pr.Page + 1
		if pr.Count < (page)*pr.Size {
			break
		}
	}
	return nil
}

func (c *Client) Post(ctx context.Context, url string, body interface{}, lock string) ([]byte, error) {
	bodyContent, _ := json.Marshal(body)
	bodyReader := bytes.NewReader(bodyContent)

	req, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(ctx, req, lock)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func (c *Client) Get(ctx context.Context, url string, lock string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(ctx, req, lock)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func (c *Client) Put(ctx context.Context, url string, body interface{}, lock string) ([]byte, error) {
	bodyContent, _ := json.Marshal(body)
	bodyReader := bytes.NewReader(bodyContent)

	req, err := http.NewRequest("PUT", url, bodyReader)
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(ctx, req, lock)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) Patch(ctx context.Context, url string, body interface{}, lock string) ([]byte, error) {
	bodyContent, _ := json.Marshal(body)
	bodyReader := bytes.NewReader(bodyContent)

	req, err := http.NewRequest("PATCH", url, bodyReader)
	if err != nil {
		return nil, err
	}

	resp, err := c.DoWithLock(ctx, req, lock)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) Delete(ctx context.Context, url string, lock string) error {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(ctx, req, lock)
	if err != nil {
		return err
	}

	return nil
}

func ConditionConstraintEqual(newTitle, newExpression, newDescription string, constraintResult *ConditionConstraintResult) bool {
	equalCount := 0

	if newTitle == EmptyString {
		newTitle = ""
	}

	if newExpression == EmptyString {
		newExpression = ""
	}

	if newDescription == EmptyString {
		newDescription = ""
	}

	for _, co := range constraintResult.Result {
		if newTitle == co.Title {
			equalCount++
		}
		if newExpression == co.Expression {
			equalCount++
		}
		if newDescription == co.Description {
			equalCount++
		}
	}
	if equalCount != 3 {
		return false
	}
	return true
}

func ConstraintEqual(newName string, constraintResult *ConstraintResult) bool {
	equalCount := 0

	if newName == EmptyString {
		newName = ""
	}

	for _, co := range constraintResult.Result {
		if newName == co.Name {
			equalCount++
		}
	}
	if equalCount != 1 {
		return false
	}
	return true
}
