// Package applicationscan holds helpers shared between britive_application and
// britive_application_scan_schedule (both in britive/resources), which race the same backend
// behavior: an application's scan task service is registered automatically but asynchronously
// when the application itself is created.
package applicationscan

import (
	"time"

	"github.com/britive/terraform-provider-britive/britive-client-go"
)

// retryAttempts/retryBaseDelay govern GetTaskServiceWithRetry's retry schedule: retryAttempts
// total attempts (the initial try plus 3 retries), with an exponentially doubling delay before
// each retry starting at retryBaseDelay - i.e. 1s, 2s, 4s between attempts.
const (
	retryAttempts  = 3
	retryBaseDelay = 1 * time.Second
)

// GetTaskServiceWithRetry retries GetApplicationScanTaskService up to retryAttempts times
// total, with an exponentially doubling delay between attempts starting at retryBaseDelay (1s,
// then 2s, then 4s). The backend provisions an application's scan task service asynchronously
// after the application itself is created, so a lookup immediately afterward can transiently
// fail (observed: "E1004: Task service does not exist for service, tenant, and app") before it
// exists yet.
//
// Shared by both ApplicationScanScheduleResource.Create and ApplicationResource.Create, which
// hit the exact same race immediately after creating their respective application/schedule.
// ApplicationScanScheduleResource.Create has no orphaned object to fall back on if every retry
// is exhausted - CreateApplicationScanTask hasn't run yet, so there's nothing to persist to
// state, and a plain Terraform Create retry is a clean, side-effect-free retry of exactly the
// same work. ApplicationResource.Create does have one: the application itself was already
// created successfully, so on exhaustion it persists that application to state anyway (with
// task_service_id/scan_enabled left unresolved) rather than deleting it - see that function's
// own comment for why. Read/Update/Delete/Import on both resources call
// client.GetApplicationScanTaskService directly instead (via resolveTaskServiceID on
// ApplicationScanScheduleResource), since by then the task service is expected to already exist
// and a failure there is a real error worth surfacing immediately, not retrying.
func GetTaskServiceWithRetry(client *britive.Client, applicationID string) (*britive.ScheduleScanTaskService, error) {
	var lastErr error
	delay := retryBaseDelay
	for attempt := 1; attempt <= retryAttempts; attempt++ {
		taskService, err := client.GetApplicationScanTaskService(applicationID)
		if err == nil {
			return taskService, nil
		}
		lastErr = err
		if attempt < retryAttempts {
			time.Sleep(delay)
			delay *= 2
		}
	}
	return nil, lastErr
}
