package projectapi

import (
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"studio-engine/internal/apperr"
)

// Go-only routes (ADR-0135): Undo/Redo over the project history, and starting a new project
// or closing the open one. Each change is announced with project_changed like any edit.

// maxProjectName bounds a new project's name (it becomes the Save as file name).
const maxProjectName = 80

// getHistory is GET /project/history: can_undo, can_redo, what each would put back.
func (a *Area) getHistory(w http.ResponseWriter, r *http.Request) error {
	writeJSON(w, 200, a.deps.Projects.HistoryStatus())
	return nil
}

// stepHistory is POST /project/undo and /project/redo; the answer is the new status.
func (a *Area) stepHistory(undo bool) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		drainBody(r)
		step := a.deps.Projects.Redo
		if undo {
			step = a.deps.Projects.Undo
		}
		if err := step(); err != nil {
			return err
		}
		writeJSON(w, 200, a.deps.Projects.HistoryStatus())
		return nil
	}
}

// newProject is POST /project/new {"name": optional}: an empty project in place of this one.
func (a *Area) newProject(w http.ResponseWriter, r *http.Request) error {
	raw, _ := getJSON(r, true)
	name, err := projectName(get(object(raw), "name"))
	if err != nil {
		return err
	}
	return a.replaceProject(w, name)
}

// closeProject is POST /project/close: the open project is closed and an empty one is shown.
func (a *Area) closeProject(w http.ResponseWriter, r *http.Request) error {
	drainBody(r)
	return a.replaceProject(w, "")
}

func (a *Area) replaceProject(w http.ResponseWriter, name string) error {
	if err := a.requireStopped(); err != nil {
		return err
	}
	if err := a.deps.Projects.NewProject(name); err != nil {
		return err
	}
	writeJSON(w, 200, a.deps.Projects.Snapshot())
	return nil
}

// projectName checks a typed project name: trimmed, 1-80 characters, no control characters.
// A missing or empty name keeps the default name.
func projectName(value any) (string, error) {
	if value == nil {
		return "", nil
	}
	text, ok := value.(string)
	if !ok {
		return "", apperr.ValueError("The project name must be text")
	}
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > maxProjectName {
		return "", apperr.ValueError("Use a project name of at most 80 characters")
	}
	if strings.IndexFunc(text, unicode.IsControl) >= 0 {
		return "", apperr.ValueError("The project name cannot contain control characters")
	}
	return text, nil
}
