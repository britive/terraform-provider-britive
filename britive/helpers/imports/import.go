package imports

import (
	"fmt"
	"regexp"
)

// ImportHelper - Helper functions for terraform imports
type ImportHelper struct {
}

// NewImportHelper - Initializes new ImportHelper
func NewImportHelper() *ImportHelper {
	return &ImportHelper{}
}

type ImportHelperData struct {
	ID     string
	Fields map[string]string
}

func (ih *ImportHelper) ParseImportID(idRegexes []string, data *ImportHelperData) error {
	for _, idFormat := range idRegexes {
		re, err := regexp.Compile(idFormat)
		if err != nil {
			return err
		}
		if matches := re.FindStringSubmatch(data.ID); matches != nil {
			if data.Fields == nil {
				data.Fields = make(map[string]string)
			}
			for i, name := range re.SubexpNames() {
				if i != 0 && name != "" {
					data.Fields[name] = matches[i]
				}
			}
			return nil
		}
	}
	return fmt.Errorf("import ID %q does not match expected formats", data.ID)
}

// // FetchImportFieldValue - Helper function to parse Import ID, and return the value for a given field
// func (ih *ImportHelper) FetchImportFieldValue(idRegexes []string, d *schema.ResourceData, field string) (string, error) {
// 	for _, idFormat := range idRegexes {
// 		re, err := regexp.Compile(idFormat)
// 		if err != nil {
// 			return "", fmt.Errorf("invalid import format. %s", err)
// 		}

// 		if fieldValues := re.FindStringSubmatch(d.Id()); fieldValues != nil {
// 			for i := 1; i < len(fieldValues); i++ {
// 				fieldName := re.SubexpNames()[i]
// 				fieldValue := fieldValues[i]
// 				if strings.EqualFold(fieldName, field) {
// 					return fieldValue, nil
// 				}
// 			}

// 			return "", fmt.Errorf("Value not found for field %s", field)
// 		}
// 	}
// 	return "", fmt.Errorf("import value %q doesn't match any of the accepted formats: %v", d.Id(), idRegexes)
// }
