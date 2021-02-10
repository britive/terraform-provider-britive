package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetProfileAssociationResources - Returns a all associations linked with profile
func (c *Client) GetProfileAssociationResources(profileID string) (*[]ProfileAssociationResource, error) {
	//TODO: Warning Recursion - Get by Filter
	requestURL := fmt.Sprintf("%s/paps/%s/resources", c.APIBaseURL, profileID)
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequestWithLock(req, profileID)
	if err != nil {
		return nil, err
	}
	profileAssociationResources := make([]ProfileAssociationResource, 0)
	err = json.Unmarshal(body, &profileAssociationResources)
	if err != nil {
		return nil, err
	}
	return &profileAssociationResources, nil
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

	_, err = c.doRequestWithLock(req, profileID)

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

	_, err = c.doRequestWithLock(req, profileID)

	return err
}
