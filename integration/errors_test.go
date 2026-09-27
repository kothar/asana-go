package integration

import (
	"testing"

	"github.com/kothar/asana-go"
)

func TestInvalidToken(t *testing.T) {
	setup(t)

	_, err := newClient("not-a-valid-token").CurrentUser()
	if !asana.IsAuthError(err) {
		t.Errorf("expected an auth error for an invalid token, got %v", err)
	}
	if !asana.IsFatalError(err) || asana.IsRecoverableError(err) {
		t.Errorf("expected an auth error to be fatal and not recoverable, got %v", err)
	}
}

func TestMissingTask(t *testing.T) {
	f := setup(t)
	p := f.newProject(t, "project")
	task := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})
	if err := task.Delete(f.client); err != nil {
		t.Fatal(err)
	}

	err := task.Update(f.client, &asana.UpdateTaskRequest{TaskBase: asana.TaskBase{Notes: "gone"}})
	if !asana.IsNotFoundError(err) {
		t.Errorf("expected a not found error updating a deleted task, got %v", err)
	}
	e, ok := asana.IsAsanaError(err)
	if !ok || e.Message == "" {
		t.Errorf("expected Asana's error message to be decoded, got %+v", e)
	}
}
