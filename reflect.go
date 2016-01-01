package asana

import (
	"reflect"
	"strings"
)

var expandableType = reflect.TypeOf(expandable{})

// Populates expandable objects with a reference to the current client.
// Injection will stop when an expandable type with a client already set is
// found.
func injectClient(client *Client, object interface{}) {
	clientValue := reflect.ValueOf(client)
	value := reflect.ValueOf(object)

	injectClientValue(clientValue, value)
}

func injectClientValue(clientValue, value reflect.Value) {

	if value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface {
		if !value.IsNil() {
			injectClientValue(clientValue, value.Elem())
		}
		return
	}

	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		for j := 0; j < value.Len(); j++ {
			element := value.Index(j)
			if element.Kind() == reflect.Struct {
				injectClientValue(clientValue, element)
			}
		}
		return
	}

	if value.Kind() == reflect.Struct {
		if exp := value.FieldByName("expandable"); exp.IsValid() {
			if exp.Type() == expandableType {
				clientField := exp.FieldByName("Client")
				if !clientField.IsNil() {
					return
				}

				clientField.Set(clientValue)

				// Inject into child fields
				for i := 0; i < value.NumField(); i++ {
					injectClientValue(clientValue, value.Field(i))
				}
			}
		}
	}
	return
}

// Fields gets all valid JSON fields for a type
func Fields(i interface{}) *Options {
	t := reflect.TypeOf(i)
	if t.Kind() != reflect.Struct {
		panic("Invalid type requested")
	}

	result := &Options{}

	gatherFields(t, result)

	return result
}

func gatherFields(t reflect.Type, result *Options) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// Process a tag if present
		if jsonTag := f.Tag.Get("json"); jsonTag != "" {
			name := strings.Split(jsonTag, ",")[0]
			if name != "" && name != "-" {
				result.Fields = append(result.Fields, name)
			}
		}

		if f.Anonymous {
			gatherFields(f.Type, result)
		}
	}
}
