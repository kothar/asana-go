package asana

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestStorySubtypeFields_DependencyTags covers a bug where Dependency and
// Dependent both carried the json tag "duplicated_from", colliding with
// DuplicatedFrom. encoding/json drops every field in a same-depth name
// collision, so all three were silently ignored on both marshal and unmarshal.
func TestStorySubtypeFields_DependencyTags(t *testing.T) {
	const payload = `{
		"duplicated_from": {"gid": "111"},
		"duplicate_of":    {"gid": "222"},
		"dependency":      {"gid": "333"},
		"dependent":       {"gid": "444"}
	}`

	var f StorySubtypeFields
	if err := json.Unmarshal([]byte(payload), &f); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		got  *Task
		want string
	}{
		{"DuplicatedFrom", f.DuplicatedFrom, "111"},
		{"DuplicateOf", f.DuplicateOf, "222"},
		{"Dependency", f.Dependency, "333"},
		{"Dependent", f.Dependent, "444"},
	} {
		if tc.got == nil {
			t.Errorf("%s was not populated; expected task %q", tc.name, tc.want)
			continue
		}
		if tc.got.ID != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got.ID, tc.want)
		}
	}
}

// TestStorySubtypeFields_DependencyTagsRoundTrip ensures the same fields
// survive marshalling. Under the collision every one of them was omitted.
func TestStorySubtypeFields_DependencyTagsRoundTrip(t *testing.T) {
	f := StorySubtypeFields{
		DuplicatedFrom: &Task{ID: "111"},
		Dependency:     &Task{ID: "333"},
		Dependent:      &Task{ID: "444"},
	}

	bs, err := json.Marshal(&f)
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"duplicated_from", "dependency", "dependent"} {
		if !strings.Contains(string(bs), `"`+key+`"`) {
			t.Errorf("marshalled output is missing %q: %s", key, bs)
		}
	}
}

// TestStoryJSONTagsAreUnique guards against reintroducing a duplicate tag.
// encoding/json resolves a same-depth name collision by discarding all of the
// conflicting fields, so this fails silently at runtime rather than loudly.
func TestStoryJSONTagsAreUnique(t *testing.T) {
	for _, target := range []struct {
		name string
		typ  reflect.Type
	}{
		{"Story", reflect.TypeOf(Story{})},
		{"StorySubtypeFields", reflect.TypeOf(StorySubtypeFields{})},
		{"StoryBase", reflect.TypeOf(StoryBase{})},
	} {
		// jsonName -> depth -> field names
		seen := map[string]map[int][]string{}

		var walk func(reflect.Type, int)
		walk = func(typ reflect.Type, depth int) {
			for i := 0; i < typ.NumField(); i++ {
				sf := typ.Field(i)

				// Embedded structs are promoted; their fields compete one level down.
				if sf.Anonymous && sf.Type.Kind() == reflect.Struct && sf.Tag.Get("json") == "" {
					walk(sf.Type, depth+1)
					continue
				}
				if !sf.IsExported() {
					continue
				}

				name := strings.Split(sf.Tag.Get("json"), ",")[0]
				if name == "-" || name == "" {
					continue
				}
				if seen[name] == nil {
					seen[name] = map[int][]string{}
				}
				seen[name][depth] = append(seen[name][depth], sf.Name)
			}
		}
		walk(target.typ, 0)

		for name, byDepth := range seen {
			for depth, fields := range byDepth {
				if len(fields) > 1 {
					t.Errorf("%s: json tag %q is used by %d fields at the same depth (%v); "+
						"encoding/json will silently drop all of them",
						target.name, name, len(fields), fields)
					_ = depth
				}
			}
		}
	}
}
