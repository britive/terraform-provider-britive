package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Broker pools live under /resource-manager/remote-broker/pools. A pool owns its tokens and
// labels; brokers join a pool on their own (using one of its tokens) and resources are linked
// to it by britive_resource_manager_resource_broker_pools, so both are read-only here.
//
// Every write to a pool, or to anything under it, takes a per-pool lock, so parallel token or
// label changes on the same pool never race each other (a pool allows at most 3 tokens, with
// at most 2 active).

const (
	remoteBrokerPoolsPath = "resource-manager/remote-broker/pools"
	// remoteBrokerPageSize is the page size used for every paginated broker-pool list call.
	remoteBrokerPageSize = 100
	// remoteBrokerMaxPages guards the pagination loops against a server that keeps
	// returning the same continuation key.
	remoteBrokerMaxPages = 1000
)

// errCodePoolNotExist is the error code Britive returns (with HTTP 400, not 404) for a token or
// label call on a pool that does not exist: "E1004: pool does not exist".
const errCodePoolNotExist = "E1004"

// poolGone turns Britive's "pool does not exist" error into ErrNotFound, so callers handle a
// deleted pool like any other missing object. Other errors are returned unchanged.
func poolGone(err error) error {
	if err != nil && strings.HasPrefix(err.Error(), errCodePoolNotExist+": ") {
		return fmt.Errorf("%w: %s", ErrNotFound, err.Error())
	}
	return err
}

func brokerPoolLock(poolID string) string {
	return fmt.Sprintf("%s/%s", brokerPoolLockName, poolID)
}

func (c *Client) brokerPoolURL(poolID string, parts ...string) string {
	u := fmt.Sprintf("%s/%s/%s", c.APIBaseURL, remoteBrokerPoolsPath, url.PathEscape(poolID))
	for _, p := range parts {
		u += "/" + url.PathEscape(p)
	}
	return u
}

// ---------------------------------------------------------------------------
// Pools
// ---------------------------------------------------------------------------

// CreateBrokerPool creates a broker pool. The response carries the pool ID, name, timings and
// the value of the pool's automatically created "default" token, but not the description or
// the default token's status.
func (c *Client) CreateBrokerPool(request RemoteBrokerPoolCreateRequest) (*RemoteBrokerPool, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", c.APIBaseURL, remoteBrokerPoolsPath), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, brokerPoolLockName)
	if err != nil {
		return nil, err
	}

	result := &RemoteBrokerPool{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetBrokerPool returns one broker pool. Returns ErrNotFound when the pool does not exist.
func (c *Client) GetBrokerPool(poolID string) (*RemoteBrokerPool, error) {
	req, err := http.NewRequest("GET", c.brokerPoolURL(poolID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(body)) == emptyString {
		return nil, ErrNotFound
	}

	result := &RemoteBrokerPool{}
	if err := json.Unmarshal(body, result); err != nil {
		return nil, err
	}
	if result.PoolID == emptyString {
		return nil, ErrNotFound
	}
	return result, nil
}

// ListBrokerPools returns every broker pool whose name starts with namePrefix (all pools when
// namePrefix is empty), following the pagination key until the last page.
func (c *Client) ListBrokerPools(namePrefix string) ([]RemoteBrokerPool, error) {
	pools := make([]RemoteBrokerPool, 0)
	startKey := emptyString

	for i := 0; i < remoteBrokerMaxPages; i++ {
		query := url.Values{}
		query.Set("pageSize", strconv.Itoa(remoteBrokerPageSize))
		if namePrefix != emptyString {
			query.Set("name", namePrefix)
		}
		if startKey != emptyString {
			query.Set("exclusiveStartKey", startKey)
		}

		req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s?%s", c.APIBaseURL, remoteBrokerPoolsPath, query.Encode()), nil)
		if err != nil {
			return nil, err
		}

		body, err := c.Do(req)
		if errors.Is(err, ErrNotFound) {
			return pools, nil
		}
		if err != nil {
			return nil, err
		}

		page := RemoteBrokerPoolPage{}
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		pools = append(pools, page.Data...)

		if page.LastEvaluatedKey == emptyString || page.LastEvaluatedKey == startKey {
			return pools, nil
		}
		startKey = page.LastEvaluatedKey
	}
	return pools, nil
}

// UpdateBrokerPool replaces a broker pool's name, description and timings.
func (c *Client) UpdateBrokerPool(poolID string, request RemoteBrokerPoolUpdateRequest) (*RemoteBrokerPool, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", c.brokerPoolURL(poolID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, brokerPoolLock(poolID))
	if err != nil {
		return nil, err
	}

	result := &RemoteBrokerPool{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteBrokerPool deletes a broker pool. The API also deletes the pool's tokens, labels and
// resource links. A pool that is already gone is treated as deleted.
func (c *Client) DeleteBrokerPool(poolID string) error {
	req, err := http.NewRequest("DELETE", c.brokerPoolURL(poolID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, brokerPoolLock(poolID))
	if err == nil || errors.Is(err, ErrNoContent) || errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

// ---------------------------------------------------------------------------
// Tokens
// ---------------------------------------------------------------------------

// GetBrokerPoolTokens returns every token of a broker pool, including the "default" token.
// Returns ErrNotFound when the pool does not exist.
func (c *Client) GetBrokerPoolTokens(poolID string) ([]RemoteBrokerPoolToken, error) {
	req, err := http.NewRequest("GET", c.brokerPoolURL(poolID, "tokens"), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err = poolGone(err); err != nil {
		return nil, err
	}

	tokens := make([]RemoteBrokerPoolToken, 0)
	if strings.TrimSpace(string(body)) == emptyString {
		return tokens, nil
	}
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

// GetBrokerPoolToken returns one token of a broker pool by name. Returns ErrNotFound when the
// pool or the token does not exist.
func (c *Client) GetBrokerPoolToken(poolID, tokenName string) (*RemoteBrokerPoolToken, error) {
	tokens, err := c.GetBrokerPoolTokens(poolID)
	if err != nil {
		return nil, err
	}
	for i := range tokens {
		if tokens[i].TokenName == tokenName {
			return &tokens[i], nil
		}
	}
	return nil, ErrNotFound
}

// CreateBrokerPoolToken creates a token in a broker pool. The API always creates new tokens
// with status INACTIVE.
func (c *Client) CreateBrokerPoolToken(poolID string, request RemoteBrokerPoolTokenCreateRequest) (*RemoteBrokerPoolToken, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.brokerPoolURL(poolID, "tokens"), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, brokerPoolLock(poolID))
	if err != nil {
		return nil, err
	}

	result := &RemoteBrokerPoolToken{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateBrokerPoolToken changes a token's status and/or description. This works for the
// pool's "default" token too.
func (c *Client) UpdateBrokerPoolToken(poolID, tokenName string, request RemoteBrokerPoolTokenUpdateRequest) (*RemoteBrokerPoolToken, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", c.brokerPoolURL(poolID, "tokens", tokenName), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, brokerPoolLock(poolID))
	if err != nil {
		return nil, err
	}

	result := &RemoteBrokerPoolToken{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteBrokerPoolToken deletes a token. A token (or pool) that is already gone is treated as
// deleted, because deleting a pool also deletes its tokens.
func (c *Client) DeleteBrokerPoolToken(poolID, tokenName string) error {
	req, err := http.NewRequest("DELETE", c.brokerPoolURL(poolID, "tokens", tokenName), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, brokerPoolLock(poolID))
	err = poolGone(err)
	if err == nil || errors.Is(err, ErrNoContent) || errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

// ---------------------------------------------------------------------------
// Labels
// ---------------------------------------------------------------------------

// GetBrokerPoolLabels returns every label assigned to a broker pool. Returns ErrNotFound when
// the pool does not exist.
func (c *Client) GetBrokerPoolLabels(poolID string) ([]RemoteBrokerPoolLabel, error) {
	req, err := http.NewRequest("GET", c.brokerPoolURL(poolID, "labels"), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err = poolGone(err); err != nil {
		return nil, err
	}

	labels := make([]RemoteBrokerPoolLabel, 0)
	if strings.TrimSpace(string(body)) == emptyString {
		return labels, nil
	}
	if err := json.Unmarshal(body, &labels); err != nil {
		return nil, err
	}
	return labels, nil
}

// CreateBrokerPoolLabel adds a label key, with its values, to a broker pool.
func (c *Client) CreateBrokerPoolLabel(poolID string, label RemoteBrokerPoolLabel) error {
	if label.LabelValues == nil {
		label.LabelValues = []string{}
	}
	body, err := json.Marshal(label)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.brokerPoolURL(poolID, "labels"), strings.NewReader(string(body)))
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, brokerPoolLock(poolID))
	return err
}

// UpdateBrokerPoolLabel replaces the values of a label key on a broker pool. An empty list is
// accepted and leaves the key with no values.
func (c *Client) UpdateBrokerPoolLabel(poolID, key string, values []string) error {
	if values == nil {
		values = []string{}
	}
	body, err := json.Marshal(RemoteBrokerPoolLabel{LabelValues: values})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", c.brokerPoolURL(poolID, "labels", key), strings.NewReader(string(body)))
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, brokerPoolLock(poolID))
	return err
}

// DeleteBrokerPoolLabel removes a label key from a broker pool.
func (c *Client) DeleteBrokerPoolLabel(poolID, key string) error {
	req, err := http.NewRequest("DELETE", c.brokerPoolURL(poolID, "labels", key), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, brokerPoolLock(poolID))
	err = poolGone(err)
	if err == nil || errors.Is(err, ErrNoContent) || errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

// ---------------------------------------------------------------------------
// Brokers and linked resources (read-only)
// ---------------------------------------------------------------------------

// GetBrokerPoolActiveBrokers returns the active brokers of a broker pool, following the
// pagination key until the last page.
func (c *Client) GetBrokerPoolActiveBrokers(poolID string) ([]RemoteBroker, error) {
	brokers := make([]RemoteBroker, 0)
	startKey := emptyString

	for i := 0; i < remoteBrokerMaxPages; i++ {
		query := url.Values{}
		query.Set("status", "active")
		query.Set("pageSize", strconv.Itoa(remoteBrokerPageSize))
		if startKey != emptyString {
			query.Set("exclusiveStartKey", startKey)
		}

		req, err := http.NewRequest("GET", fmt.Sprintf("%s?%s", c.brokerPoolURL(poolID, "brokers", "paginated"), query.Encode()), nil)
		if err != nil {
			return nil, err
		}

		body, err := c.Do(req)
		if err != nil {
			return nil, err
		}

		page := RemoteBrokerPage{}
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		brokers = append(brokers, page.Data...)

		if page.LastEvaluatedKey == emptyString || page.LastEvaluatedKey == startKey {
			return brokers, nil
		}
		startKey = page.LastEvaluatedKey
	}
	return brokers, nil
}

// GetBrokerPoolLinkedResources returns the resources linked to a broker pool, reading every
// page of the resources query.
func (c *Client) GetBrokerPoolLinkedResources(poolID string) ([]BrokerPoolLinkedResource, error) {
	resources := make([]BrokerPoolLinkedResource, 0)

	for page := 0; page < remoteBrokerMaxPages; page++ {
		query := url.Values{}
		query.Set("filter", fmt.Sprintf("brokerPool eq %s", poolID))
		query.Set("page", strconv.Itoa(page))
		query.Set("size", strconv.Itoa(remoteBrokerPageSize))

		req, err := http.NewRequest("GET", fmt.Sprintf("%s/resource-manager/resources?%s", c.APIBaseURL, query.Encode()), nil)
		if err != nil {
			return nil, err
		}

		body, err := c.Do(req)
		if err != nil {
			return nil, err
		}

		result := BrokerPoolLinkedResourcePage{}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}
		resources = append(resources, result.Data...)

		if len(result.Data) == 0 || len(resources) >= result.Count {
			return resources, nil
		}
	}
	return resources, nil
}
