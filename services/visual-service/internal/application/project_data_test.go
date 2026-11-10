package application

import (
	"context"
	"errors"
	"testing"
)

type fakeProjectData struct {
	deleted []string
	err     error
}

func (f *fakeProjectData) DeleteProjectData(_ context.Context, projectID string) error {
	f.deleted = append(f.deleted, projectID)
	return f.err
}

func TestDeleteProject_RemovesTheProjectsData(t *testing.T) {
	store := &fakeProjectData{}
	if err := NewDeleteProjectUseCase(store).Execute(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	if len(store.deleted) != 1 || store.deleted[0] != "p1" {
		t.Fatalf("deleted %v", store.deleted)
	}
}

func TestDeleteProject_RefusesAnEmptyProject(t *testing.T) {
	store := &fakeProjectData{}
	if err := NewDeleteProjectUseCase(store).Execute(context.Background(), " "); !errors.Is(err, ErrNoProjectID) {
		t.Fatalf("err = %v, want ErrNoProjectID", err)
	}
	if len(store.deleted) != 0 {
		t.Fatal("nothing may be deleted without a project")
	}
}

func TestDeleteProject_ReportsAStoreFailure(t *testing.T) {
	store := &fakeProjectData{err: errors.New("db down")}
	if err := NewDeleteProjectUseCase(store).Execute(context.Background(), "p1"); err == nil {
		t.Fatal("a store failure must surface")
	}
}
