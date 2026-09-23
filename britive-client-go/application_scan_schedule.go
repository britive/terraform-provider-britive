package britive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Scan schedule tasks for an application live under the application's scan task-service,
// which is registered automatically when the application itself is created and is looked
// up by name+appId - confirmed by capture, there is no dedicated sub-path lookup route for
// applications the way resource manager has (/tasks/services/resource-scan/resource-types/{id}).
// GetApplicationScanTaskService is expected to always succeed for any application this
// provider manages. Tasks are listed and filtered client-side on Read - no dedicated
// single-item GET route was ever exercised for applications in the capture this was
// designed from.

// GetApplicationScanTaskService retrieves an application's scan task service, including its
// taskServiceId and current enabled status.
func (c *Client) GetApplicationScanTaskService(applicationID string) (*ScheduleScanTaskService, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/tasks/services?name=environmentScanner&appId=%s", c.APIBaseURL, url.QueryEscape(applicationID)), nil)
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

// CreateApplicationScanTask creates a new scheduled scan task under an application's
// already-registered task service.
func (c *Client) CreateApplicationScanTask(taskServiceID string, task ApplicationScheduleScanTask) (*ApplicationScheduleScanTaskDetail, error) {
	body, err := json.Marshal(task)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/tasks/services/%s/tasks", c.APIBaseURL, taskServiceID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, applicationScanScheduleLockName)
	if err != nil {
		return nil, err
	}

	result := &ApplicationScheduleScanTaskDetail{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListApplicationScanTasks lists every scheduled scan task under an application's task
// service.
func (c *Client) ListApplicationScanTasks(taskServiceID string) ([]ApplicationScheduleScanTaskDetail, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/tasks/services/%s/tasks", c.APIBaseURL, taskServiceID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	var result []ApplicationScheduleScanTaskDetail
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetApplicationScanTask retrieves a single scheduled scan task by ID. No dedicated
// single-item GET route was confirmed for applications (unlike resource manager's, which
// now has one) - list and filter client-side instead.
func (c *Client) GetApplicationScanTask(taskServiceID string, taskID string) (*ApplicationScheduleScanTaskDetail, error) {
	tasks, err := c.ListApplicationScanTasks(taskServiceID)
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

// UpdateApplicationScanTask saves a scheduled scan task's full configuration. The API fully
// replaces name/startTime/frequencyType/frequencyInterval/properties on every call
// (confirmed by capture).
func (c *Client) UpdateApplicationScanTask(taskServiceID string, taskID string, task ApplicationScheduleScanTask) (*ApplicationScheduleScanTaskDetail, error) {
	body, err := json.Marshal(task)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/tasks/services/%s/tasks/%s", c.APIBaseURL, taskServiceID, taskID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	respBody, err := c.DoWithLock(req, applicationScanScheduleLockName)
	if err != nil {
		return nil, err
	}

	result := &ApplicationScheduleScanTaskDetail{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteApplicationScanTask deletes a scheduled scan task.
func (c *Client) DeleteApplicationScanTask(taskServiceID string, taskID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/tasks/services/%s/tasks/%s", c.APIBaseURL, taskServiceID, taskID), nil)
	if err != nil {
		return err
	}

	_, err = c.DoWithLock(req, applicationScanScheduleLockName)
	if errors.Is(err, ErrNoContent) || err == nil {
		return nil
	}
	return err
}

// EnableApplicationScanTaskService turns scanning on for an application's entire task
// service - application-wide, not scoped to any individual schedule task (confirmed by
// capture - empty request body).
func (c *Client) EnableApplicationScanTaskService(taskServiceID string) (*ScheduleScanTaskService, error) {
	return c.setApplicationScanTaskServiceEnabled(taskServiceID, "enabled-statuses")
}

// DisableApplicationScanTaskService turns scanning off for an application's entire task
// service - see EnableApplicationScanTaskService.
func (c *Client) DisableApplicationScanTaskService(taskServiceID string) (*ScheduleScanTaskService, error) {
	return c.setApplicationScanTaskServiceEnabled(taskServiceID, "disabled-statuses")
}

func (c *Client) setApplicationScanTaskServiceEnabled(taskServiceID string, statusPath string) (*ScheduleScanTaskService, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/tasks/services/%s/%s", c.APIBaseURL, taskServiceID, statusPath), strings.NewReader("{}"))
	if err != nil {
		return nil, err
	}

	body, err := c.DoWithLock(req, applicationScanScheduleLockName)
	if err != nil {
		return nil, err
	}

	result := &ScheduleScanTaskService{}
	if err := json.Unmarshal(body, result); err != nil {
		return nil, err
	}
	return result, nil
}
