package planinput

import (
	"strings"
	"testing"
)

// FuzzParse checks that Parse never panics on arbitrary bytes and that every
// Document it accepts satisfies the invariants the rest of ADP relies on:
// version is the supported one, at least one phase or task is present, all
// string fields are trimmed, phase IDs are unique, and any non-empty status or
// priority is a recognized value. A parser that returns a Document violating
// these would let malformed plan state past the intake boundary.
func FuzzParse(f *testing.F) {
	seeds := []string{
		"",
		"   \n\t ",
		"version: 1\ntasks:\n  - title: hello\n",
		"version: 1\nphases:\n  - id: p1\n    title: Setup\n",
		"version: 2\ntasks:\n  - title: x\n",
		"version: 1\n",
		"version: 1\ntasks:\n  - title: t\n    status: done\n    priority: high\n",
		"version: 1\ntasks:\n  - title: t\n    status: bogus\n",
		"version: 1\nphases:\n  - id: dup\n    title: A\n  - id: dup\n    title: B\n",
		"version: 1\ntasks:\n  - title: \"  spaced  \"\n",
		"not: valid: yaml: [",
		"version: 1\nphases: not-a-list\n",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		doc, err := Parse(data)
		if err != nil {
			// A rejected input must not also return a populated document that a
			// caller might mistakenly use.
			if doc.Version != 0 || len(doc.Phases) != 0 || len(doc.Tasks) != 0 {
				t.Fatalf("error path returned non-zero document: %+v", doc)
			}
			return
		}

		// Accepted documents must satisfy every invariant Parse claims to enforce.
		if doc.Version != Version {
			t.Fatalf("accepted document has version %d, want %d", doc.Version, Version)
		}
		if len(doc.Phases) == 0 && len(doc.Tasks) == 0 {
			t.Fatal("accepted document has neither phases nor tasks")
		}

		seen := make(map[string]struct{}, len(doc.Phases))
		for i, p := range doc.Phases {
			if p.ID == "" {
				t.Fatalf("phase[%d] accepted with empty id", i)
			}
			if p.Title == "" {
				t.Fatalf("phase[%d] accepted with empty title", i)
			}
			if p.ID != strings.TrimSpace(p.ID) {
				t.Fatalf("phase[%d].id %q not trimmed", i, p.ID)
			}
			if p.Title != strings.TrimSpace(p.Title) {
				t.Fatalf("phase[%d].title %q not trimmed", i, p.Title)
			}
			if p.Goal != strings.TrimSpace(p.Goal) {
				t.Fatalf("phase[%d].goal %q not trimmed", i, p.Goal)
			}
			if _, dup := seen[p.ID]; dup {
				t.Fatalf("duplicate phase id %q accepted", p.ID)
			}
			seen[p.ID] = struct{}{}
		}

		for i, task := range doc.Tasks {
			if task.Title == "" {
				t.Fatalf("task[%d] accepted with empty title", i)
			}
			if task.Title != strings.TrimSpace(task.Title) {
				t.Fatalf("task[%d].title %q not trimmed", i, task.Title)
			}
			if task.Status != "" {
				if _, ok := validTaskStatuses[task.Status]; !ok {
					t.Fatalf("task[%d] accepted with invalid status %q", i, task.Status)
				}
			}
			if task.Priority != "" {
				if _, ok := validTaskPriorities[task.Priority]; !ok {
					t.Fatalf("task[%d] accepted with invalid priority %q", i, task.Priority)
				}
			}
		}
	})
}
