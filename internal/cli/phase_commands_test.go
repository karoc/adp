package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	taskstore "github.com/karoc/adp/internal/tasks"
)

func TestPhaseListAndShowCommandsPrintJSON(t *testing.T) {
	phase := testPhase("p3", "Project planning", taskstore.PhaseStatusActive)
	phase.Goal = "phase gate smoke"
	store := &fakeTaskStore{phases: []taskstore.Phase{phase}}
	var listOut bytes.Buffer
	var listErr bytes.Buffer
	var showOut bytes.Buffer
	var showErr bytes.Buffer
	deps := Dependencies{
		WorkspaceStore:   &fakeStore{cfg: testConfig()},
		TaskStoreFactory: func(string) TaskStore { return store },
	}

	listCode := NewApp(deps, &listOut, &listErr).Execute(context.Background(), []string{"phase", "list", "--workspace", "game-a", "--format", "json"})
	showCode := NewApp(deps, &showOut, &showErr).Execute(context.Background(), []string{"phase", "show", "--workspace", "game-a", "p3", "--format", "json"})

	if listCode != 0 {
		t.Fatalf("phase list exit code = %d, stderr = %q", listCode, listErr.String())
	}
	if showCode != 0 {
		t.Fatalf("phase show exit code = %d, stderr = %q", showCode, showErr.String())
	}

	listPhase := findJSONObject(t, decodeJSONObjectList(t, listOut.Bytes(), "phases"), "id", "p3")
	assertJSONStringField(t, listPhase, "status", "active")
	assertJSONStringField(t, listPhase, "title", "Project planning")

	detail := decodeJSONObject(t, showOut.Bytes())
	assertJSONStringField(t, detail, "id", "p3")
	assertJSONStringField(t, detail, "status", "active")
	assertJSONStringField(t, detail, "goal", "phase gate smoke")
	assertJSONFieldAbsent(t, detail, "acceptance")
	assertJSONFieldAbsent(t, detail, "commit")
	assertJSONFieldAbsent(t, detail, "push")
}

func decodeJSONObject(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatalf("output is not a JSON object: %v\n%s", err, data)
	}
	return object
}

func decodeJSONObjectList(t *testing.T, data []byte, field string) []map[string]any {
	t.Helper()
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("output is not parseable JSON: %v\n%s", err, data)
	}
	switch value := raw.(type) {
	case []any:
		return objectsFromJSONArray(t, value, "top-level array")
	case map[string]any:
		return assertJSONObjectListField(t, value, field)
	default:
		t.Fatalf("JSON output is %T, want object or array", raw)
		return nil
	}
}

func findJSONObject(t *testing.T, objects []map[string]any, field string, want string) map[string]any {
	t.Helper()
	for _, object := range objects {
		if got, ok := object[field].(string); ok && got == want {
			return object
		}
	}
	t.Fatalf("missing JSON object with %s = %q in %#v", field, want, objects)
	return nil
}

func assertJSONObjectField(t *testing.T, object map[string]any, field string) map[string]any {
	t.Helper()
	value, ok := object[field]
	if !ok {
		t.Fatalf("JSON object missing field %q: %#v", field, object)
	}
	nested, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("JSON field %q = %T, want object", field, value)
	}
	return nested
}

func assertJSONObjectListField(t *testing.T, object map[string]any, field string) []map[string]any {
	t.Helper()
	value, ok := object[field]
	if !ok {
		t.Fatalf("JSON object missing field %q: %#v", field, object)
	}
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("JSON field %q = %T, want array", field, value)
	}
	return objectsFromJSONArray(t, items, field)
}

func objectsFromJSONArray(t *testing.T, items []any, label string) []map[string]any {
	t.Helper()
	objects := make([]map[string]any, 0, len(items))
	for i, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("%s item %d = %T, want object", label, i, item)
		}
		objects = append(objects, object)
	}
	return objects
}

func assertJSONStringField(t *testing.T, object map[string]any, field string, want string) {
	t.Helper()
	value, ok := object[field]
	if !ok {
		t.Fatalf("JSON object missing field %q: %#v", field, object)
	}
	got, ok := value.(string)
	if !ok {
		t.Fatalf("JSON field %q = %T, want string", field, value)
	}
	if got != want {
		t.Fatalf("JSON field %q = %q, want %q", field, got, want)
	}
}

func assertJSONNumberField(t *testing.T, object map[string]any, field string, want int) {
	t.Helper()
	value, ok := object[field]
	if !ok {
		t.Fatalf("JSON object missing field %q: %#v", field, object)
	}
	got, ok := value.(float64)
	if !ok {
		t.Fatalf("JSON field %q = %T, want number", field, value)
	}
	if got != float64(want) {
		t.Fatalf("JSON field %q = %s, want %d", field, fmt.Sprint(got), want)
	}
}

func assertJSONFieldAbsent(t *testing.T, object map[string]any, field string) {
	t.Helper()
	if _, ok := object[field]; ok {
		t.Fatalf("JSON object unexpectedly included field %q: %#v", field, object)
	}
}
