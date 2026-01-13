package britive_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
)

// GetSupportedConstraintTypes - Returns a set of supported constraint types for a given profile permission
func (c *Client) GetSupportedConstraintTypes(ctx context.Context, profileId, permissionName, permissionType string) ([]string, error) {
	resourceURL := fmt.Sprintf(`%s/paps/%s/permissions/%s/%s/supported-constraint-types`, c.APIBaseURL, profileId, permissionName, permissionType)

	body, err := c.Get(ctx, resourceURL, ConstraintLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
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
func (c *Client) CreateConstraint(ctx context.Context, profileID, permissionName, permissionType, constraintType string, constraint Constraint) (*Constraint, error) {
	url := fmt.Sprintf("%v/paps/%s/permissions/%s/%s/constraints/%s?operation=add", c.APIBaseURL, profileID, permissionName, permissionType, constraintType)

	_, err := c.Put(ctx, url, constraint, ConstraintLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &constraint, nil
	}

	return nil, err
}

// CreateConditionConstraint - Add new permission constraint of condition type
func (c *Client) CreateConditionConstraint(ctx context.Context, profileID, permissionName, permissionType, constraintType string, constraint ConditionConstraint) (*ConditionConstraint, error) {
	url := fmt.Sprintf("%v/paps/%s/permissions/%s/%s/constraints/%s?operation=add", c.APIBaseURL, profileID, permissionName, permissionType, constraintType)

	log.Printf("====== permissionType: %s, \nurl: %s, constraint: %#v", permissionType, url, constraint)
	_, err := c.Put(ctx, url, constraint, ConstraintLockName)
	log.Printf("===== err : %#v", err)
	if errors.Is(err, ErrNoContent) || err == nil {
		return &constraint, nil
	}

	return nil, err
}

// GetConstraint - Get permission constraint
func (c *Client) GetConstraint(ctx context.Context, profileID, permissionName, permissionType, constraintType string) (*ConstraintResult, error) {
	url := fmt.Sprintf("%s/paps/%s/permissions/%s/%s/constraints/%s", c.APIBaseURL, profileID, permissionName, permissionType, constraintType)

	body, err := c.Get(ctx, url, ConstraintLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
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
func (c *Client) GetConditionConstraint(ctx context.Context, profileID, permissionName, permissionType, constraintType string) (*ConditionConstraintResult, error) {
	url := fmt.Sprintf("%s/paps/%s/permissions/%s/%s/constraints/%s", c.APIBaseURL, profileID, permissionName, permissionType, constraintType)
	body, err := c.Get(ctx, url, ConstraintLockName)
	if err != nil {
		return nil, err
	}

	if string(body) == EmptyString {
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
func (c *Client) DeleteConstraint(ctx context.Context, profileID, permissionName, permissionType, constraintType, constraintName string) error {
	if strings.EqualFold(constraintType, "condition") {
		co := ConditionConstraint{}
		co.Title = constraintName

		url := fmt.Sprintf("%v/paps/%s/permissions/%s/%s/constraints/%s?operation=remove", c.APIBaseURL, profileID, permissionName, permissionType, constraintType)

		_, err := c.Put(ctx, url, co, ConstraintLockName)
		if errors.Is(err, ErrNoContent) || err == nil {
			return nil
		}

		return err
	} else {
		co := Constraint{}
		co.Name = constraintName

		url := fmt.Sprintf("%v/paps/%s/permissions/%s/%s/constraints/%s?operation=remove", c.APIBaseURL, profileID, permissionName, permissionType, constraintType)

		_, err := c.Put(ctx, url, co, ConstraintLockName)
		if errors.Is(err, ErrNoContent) || err == nil {
			return nil
		}

		return err
	}
}
