package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Access builder settings are a per-application record under
// /apps/{appId}/access-request-settings. Association approvers are created and updated through
// their own routes, but are read from, and removed through, the settings record - the API has
// no dedicated read or delete route for a single association approver. Approvers groups have
// their own create/read/update/delete routes.

// accessBuilderSettingsNotFoundCode is the error code the API returns (with HTTP 400) when an
// application has no access builder settings record yet.
const accessBuilderSettingsNotFoundCode = "E1001"

// accessBuilderSettingsExistsMessage is part of the error message the API returns when
// creating a settings record that already exists.
const accessBuilderSettingsExistsMessage = "already exist"

// approversGroupNotFoundCode is the error code the API returns (with HTTP 400) for an approvers
// group ID that doesn't exist.
const approversGroupNotFoundCode = "E1004"

// FlexibleInt64 is an integer the API may return as a JSON number, a numeric string, or null.
// profileExpirationTimeout is sent as a number but returned as a string.
type FlexibleInt64 struct {
	Value int64
	Valid bool
}

// UnmarshalJSON accepts a JSON number, a numeric string, or null.
func (f *FlexibleInt64) UnmarshalJSON(data []byte) error {
	s := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if s == "" || s == "null" {
		*f = FlexibleInt64{}
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*f = FlexibleInt64{Value: v, Valid: true}
	return nil
}

// GetAccessBuilderSettings returns an application's access builder settings, or ErrNotFound
// when the application has no settings record yet.
func (c *Client) GetAccessBuilderSettings(applicationID string) (*AccessBuilderSettings, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/apps/%s/access-request-settings", c.APIBaseURL, applicationID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		if strings.HasPrefix(err.Error(), accessBuilderSettingsNotFoundCode) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	settings := &AccessBuilderSettings{}
	if err := json.Unmarshal(body, settings); err != nil {
		return nil, err
	}
	return settings, nil
}

// CreateAccessBuilderSettings creates an application's access builder settings record. An
// existing record is not an error: the first association approver created for an application
// also creates the record, so callers adopt it instead.
func (c *Client) CreateAccessBuilderSettings(applicationID string, managerApproval AccessRequestManagerApproval) error {
	body, err := json.Marshal(map[string]interface{}{"managerApproval": managerApproval})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/apps/%s/access-request-settings", c.APIBaseURL, applicationID), strings.NewReader(string(body)))
	if err != nil {
		return err
	}

	_, err = c.Do(req)
	if err != nil && strings.Contains(err.Error(), accessBuilderSettingsExistsMessage) {
		return nil
	}
	return err
}

// PatchAccessBuilderSettings updates only the fields set on patch.
func (c *Client) PatchAccessBuilderSettings(applicationID string, patch AccessBuilderSettingsPatch) error {
	body, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/apps/%s/access-request-settings", c.APIBaseURL, applicationID), strings.NewReader(string(body)))
	if err != nil {
		return err
	}

	_, err = c.Do(req)
	if errors.Is(err, ErrNoContent) {
		return nil
	}
	return err
}

// CreateApproversGroup creates an approvers group under an application.
func (c *Client) CreateApproversGroup(applicationID string, group ApproversGroup) (*ApproversGroup, error) {
	body, err := json.Marshal(group)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/apps/%s/approvers-groups", c.APIBaseURL, applicationID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	result := &ApproversGroup{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetApproversGroup returns a single approvers group, or ErrNotFound when it doesn't exist.
func (c *Client) GetApproversGroup(applicationID string, groupID string) (*ApproversGroup, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/apps/%s/approvers-groups/%s", c.APIBaseURL, applicationID, groupID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		if strings.HasPrefix(err.Error(), approversGroupNotFoundCode) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	result := &ApproversGroup{}
	if err := json.Unmarshal(body, result); err != nil {
		return nil, err
	}
	if result.ID == "" {
		return nil, ErrNotFound
	}
	return result, nil
}

// PatchApproversGroup updates only the fields set on patch. members, when set, replaces the
// whole member list.
func (c *Client) PatchApproversGroup(applicationID string, groupID string, patch ApproversGroupPatch) error {
	body, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/apps/%s/approvers-groups/%s", c.APIBaseURL, applicationID, groupID), strings.NewReader(string(body)))
	if err != nil {
		return err
	}

	_, err = c.Do(req)
	if errors.Is(err, ErrNoContent) {
		return nil
	}
	return err
}

// DeleteApproversGroup deletes an approvers group. The API rejects this while an association
// approver still uses the group.
func (c *Client) DeleteApproversGroup(applicationID string, groupID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/apps/%s/approvers-groups/%s", c.APIBaseURL, applicationID, groupID), nil)
	if err != nil {
		return err
	}

	_, err = c.Do(req)
	if errors.Is(err, ErrNoContent) || errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

// CreateAssociationApprover creates an association approver. Creating the first association
// approver for an application also creates its access builder settings record.
func (c *Client) CreateAssociationApprover(applicationID string, approver AssociationApprover) (*AssociationApprover, error) {
	if approver.AssociationTags == nil {
		approver.AssociationTags = []interface{}{}
	}
	body, err := json.Marshal(approver)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/apps/%s/association-approvers", c.APIBaseURL, applicationID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	result := &AssociationApprover{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// PatchAssociationApprover updates only the fields set on patch. associations and
// approversGroups, when set, replace the whole list.
func (c *Client) PatchAssociationApprover(applicationID string, approverID string, patch AssociationApproverPatch) error {
	if patch.AssociationTags == nil {
		patch.AssociationTags = []interface{}{}
	}
	body, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/apps/%s/association-approvers/%s", c.APIBaseURL, applicationID, approverID), strings.NewReader(string(body)))
	if err != nil {
		return err
	}

	_, err = c.Do(req)
	if errors.Is(err, ErrNoContent) {
		return nil
	}
	return err
}

// ListNotificationMediums lists every notification medium in the tenant.
func (c *Client) ListNotificationMediums() ([]NotificationMedium, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/notification-service/notificationmediums", c.APIBaseURL), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	result := &NotificationMediumList{}
	if err := json.Unmarshal(body, result); err != nil {
		return nil, err
	}
	return result.Result, nil
}

// GetNotificationMediumByName returns the notification medium with the given name.
func (c *Client) GetNotificationMediumByName(name string) (*NotificationMedium, error) {
	mediums, err := c.ListNotificationMediums()
	if err != nil {
		return nil, err
	}

	for i := range mediums {
		if mediums[i].Name == name {
			return &mediums[i], nil
		}
	}
	return nil, ErrNotFound
}
