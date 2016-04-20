//go:generate go-bindata -pkg schema data

package schema

import (
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

type portsFormatChecker struct{}

func (checker portsFormatChecker) IsFormat(input string) bool {
	return true
}

func init() {
	gojsonschema.FormatCheckers.Add("expose", portsFormatChecker{})
	gojsonschema.FormatCheckers.Add("ports", portsFormatChecker{})
}

func Validate(config map[string]interface{}) error {
	schemaData, err := Asset("data/config_schema_v2.1.json")
	if err != nil {
		return err
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaData))
	dataLoader := gojsonschema.NewGoLoader(config)

	result, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		return fmt.Errorf("config is invalid: %#v", result.Errors())
	}

	return nil
}
