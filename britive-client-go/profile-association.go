package britive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetProfileAssociations - Returns a all associations linked with profile
func (c *Client) GetProfileAssociations(profileID string) (*[]ProfileAssociation, error) {
	requestURL := fmt.Sprintf("%s/paps/%s/scopes", c.APIBaseURL, profileID)
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequestWithLock(req, profileID)
	if err != nil {
		return nil, err
	}
	profileAssociations := make([]ProfileAssociation, 0)
	err = json.Unmarshal(body, &profileAssociations)
	if err != nil {
		return nil, err
	}
	return &profileAssociations, nil
}

// SaveProfileAssociations - Save profile associations
func (c *Client) SaveProfileAssociations(profileID string, associations []ProfileAssociation) error {
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
