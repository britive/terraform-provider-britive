package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Schedule scan tasks live under a resource type's scan task-service. The task service is
// now registered automatically when the resource type itself is created (see
// britive.ResourceType.TaskServiceID) and deleted automatically when the resource type is
// deleted - there's no bootstrap/cleanup call for it in this file at all anymore, and
// GetScheduleScanTaskService is expected to always succeed for any resource type this
// provider manages. Tasks themselves are created via a dedicated POST directly under the
// task service, and (now that the route works - it used to 500) read back via a real
// single-item GET rather than listing and filtering.

// GetScheduleScanTaskService retrieves a resource type's scan task service, including its
// taskServiceId and current enabled status. Used to resolve taskServiceId for
// Read/Update/Delete/Import.
func (c *Client) GetScheduleScanTaskService(resourceTypeID string) (*ScheduleScanTaskService, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/tasks/services/resource-scan/resource-types/%s", c.APIBaseURL, resourceTypeID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	result := &ScheduleScanTaskService{}
	if err := json.Unmarshal(body, result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateScheduleScanTask creates a new scheduled scan task under an already-registered task
// service. The task service itself is never created here - the backend rejects a task in the
// body of the old bundled resource-types create call with 400 now; resource type creation is
// the only way a taskServiceId comes into existence.
func (c *Client) CreateScheduleScanTask(taskServiceID string, task ScheduleScanTask) (*ScheduleScanTaskDetail, error) {
	body, err := json.Marshal(task)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/tasks/services/resource-scan/%s/tasks", c.APIBaseURL, taskServiceID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, scheduleScanLockName)
	if err != nil {
		return nil, err
	}

	result := &ScheduleScanTaskDetail{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListScheduleScanTasks lists every scheduled scan task under a task service.
func (c *Client) ListScheduleScanTasks(taskServiceID string) ([]ScheduleScanTaskDetail, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/tasks/services/resource-scan/%s/tasks", c.APIBaseURL, taskServiceID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	var result []ScheduleScanTaskDetail
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetScheduleScanTask retrieves a single scheduled scan task by ID. This route used to
// return 500 (the backend team has since fixed it) - it's now confirmed to still respond
// with a JSON array (like the plain list endpoint), not a bare object, so this unmarshals
// as a list and picks out the matching taskID rather than assuming a single-object body.
func (c *Client) GetScheduleScanTask(taskServiceID string, taskID string) (*ScheduleScanTaskDetail, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/tasks/services/resource-scan/%s/tasks/%s", c.APIBaseURL, taskServiceID, taskID), nil)
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

	var tasks []ScheduleScanTaskDetail
	if err := json.Unmarshal(body, &tasks); err != nil {
		return nil, err
	}

	for i := range tasks {
		if tasks[i].TaskID == taskID {
			return &tasks[i], nil
		}
	}
	return nil, ErrNotFound
}

// UpdateScheduleScanTask saves a scheduled scan task's full configuration. The API fully
// replaces name/description/properties/frequencyType/frequencyInterval/startTime on every
// call (confirmed by capture - e.g. sending properties: {} clears every existing filter).
func (c *Client) UpdateScheduleScanTask(taskServiceID string, taskID string, task ScheduleScanTask) (*ScheduleScanTaskDetail, error) {
	body, err := json.Marshal(task)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/tasks/services/resource-scan/%s/tasks/%s", c.APIBaseURL, taskServiceID, taskID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, scheduleScanLockName)
	if err != nil {
		return nil, err
	}

	result := &ScheduleScanTaskDetail{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteScheduleScanTask deletes a scheduled scan task.
func (c *Client) DeleteScheduleScanTask(taskServiceID string, taskID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/tasks/services/resource-scan/%s/tasks/%s", c.APIBaseURL, taskServiceID, taskID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, scheduleScanLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}

// EnableScheduleScanTaskService turns scanning on for a resource type's entire task
// service - resource-type-wide, not scoped to any individual schedule task (confirmed by
// capture - empty request body).
func (c *Client) EnableScheduleScanTaskService(taskServiceID string) (*ScheduleScanTaskService, error) {
	return c.setScheduleScanTaskServiceEnabled(taskServiceID, "enabled-statuses")
}

// DisableScheduleScanTaskService turns scanning off for a resource type's entire task
// service - see EnableScheduleScanTaskService.
func (c *Client) DisableScheduleScanTaskService(taskServiceID string) (*ScheduleScanTaskService, error) {
	return c.setScheduleScanTaskServiceEnabled(taskServiceID, "disabled-statuses")
}

func (c *Client) setScheduleScanTaskServiceEnabled(taskServiceID string, statusPath string) (*ScheduleScanTaskService, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/tasks/services/resource-scan/%s/%s", c.APIBaseURL, taskServiceID, statusPath), strings.NewReader("{}"))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, scheduleScanLockName)
	if err != nil {
		return nil, err
	}

	result := &ScheduleScanTaskService{}
	if err := json.Unmarshal(body, result); err != nil {
		return nil, err
	}
	return result, nil
}
