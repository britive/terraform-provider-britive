package britive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

var (
	syncOnce     sync.Once
	client       *Client
	retryWaitMin = 10 * time.Second  // minimum wait between retries (used only when the server gives no Retry-After)
	retryWaitMax = 480 * time.Second // maximum wait between retries (8 min)
)

const (
	// retryAfterFloor is the sanity-only floor applied when the server sends Retry-After.
	// Unlike retryWaitMin, it does not override a server-provided value that's genuinely short.
	retryAfterFloor = 1 * time.Second
	// retryAfterGrowth mildly escalates the server's own Retry-After value on repeated 429s
	// (rather than the previous 2^attempt scaling, which could blow a 60s hint up to 10 minutes by the 5th retry).
	retryAfterGrowth = 1.5
	// backoffJitterFrac spreads out concurrent retries (e.g. many resources created via `count`)
	// so they don't all wake up and collide again at the exact same instant.
	backoffJitterFrac = 0.2
)

// Client - Britive API client
type Client struct {
	APIBaseURL  string
	RetryClient *retryablehttp.Client
	Token       string
	Version     string
	SyncMap     *sync.Map
}

// NewClient - Initializes new Britive API client
func NewClient(apiBaseURL, token, version string, maxRetries int) (*Client, error) {
	syncOnce.Do(func() {
		if maxRetries <= 0 {
			maxRetries = defaultMaxRetries
		}

		rc := retryablehttp.NewClient()
		rc.RetryMax = maxRetries
		rc.RetryWaitMin = retryWaitMin
		rc.RetryWaitMax = retryWaitMax
		rc.Logger = nil
		rc.CheckRetry = only429CheckRetry
		rc.Backoff = func(min, max time.Duration, attempt int, resp *http.Response) time.Duration {
			if resp == nil || resp.StatusCode != http.StatusTooManyRequests {
				return retryablehttp.DefaultBackoff(min, max, attempt, resp)
			}

			method, rawURL := "", ""
			if resp.Request != nil {
				method = resp.Request.Method
				rawURL = resp.Request.URL.String()
			}

			var wait time.Duration
			source := "exponential backoff"
			effectiveMax := max

			// Retry-After present: trust the server's own value, escalating mildly (x1.5 per
			// repeated 429) rather than exponentially, so a volatile header doesn't get blown
			// out of proportion. Only a 1s sanity floor applies here — not the generic
			// retryWaitMin — since a short server-given value should be honored, not inflated.
			// If the server's raw ask itself exceeds our configured ceiling, honor it outright
			// on this attempt — the ceiling caps our own escalation, never the server's literal
			// request.
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
					if rawWait := time.Duration(secs) * time.Second; rawWait > effectiveMax {
						effectiveMax = rawWait
					}
					scaled := float64(secs) * math.Pow(retryAfterGrowth, float64(attempt))
					wait = time.Duration(scaled * float64(time.Second))
					if wait < retryAfterFloor {
						wait = retryAfterFloor
					} else if wait > effectiveMax {
						wait = effectiveMax
					}
					source = fmt.Sprintf("Retry-After=%ss * %.1f^%d", ra, retryAfterGrowth, attempt)
				}
			}

			if wait == 0 {
				wait = retryablehttp.DefaultBackoff(min, max, attempt, resp)
			}

			// Jitter avoids many concurrently-throttled requests (e.g. resources created via
			// `count`) all waking up and colliding at the same instant again.
			jittered := addJitter(wait, backoffJitterFrac)
			if jittered > effectiveMax {
				jittered = effectiveMax
			}

			log.Printf("[WARN] britive: HTTP 429 on %s %s — attempt %d/%d, retrying after %s (%s, jittered from %s)",
				method, rawURL, attempt+1, maxRetries+1, jittered, source, wait)
			return jittered
		}
		rc.HTTPClient = &http.Client{Timeout: 0}

		client = &Client{
			RetryClient: rc,
			APIBaseURL:  apiBaseURL,
			Token:       token,
			Version:     version,
			SyncMap:     &sync.Map{},
		}
	})
	return client, nil
}

// only429CheckRetry retries only on HTTP 429 (Too Many Requests).
func only429CheckRetry(_ context.Context, resp *http.Response, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}
	return false, nil
}

// addJitter randomizes d by +/- frac to keep concurrently-throttled requests from retrying
// in lockstep. frac is a fraction of d (e.g. 0.2 == +/-20%).
func addJitter(d time.Duration, frac float64) time.Duration {
	if d <= 0 {
		return d
	}
	delta := float64(d) * frac
	offset := (rand.Float64()*2 - 1) * delta // uniform in [-delta, +delta]
	jittered := time.Duration(float64(d) + offset)
	if jittered < 0 {
		jittered = 0
	}
	return jittered
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

// Do - Perform Britive API call; retries on HTTP 429 are handled by RetryClient.
func (c *Client) Do(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", fmt.Sprintf("TOKEN %s", c.Token))
	req.Header.Set("Content-Type", "application/json")
	userAgent := fmt.Sprintf("britive-client-go/%s golang/%s %s/%s britive-terraform/%s", c.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH, c.Version)
	req.Header.Add("User-Agent", userAgent)

	retryReq, err := retryablehttp.FromRequest(req)
	if err != nil {
		return nil, err
	}

	res, err := c.RetryClient.Do(retryReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNoContent {
		return []byte(emptyString), ErrNoContent
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
		if err == nil && httpErrorResponse.Message != emptyString {
			return nil, fmt.Errorf("%s: %s", httpErrorResponse.ErrorCode, httpErrorResponse.Message)
		}
		return nil, fmt.Errorf("an error occurred while processing the request\nrequest url: %s\nrequest method: %s\nresponse status: %d\nresponse body: %s", req.URL, req.Method, res.StatusCode, body)
	}

	return body, nil
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

// stripEmptyValues removes keys whose value is nil or an empty array from each
// map in the given slice. The backend API for some endpoints (e.g. policy
// permissions) echoes back fields such as "permissionScopes" as an empty list
// even when the caller never set them, which would otherwise cause a
// spurious diff against configuration that simply omits the field.
func stripEmptyValues(maps []map[string]interface{}) []map[string]interface{} {
	for _, m := range maps {
		for k, v := range m {
			if v == nil {
				delete(m, k)
				continue
			}
			if arr, ok := v.([]interface{}); ok && len(arr) == 0 {
				delete(m, k)
			}
		}
	}
	return maps
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

	oldArray = stripEmptyValues(oldArray)
	newArray = stripEmptyValues(newArray)

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
