package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// GetSupportedConstraintTypes - Returns a set of supported constraint types for a given profile permission
func (c *Client) GetSupportedConstraintTypes(profileId, permissionName, permissionType string) ([]string, error) {
	resourceURL := fmt.Sprintf(`%s/paps/%s/permissions/%s/%s/supported-constraint-types`, c.APIBaseURL, profileId, permissionName, permissionType)
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

	supportedConstraintTypes := make([]string, 0)

	err = json.Unmarshal(body, &supportedConstraintTypes)
	if err != nil {
		return nil, err
	}

	if supportedConstraintTypes == nil {
		return nil, ErrNotFound
	}

	return supportedConstraintTypes, nil
}

// CreateConstraint - Add new permission constraint
func (c *Client) CreateConstraint(profileID, permissionName, permissionType, constraintType string, constraint Constraint) (*Constraint, error) {
	rc, err := json.Marshal(constraint)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%v/paps/%s/permissions/%s/%s/constraints/%s?operation=add", c.APIBaseURL, profileID, permissionName, permissionType, constraintType), strings.NewReader(string(rc)))
	if err != nil {
		return nil, err
	}

	_, err = c.DoWithLock(req, constraintLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &constraint, nil
	}

	return nil, err
}

// CreateConditionConstraint - Add new permission constraint of condition type
func (c *Client) CreateConditionConstraint(profileID, permissionName, permissionType, constraintType string, constraint ConditionConstraint) (*ConditionConstraint, error) {
	rc, err := json.Marshal(constraint)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%v/paps/%s/permissions/%s/%s/constraints/%s?operation=add", c.APIBaseURL, profileID, permissionName, permissionType, constraintType), strings.NewReader(string(rc)))
	if err != nil {
		return nil, err
	}

	_, err = c.DoWithLock(req, constraintLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &constraint, nil
	}

	return nil, err
}

// GetConstraint - Get permission constraint
func (c *Client) GetConstraint(profileID, permissionName, permissionType, constraintType string) (*ConstraintResult, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/paps/%s/permissions/%s/%s/constraints/%s", c.APIBaseURL, profileID, permissionName, permissionType, constraintType), nil)
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

	constraint := &ConstraintResult{}
	err = json.Unmarshal([]byte(body), &constraint)
	if err != nil {
		return nil, err
	}

	if constraint == nil {
		return nil, ErrNotFound
	}

	return constraint, nil
}

// GetConditionConstraint - Get permission constraint of condition type
func (c *Client) GetConditionConstraint(profileID, permissionName, permissionType, constraintType string) (*ConditionConstraintResult, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/paps/%s/permissions/%s/%s/constraints/%s", c.APIBaseURL, profileID, permissionName, permissionType, constraintType), nil)
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

	constraint := &ConditionConstraintResult{}
	err = json.Unmarshal([]byte(body), &constraint)
	if err != nil {
		return nil, err
	}

	if constraint == nil {
		return nil, ErrNotFound
	}

	return constraint, nil
}

// DeleteConstraint - Delete permission constraint
func (c *Client) DeleteConstraint(profileID, permissionName, permissionType, constraintType, constraintName string) error {
	if strings.EqualFold(constraintType, "condition") {
		co := ConditionConstraint{}
		co.Title = constraintName

		rc, err := json.Marshal(co)
		if err != nil {
			return err
		}

		req, err := http.NewRequest("PUT", fmt.Sprintf("%v/paps/%s/permissions/%s/%s/constraints/%s?operation=remove", c.APIBaseURL, profileID, permissionName, permissionType, constraintType), strings.NewReader(string(rc)))
		if err != nil {
			return err
		}

		_, err = c.DoWithLock(req, constraintLockName)
		if errors.Is(err, ErrNoContent) || err == nil {
			return nil
		}

		return err
	} else {
		co := Constraint{}
		co.Name = constraintName

		rc, err := json.Marshal(co)
		if err != nil {
			return err
		}

		req, err := http.NewRequest("PUT", fmt.Sprintf("%v/paps/%s/permissions/%s/%s/constraints/%s?operation=remove", c.APIBaseURL, profileID, permissionName, permissionType, constraintType), strings.NewReader(string(rc)))
		if err != nil {
			return err
		}

		_, err = c.DoWithLock(req, constraintLockName)
		if errors.Is(err, ErrNoContent) || err == nil {
			return nil
		}

		return err
	}
}
