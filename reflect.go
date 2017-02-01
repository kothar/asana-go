package asana

import (
	"reflect"
	"strings"
)

var expandableType = reflect.TypeOf(Expandable{})

// Populates expandable objects with a reference to the current client.
// Object graph must not contain cycles, or an infinite loop will occur.
func (c *Client) inject(object interface{}) {
	clientValue := reflect.ValueOf(c)
	value := reflect.ValueOf(object)

	c.injectClientValue(clientValue, value)
}

func (c *Client) injectClientValue(clientValue, value reflect.Value) {
	if value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface {
		if !value.IsNil() {
			c.injectClientValue(clientValue, value.Elem())
		}
		return
	}

	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		for j := 0; j < value.Len(); j++ {
			element := value.Index(j)
			c.injectClientValue(clientValue, element)
		}
		return
	}

	if value.Kind() == reflect.Struct {
		// Inject into child fields
		for i := 0; i < value.NumField(); i++ {
			c.injectClientValue(clientValue, value.Field(i))
		}

		if value.Type() == expandableType {
			exp := value.Interface().(Expandable)
			exp.client = clientValue.Interface().(*Client)
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
