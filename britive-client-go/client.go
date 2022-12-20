package britive

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var (
	syncOnce sync.Once
	client   *Client
)

// Client - Britive API client
type Client struct {
	APIBaseURL string
	HTTPClient *http.Client
	Token      string
	Version    string
	SyncMap    *sync.Map
}

// NewClient - Initializes new Britive API client
func NewClient(apiBaseURL, token, version string) (*Client, error) {
	syncOnce.Do(func() {
		client = &Client{
			HTTPClient: &http.Client{Timeout: 0},
			APIBaseURL: apiBaseURL,
			Token:      token,
			Version:    version,
			SyncMap:    &sync.Map{},
		}
	})
	return client, nil
}

// QueryRequest - godoc
type QueryRequest struct {
	Client      *Client
	QueryParams map[string]string
	Lock        string
	Result      interface{}
}

// SortDirection - godoc
type SortDirection string

const (
	//SortDirectionAscending - godoc
	SortDirectionAscending SortDirection = "asc"
	//SortDirectionDescending - godoc
	SortDirectionDescending SortDirection = "desc"
)

// WithQuery - godoc
func (gpr *QueryRequest) WithQuery(query string) *QueryRequest {
	if query != emptyString {
		gpr.QueryParams["query"] = url.QueryEscape(query)
	}
	return gpr
}

// WithFilter - godoc
func (gpr *QueryRequest) WithFilter(filter string) *QueryRequest {
	if filter != emptyString {
		gpr.QueryParams["filter"] = url.QueryEscape(filter)
	}
	return gpr
}

// WithSort - godoc
func (gpr *QueryRequest) WithSort(name string, direction SortDirection) *QueryRequest {
	if name != emptyString && direction != emptyString {
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
		if gpr.Lock == emptyString {
			response, err = gpr.Client.Do(req)
		} else {
			response, err = gpr.Client.DoWithLock(req, gpr.Lock)
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

// DoWithLock - Perform Britive API call with lock
func (c *Client) DoWithLock(req *http.Request, key string) ([]byte, error) {
	c.lock(key)
	defer c.unlock(key)
	return c.Do(req)
}

// Do - Perform Britive API call
func (c *Client) Do(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", fmt.Sprintf("TOKEN %s", c.Token))
	req.Header.Set("Content-Type", "application/json")
	userAgent := fmt.Sprintf("britive-client-go/%s golang/%s %s/%s britive-terraform/%s", c.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH, c.Version)
	req.Header.Add("User-Agent", userAgent)

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNoContent {
		return []byte(emptyString), ErrNoContent
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusNotFound {
		return body, ErrNotFound
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusAccepted {
		var httpErrorResponse HTTPErrorResponse
		err = json.Unmarshal(body, &httpErrorResponse)
		if err == nil && httpErrorResponse.Message != emptyString {
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

func ArrayOfMapsEqual(old, new string) bool {

	equalCount := 0

	if old == emptyString {
		old = "[]"
	}

	if new == emptyString {
		new = "[]"
	}

	oldArray := []map[string]interface{}{}
	if err := json.Unmarshal([]byte(old), &oldArray); err != nil {
		panic(err)
	}

	newArray := []map[string]interface{}{}
	if err := json.Unmarshal([]byte(new), &newArray); err != nil {
		panic(err)
	}

	if len(oldArray) == len(newArray) {
		for _, v := range oldArray {
			for _, p := range newArray {
				if reflect.DeepEqual(v, p) {
					equalCount++
				}
			}
		}
		if equalCount != len(newArray) {
			return false
		}
	} else {
		return false
	}
	return true
}
