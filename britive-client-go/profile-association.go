package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetProfileAssociationResource - Returns a all associations linked with profile
func (c *Client) GetProfileAssociationResource(profileID string, name string, parentName string) (*ProfileAssociationResource, error) {
	filter := fmt.Sprintf("name eq %s", name)
	profileAssociationResources, err := c.getProfileAssociationResource(profileID, filter)
	if err != nil {
		return nil, err
	}
	var profileAssociationResource *ProfileAssociationResource
	for _, p := range profileAssociationResources {
		if p.ParentName == parentName {
			profileAssociationResource = &p
			break
		}
	}
	if profileAssociationResource == nil {
		return nil, ErrNotFound
	}
	return profileAssociationResource, nil
}

// GetProfileAssociationResourceByNativeID - Returns a all associations linked with profile
func (c *Client) GetProfileAssociationResourceByNativeID(profileID string, nativeID string) (*ProfileAssociationResource, error) {
	filter := fmt.Sprintf(`nativeId eq "%s"`, nativeID)
	return c.getUniqueProfileAssociationResource(profileID, filter)
}

func (c *Client) getUniqueProfileAssociationResource(profileID string, filter string) (*ProfileAssociationResource, error) {
	profileAssociationResources, err := c.getProfileAssociationResource(profileID, filter)
	if err != nil {
		return nil, err
	}
	if len(profileAssociationResources) == 0 {
		return nil, ErrNotFound
	}
	return &profileAssociationResources[0], nil
}

func (c *Client) getProfileAssociationResource(profileID string, filter string) ([]ProfileAssociationResource, error) {
	endpoint := fmt.Sprintf("paps/%s/resources", profileID)
	profileAssociationResources := make([]ProfileAssociationResource, 0)
	err := client.NewQueryRequest().
		WithLock(profileID).
		WithFilter(filter).
		WithResult(&profileAssociationResources).
		Query(endpoint)

	return profileAssociationResources, err
}

// SaveProfileAssociationScopes - Save profile associations
func (c *Client) SaveProfileAssociationScopes(profileID string, associations []ProfileAssociation) error {
	utb, err := json.Marshal(associations)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/paps/%s/scopes", c.APIBaseURL, profileID), strings.NewReader(string(utb)))
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, profileID)

	return err
}

// SaveProfileAssociationResourceScopes - Save profile associations
func (c *Client) SaveProfileAssociationResourceScopes(profileID string, associations []ProfileAssociation) error {
	utb, err := json.Marshal(associations)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/paps/%s/resources/scopes", c.APIBaseURL, profileID), strings.NewReader(string(utb)))
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, profileID)

	return err
}

// Get the application type for a given application ID
func (c *Client) GetApplicationType(appContainerID string) (*ApplicationType, error) {
	resourceURL := fmt.Sprintf("%s/apps/%s", c.APIBaseURL, appContainerID)
	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if string(body) == emptyString {
		return nil, ErrNotFound
	}

	applicationType := &ApplicationType{}
	err = json.Unmarshal(body, applicationType)
	if err != nil {
		return nil, err
	}

	return applicationType, nil
}

// Get the environment id on passing accountId for AWS Standalone apps. Return empty string otherwise.
func (c *Client) GetEnvId(appContainerID string, accountId string) string {
	resourceURL := fmt.Sprintf("%s/apps/%s/envAccounts/%s", c.APIBaseURL, appContainerID, accountId)
	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		return emptyString
	}

	body, err := c.Do(req)
	if err != nil {
		return emptyString
	}

	if string(body) == emptyString {
		return emptyString
	}

	envAccId := &EnvAccId{}
	err = json.Unmarshal(body, envAccId)
	if err != nil {
		return emptyString
	}

	return envAccId.EnvironmentId
}
