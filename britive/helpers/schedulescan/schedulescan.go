// Package schedulescan holds helpers shared between the provider's two scheduled-scan
// resources - britive_resource_manager_resource_type_schedule_scan (britive/resources/resourcemanager)
// and britive_application_scan_schedule (britive/resources) - which both configure tasks on
// the same underlying scan task-service API: frequencyType/frequencyInterval/startTime, with
// frequencyInterval meaning the ISO 8601 weekday (1=Monday...7=Sunday) for Weekly and the day
// of the month (1-31) for Monthly.
package schedulescan

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// WeekdayToInterval maps a day_of_week value (case-insensitive, full name or abbreviation)
// to the wire's frequencyInterval for Weekly schedules: Monday=1 ... Sunday=7 (ISO 8601
// weekday numbering).
var WeekdayToInterval = map[string]int{
	"monday": 1, "mon": 1,
	"tuesday": 2, "tue": 2,
	"wednesday": 3, "wed": 3,
	"thursday": 4, "thu": 4,
	"friday": 5, "fri": 5,
	"saturday": 6, "sat": 6,
	"sunday": 7, "sun": 7,
}

// IntervalToWeekday is WeekdayToInterval's inverse, used to render the wire's
// frequencyInterval back into a canonical day name on Read/Import.
var IntervalToWeekday = map[int]string{
	1: "Monday", 2: "Tuesday", 3: "Wednesday", 4: "Thursday", 5: "Friday", 6: "Saturday", 7: "Sunday",
}

// FormatStartTime renders the API's [hour, minute] pair as a "HH:MM" string.
func FormatStartTime(startTime []int) types.String {
	if len(startTime) != 2 {
		return types.StringNull()
	}
	return types.StringValue(fmt.Sprintf("%02d:%02d", startTime[0], startTime[1]))
}

// PreserveOptionalString uses prior state as fallback when the API returns "": if the prior
// state was null (the user never set the field), keep null; if it was "" or non-empty,
// preserve that intent. Avoids an inconsistent transition from empty string to null that an
// Optional-only string field would otherwise show when the API always returns "" for an
// unset value.
func PreserveOptionalString(apiValue string, priorState types.String) types.String {
	if apiValue == "" {
		if priorState.IsNull() {
			return types.StringNull()
		}
		return types.StringValue("")
	}
	return types.StringValue(apiValue)
}

// CanonicalCasing returns the member of canonicalValues that case-insensitively matches
// value, or value unchanged if none match. Used to normalize a case-insensitively-validated
// enum input (e.g. frequency_type, a scope type) to the exact casing the API expects, since
// the value echoed back by the API is otherwise not necessarily what the user typed.
func CanonicalCasing(value string, canonicalValues ...string) string {
	for _, c := range canonicalValues {
		if strings.EqualFold(value, c) {
			return c
		}
	}
	return value
}
