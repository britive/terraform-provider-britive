package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Schedule scan tasks live under a resource type's scan task-service, a singleton the API
// auto-creates the first time a task is created for that resource type (there's no separate
// bootstrap-only endpoint). GetScheduleScanTaskService distinguishes "not created yet" (400/E1004)
// from a real error via ErrScheduleScanTaskServiceNotBootstrapped. There's no single-item GET for
// a task - only a list - so GetScheduleScanTask lists and filters client-side.

// newScheduleScanTaskServiceStub returns the static taskService object every create-task call
// bundles alongside the task payload. Confirmed by capture: every field here is a hardcoded
// constant, identical across every create call regardless of resource type.
func newScheduleScanTaskServiceStub() ScheduleScanTaskServiceStub {
	return ScheduleScanTaskServiceStub{
		Name:    "ResourceScanner",
		Enabled: false,
		QueueID: "resourceScannerQueue",
	}
}

// GetScheduleScanTaskService retrieves a resource type's scan task-service record. Returns
// ErrScheduleScanTaskServiceNotBootstrapped (not a hard error) if no schedule scan task has
// ever been created for this resource type yet.
func (c *Client) GetScheduleScanTaskService(resourceTypeID string) (*ScheduleScanTaskService, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/tasks/services/resource-scan/resource-types/%s", c.APIBaseURL, resourceTypeID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "E1004") {
			return nil, ErrScheduleScanTaskServiceNotBootstrapped
		}
		return nil, err
	}

	result := &ScheduleScanTaskService{}
	if err := json.Unmarshal(body, result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateScheduleScanTask creates a new scheduled scan task for a resource type. Bundles the
// static taskService stub with the task payload, matching the API's combined create call -
// the taskService is created idempotently (a no-op after the first call for a resource type).
func (c *Client) CreateScheduleScanTask(resourceTypeID string, task ScheduleScanTask) (*ScheduleScanTaskDetail, error) {
	createReq := ScheduleScanTaskCreateRequest{
		TaskService: newScheduleScanTaskServiceStub(),
		Task:        task,
	}

	body, err := json.Marshal(createReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/tasks/services/resource-scan/resource-types/%s", c.APIBaseURL, resourceTypeID), strings.NewReader(string(body)))
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

// GetScheduleScanTask finds a single scheduled scan task by ID. There is no single-item GET
// endpoint for a task (confirmed by capture - only a list), so this lists and filters
// client-side. Returns ErrNotFound if no task with taskID exists in the list.
func (c *Client) GetScheduleScanTask(taskServiceID string, taskID string) (*ScheduleScanTaskDetail, error) {
	tasks, err := c.ListScheduleScanTasks(taskServiceID)
	if err != nil {
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
