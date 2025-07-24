package imports

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ImportHelper - Helper functions for terraform imports
type ImportHelper struct {
}

// NewImportHelper - Initializes new ImportHelper
func NewImportHelper() *ImportHelper {
	return &ImportHelper{}
}

// ParseImportID - Helper function to parse Import ID
func (ih *ImportHelper) ParseImportID(idRegexes []string, d *schema.ResourceData) error {
	for _, idFormat := range idRegexes {
		re, err := regexp.Compile(idFormat)
		if err != nil {
			return fmt.Errorf("invalid import format. %s", err)
		}

		if fieldValues := re.FindStringSubmatch(d.Id()); fieldValues != nil {
			for i := 1; i < len(fieldValues); i++ {
				fieldName := re.SubexpNames()[i]
				fieldValue := fieldValues[i]
				val, _ := d.GetOk(fieldName)
				if _, ok := val.(string); val == nil || ok {
					if fieldName == "id" {
						d.SetId(fieldValue)
					} else if err = d.Set(fieldName, fieldValue); err != nil {
						return err
					}
				} else if _, ok := val.(int); ok {
					if intVal, atoiErr := strconv.Atoi(fieldValue); atoiErr == nil {
						if err = d.Set(fieldName, intVal); err != nil {
							return err
						}
					} else {
						return fmt.Errorf("%s appears to be an integer, but %v cannot be parsed as an int", fieldName, fieldValue)
					}
				} else {
					return fmt.Errorf("cannot handle %s, which currently has value %v, and should be set to %#v, during import", fieldName, val, fieldValue)
				}
			}

			return nil
		}
	}
	return fmt.Errorf("import value %q doesn't match any of the accepted formats: %v", d.Id(), idRegexes)
}

// FetchImportFieldValue - Helper function to parse Import ID, and return the value for a given field
func (ih *ImportHelper) FetchImportFieldValue(idRegexes []string, d *schema.ResourceData, field string) (string, error) {
	for _, idFormat := range idRegexes {
		re, err := regexp.Compile(idFormat)
		if err != nil {
			return "", fmt.Errorf("invalid import format. %s", err)
		}

		if fieldValues := re.FindStringSubmatch(d.Id()); fieldValues != nil {
			for i := 1; i < len(fieldValues); i++ {
				fieldName := re.SubexpNames()[i]
				fieldValue := fieldValues[i]
				if strings.EqualFold(fieldName, field) {
					return fieldValue, nil
				}
			}

			return "", fmt.Errorf("Value not found for field %s", field)
		}
	}
	return "", fmt.Errorf("import value %q doesn't match any of the accepted formats: %v", d.Id(), idRegexes)
}
