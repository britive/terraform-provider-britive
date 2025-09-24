package britive

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// GetApplicationRootEnvironmentGroup - Returns root environment group
func (c *Client) GetApplicationRootEnvironmentGroup(appContainerID string, m interface{}) (*ApplicationRootEnvironmentGroup, error) {
	providerMeta := m.(*ProviderMeta)
	appCache := providerMeta.AppCache

	mutex := providerMeta.Mutex
	isLocked := false

	var appRootEnvironmentGroup *ApplicationRootEnvironmentGroup
	cacheKey := fmt.Sprintf("/apps/%s/root-environment-group", appContainerID)

	if _, ok := appCache[cacheKey]; !ok {
		mutex.Lock()
		isLocked = true
		log.Printf("=========== Cache miss appRootEnvironmentGroup")
	}
	if cacheData, ok := appCache[cacheKey]; ok {
		if isLocked {
			mutex.Unlock()
			isLocked = false
		}
		defer func() {
			if isLocked {
				mutex.Unlock()
				isLocked = false
			}
		}()
		appRootEnvironmentGroup = cacheData.Cache.(*ApplicationRootEnvironmentGroup)
		log.Printf("=========== Cache hit appRootEnvironmentGroup: %v", appRootEnvironmentGroup)
	} else {
		log.Printf("=========== Get GetApplicationRootEnvironmentGroup: %v", appContainerID)
		//TODO: Warning Recursion - Get by Filter
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/apps/%s/root-environment-group", c.APIBaseURL, appContainerID), nil)
		if err != nil {
			return nil, err
		}

		body, err := c.Do(req)
		if err != nil {
			return nil, err
		}

		appRootEnvironmentGroup := &ApplicationRootEnvironmentGroup{}
		err = json.Unmarshal(body, appRootEnvironmentGroup)
		if err != nil {
			return nil, err
		}

		newCache := &AppData{
			Cache: appRootEnvironmentGroup,
		}
		appCache[cacheKey] = newCache

		log.Printf("=========== Cached appRootEnvironmentGroup: %+v", newCache.Cache)
	}
	if isLocked {
		mutex.Unlock()
		isLocked = false
	}

	return appRootEnvironmentGroup, nil
}
