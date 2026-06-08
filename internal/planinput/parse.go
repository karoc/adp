package planinput

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const Version = 1

type Document struct {
	Version int     `json:"version" yaml:"version"`
	Phases  []Phase `json:"phases" yaml:"phases"`
	Tasks   []Task  `json:"tasks" yaml:"tasks"`
}

type Phase struct {
	ID    string `json:"id" yaml:"id"`
	Title string `json:"title" yaml:"title"`
	Goal  string `json:"goal" yaml:"goal"`
}

type Task struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Priority    string `json:"priority" yaml:"priority"`
	Phase       string `json:"phase" yaml:"phase"`
	Status      string `json:"status" yaml:"status"`
}

var validTaskStatuses = map[string]struct{}{
	"planned":     {},
	"ready":       {},
	"in_progress": {},
	"blocked":     {},
	"review":      {},
	"validated":   {},
	"done":        {},
	"canceled":    {},
}

var validTaskPriorities = map[string]struct{}{
	"critical": {},
	"urgent":   {},
	"p0":       {},
	"high":     {},
	"p1":       {},
	"normal":   {},
	"medium":   {},
	"p2":       {},
	"low":      {},
	"p3":       {},
}

func Parse(data []byte) (Document, error) {
	if strings.TrimSpace(string(data)) == "" {
		return Document{}, errors.New("plan input is empty")
	}

	var doc Document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return Document{}, fmt.Errorf("parse plan input: %w", err)
	}

	trimDocument(&doc)

	if doc.Version != Version {
		return Document{}, fmt.Errorf("unsupported plan input version %d: expected %d", doc.Version, Version)
	}
	if len(doc.Phases) == 0 && len(doc.Tasks) == 0 {
		return Document{}, errors.New("plan input must include at least one phase or task")
	}
	if err := validatePhases(doc.Phases); err != nil {
		return Document{}, err
	}
	if err := validateTasks(doc.Tasks); err != nil {
		return Document{}, err
	}

	return doc, nil
}

func trimDocument(doc *Document) {
	for i := range doc.Phases {
		doc.Phases[i].ID = strings.TrimSpace(doc.Phases[i].ID)
		doc.Phases[i].Title = strings.TrimSpace(doc.Phases[i].Title)
		doc.Phases[i].Goal = strings.TrimSpace(doc.Phases[i].Goal)
	}
	for i := range doc.Tasks {
		doc.Tasks[i].Title = strings.TrimSpace(doc.Tasks[i].Title)
		doc.Tasks[i].Description = strings.TrimSpace(doc.Tasks[i].Description)
		doc.Tasks[i].Priority = strings.TrimSpace(doc.Tasks[i].Priority)
		doc.Tasks[i].Phase = strings.TrimSpace(doc.Tasks[i].Phase)
		doc.Tasks[i].Status = strings.TrimSpace(doc.Tasks[i].Status)
	}
}

func validatePhases(phases []Phase) error {
	seen := make(map[string]struct{}, len(phases))
	for i, phase := range phases {
		if phase.ID == "" {
			return fmt.Errorf("phase[%d].id is required", i)
		}
		if phase.Title == "" {
			return fmt.Errorf("phase[%d].title is required", i)
		}
		if _, ok := seen[phase.ID]; ok {
			return fmt.Errorf("duplicate phase id %q", phase.ID)
		}
		seen[phase.ID] = struct{}{}
	}
	return nil
}

func validateTasks(tasks []Task) error {
	for i, task := range tasks {
		if task.Title == "" {
			return fmt.Errorf("task[%d].title is required", i)
		}
		if task.Status != "" {
			if _, ok := validTaskStatuses[task.Status]; !ok {
				return fmt.Errorf("task[%d].status %q is invalid", i, task.Status)
			}
		}
		if task.Priority != "" {
			if _, ok := validTaskPriorities[task.Priority]; !ok {
				return fmt.Errorf("task[%d].priority %q is invalid", i, task.Priority)
			}
		}
	}
	return nil
}
