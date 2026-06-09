package britive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	retryWaitMin = time.Second       // minimum wait between retries
	retryWaitMax = 600 * time.Second // maximum wait between retries
)

var (
	syncOnce sync.Once
	client   *Client
)

// Client - Britive API client
type Client struct {
	APIBaseURL   string
	HTTPClient   *http.Client
	Token        string
	Version      string
	SyncMap      *sync.Map
	MaxRetries   int
	RetryWaitMin time.Duration
	RetryWaitMax time.Duration
}

// NewClient - Initializes new Britive API client
func NewClient(apiBaseURL, token, version string, maxRetries, retryWaitMinSecs, retryWaitMaxSecs int) (*Client, error) {
	syncOnce.Do(func() {
		if maxRetries <= 0 {
			maxRetries = defaultMaxRetries
		}
		waitMin := retryWaitMin
		if retryWaitMinSecs > 0 {
			waitMin = time.Duration(retryWaitMinSecs) * time.Second
		}
		waitMax := retryWaitMax
		if retryWaitMaxSecs > 0 {
			waitMax = time.Duration(retryWaitMaxSecs) * time.Second
		}
		client = &Client{
			HTTPClient:   &http.Client{Timeout: 0},
			APIBaseURL:   apiBaseURL,
			Token:        token,
			Version:      version,
			SyncMap:      &sync.Map{},
			MaxRetries:   maxRetries,
			RetryWaitMin: waitMin,
			RetryWaitMax: waitMax,
		}
	})
	return client, nil
}

func init() {
	rand.Seed(time.Now().UnixNano()) //nolint:staticcheck
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

// Do - Perform Britive API call with exponential backoff retry on HTTP 429
func (c *Client) Do(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", fmt.Sprintf("TOKEN %s", c.Token))
	req.Header.Set("Content-Type", "application/json")
	userAgent := fmt.Sprintf("britive-client-go/%s golang/%s %s/%s britive-terraform/%s", c.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH, c.Version)
	req.Header.Add("User-Agent", userAgent)

	// Preserve request body bytes so they can be replayed on retry
	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		var readErr error
		bodyBytes, readErr = ioutil.ReadAll(req.Body)
		req.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
	}

	log.Printf("[DEBUG] britive-retry: %s %s (max_retries=%d)", req.Method, req.URL, c.MaxRetries)

	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 && bodyBytes != nil {
			req.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
			req.ContentLength = int64(len(bodyBytes))
		}

		log.Printf("[DEBUG] britive-retry: sending request (attempt %d/%d) %s %s", attempt+1, c.MaxRetries+1, req.Method, req.URL)

		res, err := c.HTTPClient.Do(req)
		if err != nil {
			log.Printf("[DEBUG] britive-retry: request error on attempt %d/%d: %s", attempt+1, c.MaxRetries+1, err)
			return nil, err
		}

		log.Printf("[DEBUG] britive-retry: response status %d on attempt %d/%d for %s %s", res.StatusCode, attempt+1, c.MaxRetries+1, req.Method, req.URL)

		if res.StatusCode == http.StatusTooManyRequests {
			retryAfter := res.Header.Get("Retry-After")
			ioutil.ReadAll(res.Body) //nolint:errcheck
			res.Body.Close()
			if attempt >= c.MaxRetries {
				log.Printf("[WARN] britive-retry: rate limited, exhausted all %d retries for %s %s", c.MaxRetries, req.Method, req.URL)
				break
			}
			wait := calculateBackoff(attempt, c.RetryWaitMin, c.RetryWaitMax, retryAfter)
			log.Printf("[WARN] britive-retry: rate limited (HTTP 429) on attempt %d/%d, waiting %s before retry (Retry-After header: %q)", attempt+1, c.MaxRetries+1, wait, retryAfter)
			time.Sleep(wait)
			log.Printf("[DEBUG] britive-retry: resuming after wait, next attempt %d/%d", attempt+2, c.MaxRetries+1)
			continue
		}

		defer res.Body.Close()

		if res.StatusCode == http.StatusNoContent {
			log.Printf("[DEBUG] britive-retry: success (204 No Content) on attempt %d/%d", attempt+1, c.MaxRetries+1)
			return []byte(emptyString), ErrNoContent
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		if res.StatusCode == http.StatusNotFound {
			log.Printf("[DEBUG] britive-retry: not found (404) on attempt %d/%d for %s %s", attempt+1, c.MaxRetries+1, req.Method, req.URL)
			return body, ErrNotFound
		}

		if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusAccepted {
			var httpErrorResponse HTTPErrorResponse
			err = json.Unmarshal(body, &httpErrorResponse)
			if err == nil && httpErrorResponse.Message != emptyString {
				log.Printf("[DEBUG] britive-retry: failed (status %d) on attempt %d/%d: %s: %s", res.StatusCode, attempt+1, c.MaxRetries+1, httpErrorResponse.ErrorCode, httpErrorResponse.Message)
				return nil, fmt.Errorf("%s: %s", httpErrorResponse.ErrorCode, httpErrorResponse.Message)
			}
			log.Printf("[DEBUG] britive-retry: failed (status %d) on attempt %d/%d for %s %s", res.StatusCode, attempt+1, c.MaxRetries+1, req.Method, req.URL)
			return nil, fmt.Errorf("an error occurred while processing the request\nrequest url: %s\nrequest method: %s\nresponse status: %d\nresponse body: %s", req.URL, req.Method, res.StatusCode, body)
		}

		log.Printf("[DEBUG] britive-retry: success (status %d) on attempt %d/%d for %s %s", res.StatusCode, attempt+1, c.MaxRetries+1, req.Method, req.URL)
		return body, nil
	}

	return nil, fmt.Errorf("rate limited: request to %s failed after %d retries", req.URL, c.MaxRetries)
}

// calculateBackoff returns the wait duration before the next retry.
// Honors the Retry-After header when present; otherwise uses exponential
// backoff with full jitter: random(0, min(waitMax, waitMin * 2^attempt)).
func calculateBackoff(attempt int, waitMin, waitMax time.Duration, retryAfterHeader string) time.Duration {
	if retryAfterHeader != "" {
		if seconds, err := strconv.Atoi(retryAfterHeader); err == nil && seconds > 0 {
			wait := time.Duration(seconds) * time.Second
			if wait < waitMin {
				return waitMin
			}
			if wait > waitMax {
				return waitMax
			}
			return wait
		}
	}
	cap := math.Min(float64(waitMax), float64(waitMin)*math.Pow(2, float64(attempt)))
	return time.Duration(rand.Float64() * cap)
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

func MembersEqual(old, new string) bool {

	equalCount := 0

	if old == emptyString {
		old = "{}"
	}

	if new == emptyString {
		new = "{}"
	}

	var oldArray map[string][]map[string]interface{}
	if err := json.Unmarshal([]byte(old), &oldArray); err != nil {
		panic(err)
	}

	var newArray map[string][]map[string]interface{}
	if err := json.Unmarshal([]byte(new), &newArray); err != nil {
		panic(err)
	}

	if len(oldArray) == len(newArray) {
		for key, val := range oldArray {
			memOld, err := json.Marshal(val)
			if err != nil {
				panic(err)
			}
			memNew, err := json.Marshal(newArray[key])
			if err != nil {
				panic(err)
			}
			switch key {
			case "serviceIdentities":
				if ArrayOfMapsEqual(string(memOld), string(memNew)) {
					equalCount++
				}
			case "tags":
				if ArrayOfMapsEqual(string(memOld), string(memNew)) {
					equalCount++
				}
			case "tokens":
				if ArrayOfMapsEqual(string(memOld), string(memNew)) {
					equalCount++
				}
			case "users":
				if ArrayOfMapsEqual(string(memOld), string(memNew)) {
					equalCount++
				}
			case "aiIdentities":
				if ArrayOfMapsEqual(string(memOld), string(memNew)) {
					equalCount++
				}
			default:
				return false
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

func ConditionEqual(old, new string) bool {

	count := 3
	equalCount := 0

	if old == emptyString {
		old = "{}"
	}

	if new == emptyString {
		new = "{}"
	}

	var oldArray map[string]interface{}
	if err := json.Unmarshal([]byte(old), &oldArray); err != nil {
		panic(err)
	}

	var newArray map[string]interface{}
	if err := json.Unmarshal([]byte(new), &newArray); err != nil {
		panic(err)
	}

	items := []string{"approval", "ipAddress", "timeOfAccess"}

	for i := 0; i < count; i++ {
		memOld := []byte(emptyString)
		memNew := []byte(emptyString)
		memOld, err := json.Marshal(oldArray[items[i]])
		if err != nil {
			panic(err)
		}
		memNew, err = json.Marshal(newArray[items[i]])
		if err != nil {
			panic(err)
		}
		switch items[i] {
		case "approval":
			if ApprovalBlockEqual(string(memOld), string(memNew)) {
				equalCount++
			}
		case "ipAddress":
			if IPAddressBlockEqual(string(memOld), string(memNew)) {
				equalCount++
			}
		case "timeOfAccess":
			if TimeOfAccessBlockEqual(string(memOld), string(memNew)) {
				equalCount++
			}
		default:
			return false
		}
	}
	if equalCount != count {
		return false
	}

	return true
}

func ApprovalBlockEqual(old, new string) bool {

	equalCount := 0

	if old == emptyString {
		old = "{}"
	}

	if new == emptyString {
		new = "{}"
	}

	var oldArray map[string]interface{}
	if err := json.Unmarshal([]byte(old), &oldArray); err != nil {
		panic(err)
	}

	var newArray map[string]interface{}
	if err := json.Unmarshal([]byte(new), &newArray); err != nil {
		panic(err)
	}

	isNewManagerApprovalReq := false

	if val, ok := newArray["managerApproval"]; ok {
		managerApproval := val.(map[string]interface{})
		if reqVal, ok := managerApproval["required"]; (ok && reqVal == false) || !ok {
			isNewManagerApprovalReq = true
		}
	}

	if _, ok := oldArray["managerApproval"]; !ok && isNewManagerApprovalReq {
		oldArray["managerApproval"] = newArray["managerApproval"]
	}

	if len(oldArray) == len(newArray) {
		for key, val := range oldArray {
			memOld, err := json.Marshal(val)
			if err != nil {
				panic(err)
			}
			memNew, err := json.Marshal(newArray[key])
			if err != nil {
				panic(err)
			}
			switch key {
			case "approvers":
				if ApproversBlockEqual(string(memOld), string(memNew)) {
					equalCount++
				}
			case "managerApproval":
				if ManagerApprovalBlockEqual(string(memOld), string(memNew)) {
					equalCount++
				}
			case "isValidForInDays":
				if string(memOld) == string(memNew) {
					equalCount++
				}
			case "notificationMedium":
				if val == nil {
					val = "{}"
				}
				if newArray[key] == nil {
					newArray[key] = "{}"
				}
				if reflect.TypeOf(val).Name() == "string" || reflect.TypeOf(newArray[key]).Name() == "string" {
					if string(memOld) == string(memNew) {
						equalCount++
					}
				} else {
					if ArrayOfInterfaceEqual(val, newArray[key]) {
						equalCount++
					}
				}
			case "timeToApprove":
				if string(memOld) == string(memNew) {
					equalCount++
				}
			case "validFor":
				if string(memOld) == string(memNew) {
					equalCount++
				}
			default:
				return false
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

func ApproversBlockEqual(old, new string) bool {
	equalCount := 0

	if old == emptyString {
		old = "{}"
	}

	if new == emptyString {
		new = "{}"
	}

	oldArray := make(map[string][]interface{})

	if err := json.Unmarshal([]byte(old), &oldArray); err != nil {
		panic(err)
	}

	newArray := make(map[string][]interface{})

	if err := json.Unmarshal([]byte(new), &newArray); err != nil {
		panic(err)
	}

	if len(oldArray) == len(newArray) {
		for key, val := range oldArray {
			switch key {
			case "tags":
				if ArrayOfInterfaceEqual(val, newArray[key]) {
					equalCount++
				}
			case "userIds":
				if ArrayOfInterfaceEqual(val, newArray[key]) {
					equalCount++
				}
			case "channelIds":
				if ArrayOfInterfaceEqual(val, newArray[key]) {
					equalCount++
				}
			case "slackAppChannels":
				if ArrayOfInterfaceEqual(val, newArray[key]) {
					equalCount++
				}
			case "teamsAppChannels":
				if TeamsAppChannelsBlockEqual(val, newArray[key]) {
					equalCount++
				}
			default:
				return false
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

func ManagerApprovalBlockEqual(old, new string) bool {
	equalCount := 0

	if old == emptyString {
		old = "{}"
	}

	if new == emptyString {
		new = "{}"
	}

	oldArray := make(map[string]interface{})

	if err := json.Unmarshal([]byte(old), &oldArray); err != nil {
		panic(err)
	}

	newArray := make(map[string]interface{})

	if err := json.Unmarshal([]byte(new), &newArray); err != nil {
		panic(err)
	}

	if len(oldArray) == len(newArray) {
		for key, val := range oldArray {
			switch key {
			case "condition":
				if val == newArray[key] {
					equalCount++
				}
			case "required":
				if val == newArray[key] {
					equalCount++
				}
			default:
				return false
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

func TimeOfAccessBlockEqual(old, new string) bool {
	equalCount := 0

	if old == emptyString {
		old = "{}"
	}

	if new == emptyString {
		new = "{}"
	}

	var oldArray map[string]interface{}
	if err := json.Unmarshal([]byte(old), &oldArray); err != nil {
		panic(err)
	}

	var newArray map[string]interface{}
	if err := json.Unmarshal([]byte(new), &newArray); err != nil {
		panic(err)
	}

	if len(oldArray) == len(newArray) {
		for key, val := range oldArray {
			memOld, err := json.Marshal(val)
			if err != nil {
				panic(err)
			}
			memNew, err := json.Marshal(newArray[key])
			if err != nil {
				panic(err)
			}
			switch key {
			case "dateSchedule":
				if reflect.DeepEqual(memOld, memNew) {
					equalCount++
				}
			case "daysSchedule":
				if DaysScheduleBlockEqual(string(memOld), string(memNew)) {
					equalCount++
				}
			default:
				return false
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

func SliceIgnoreOrderEqual(old, new []string) bool {
	if len(old) != len(new) {
		return false
	}
	sort.Strings(old)
	sort.Strings(new)

	return reflect.DeepEqual(old, new)
}

func ArrayOfInterfaceEqual(old, new interface{}) bool {
	oldInterface := old.([]interface{})
	newInterface := new.([]interface{})

	oldSlice := make([]string, len(oldInterface))
	for i, v := range oldInterface {
		oldSlice[i] = v.(string)
	}
	newSlice := make([]string, len(newInterface))
	for i, v := range newInterface {
		newSlice[i] = v.(string)
	}
	return SliceIgnoreOrderEqual(oldSlice, newSlice)
}

func DaysScheduleBlockEqual(old, new string) bool {
	equalCount := 0

	if old == emptyString {
		old = "{}"
	}

	if new == emptyString {
		new = "{}"
	}

	oldArray := make(map[string]interface{})
	if err := json.Unmarshal([]byte(old), &oldArray); err != nil {
		panic(err)
	}

	newArray := make(map[string]interface{})
	if err := json.Unmarshal([]byte(new), &newArray); err != nil {
		panic(err)
	}

	if len(oldArray) == len(newArray) {
		for key, val := range oldArray {
			memOld, err := json.Marshal(val)
			if err != nil {
				panic(err)
			}
			memNew, err := json.Marshal(newArray[key])
			if err != nil {
				panic(err)
			}
			switch key {
			case "fromTime":
				if string(memOld) == string(memNew) {
					equalCount++
				}
			case "toTime":
				if string(memOld) == string(memNew) {
					equalCount++
				}
			case "timezone":
				if string(memOld) == string(memNew) {
					equalCount++
				}
			case "days":
				if ArrayOfInterfaceEqual(val, newArray[key]) {
					equalCount++
				}
			default:
				return false
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

func IPAddressBlockEqual(old, new string) bool {

	if old == emptyString {
		old = ""
	}

	if new == emptyString {
		new = ""
	}

	if len(old) != len(new) {
		return false
	}

	old = strings.TrimPrefix(old, "\"")
	new = strings.TrimPrefix(new, "\"")
	old = strings.TrimSuffix(old, "\"")
	new = strings.TrimSuffix(new, "\"")

	oldSlice := strings.Split(strings.TrimSpace(old), ",")
	newSlice := strings.Split(strings.TrimSpace(new), ",")

	for i := range oldSlice {
		oldSlice[i] = strings.TrimSpace(oldSlice[i])
	}
	for j := range newSlice {
		newSlice[j] = strings.TrimSpace(newSlice[j])
	}

	return SliceIgnoreOrderEqual(oldSlice, newSlice)
}

func ConditionConstraintEqual(newTitle, newExpression, newDescription string, constraintResult *ConditionConstraintResult) bool {
	equalCount := 0

	if newTitle == emptyString {
		newTitle = ""
	}

	if newExpression == emptyString {
		newExpression = ""
	}

	if newDescription == emptyString {
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

	if newName == emptyString {
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

func TeamsAppChannelsBlockEqual(oldSlice, newSlice []interface{}) bool {
	equalCount := 0

	if oldSlice == nil {
		oldSlice = make([]interface{}, 0)
	}

	if newSlice == nil {
		newSlice = make([]interface{}, 0)
	}

	var oldArray []map[string]interface{}
	var newArray []map[string]interface{}

	for _, val := range oldSlice {
		oldMap, err := val.(map[string]interface{})
		if err != true {
			panic(err)
		}
		oldArray = append(oldArray, oldMap)
	}

	for _, val := range newSlice {
		newMap, err := val.(map[string]interface{})
		if err != true {
			panic(err)
		}
		newArray = append(newArray, newMap)
	}

	if len(oldArray) == len(newArray) {
		for _, oldVal := range oldArray {
			for _, newVal := range newArray {
				if TeamsAppChannelsMapEqual(oldVal, newVal) {
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

func TeamsAppChannelsMapEqual(oldMap, newMap map[string]interface{}) bool {
	count := 2
	equalCount := 0

	if len(oldMap) == len(newMap) {
		for oldKey, oldVal := range oldMap {
			for newKey, newVal := range newMap {
				if strings.EqualFold(oldKey, newKey) {
					switch oldKey {
					case "team":
						if oldVal.(string) == newVal.(string) {
							equalCount++
						}
					case "channels":
						if ArrayOfInterfaceEqual(oldVal, newVal) {
							equalCount++
						}
					default:
						return false
					}
				}
			}
		}
		if equalCount != count {
			return false
		}
	} else {
		return false
	}

	return true
}

func DiffSuppressCommaSeparatedStrings(old, new string) bool {
	oldSlice := strings.Split(strings.TrimSpace(old), ",")
	newSlice := strings.Split(strings.TrimSpace(new), ",")

	for i := range oldSlice {
		oldSlice[i] = strings.TrimSpace(oldSlice[i])
	}
	for j := range newSlice {
		newSlice[j] = strings.TrimSpace(newSlice[j])
	}

	return SliceIgnoreOrderEqual(oldSlice, newSlice)
}

func ResourceLabelsMapEqual(oldMap, newMap map[string]interface{}) bool {
	equalCount := 0

	if len(oldMap) == len(newMap) {
		for oldKey, oldVal := range oldMap {
			for newKey, newVal := range newMap {
				if strings.EqualFold(oldKey, newKey) {
					if DiffSuppressCommaSeparatedStrings(oldVal.(string), newVal.(string)) {
						equalCount++
					}
				}
			}
		}
		if equalCount != len(newMap) {
			return false
		}
	} else {
		return false
	}

	return true
}
