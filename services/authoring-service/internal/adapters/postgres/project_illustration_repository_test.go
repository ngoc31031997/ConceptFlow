package postgres

import (
	"context"
	"os"
	"testing"

	"authoring/internal/domain"
)

func TestProjectIllustrationRepositoryAgainstPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := NewPool(ctx, url, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	r := NewPromptTemplateRepository(pool)
	if err := r.SeedIllustrations(ctx); err != nil {
		t.Fatal(err)
	}
	rows := []domain.ProjectIllustration{
		{Position: 1, Name: "Tooth", State: domain.PIReused, IllustrationID: "builtin-Tooth", FolderID: "co-the-suc-khoe"},
		{Position: 2, Name: "Motorbike", Description: "xe máy", State: domain.PIPlanned, FolderID: "phuong-tien", Shots: []string{"1.1"}},
	}
	if err := r.ReplaceProjectIllustrations(ctx, "pi-test", rows); err != nil {
		t.Fatal(err)
	}
	got, err := r.ListProjectIllustrations(ctx, "pi-test")
	if err != nil || len(got) != 2 {
		t.Fatalf("list: %v %v", got, err)
	}
	if got[0].Illustration == nil || got[0].Illustration.Name != "Tooth" || !got[0].Ready() {
		t.Fatalf("reused row not joined with its drawing: %+v", got[0])
	}
	if got[1].Ready() || got[1].Shots[0] != "1.1" {
		t.Fatalf("planned row: %+v", got[1])
	}
	got[1].State, got[1].Error = domain.PIFailed, "hỏng"
	if err := r.UpdateProjectIllustration(ctx, got[1]); err != nil {
		t.Fatal(err)
	}
	again, _ := r.ListProjectIllustrations(ctx, "pi-test")
	if again[1].State != domain.PIFailed || again[1].Error != "hỏng" {
		t.Fatalf("update lost: %+v", again[1])
	}
	// Re-planning replaces the list; deleting the project empties it.
	if err := r.ReplaceProjectIllustrations(ctx, "pi-test", rows[:1]); err != nil {
		t.Fatal(err)
	}
	if again, _ = r.ListProjectIllustrations(ctx, "pi-test"); len(again) != 1 {
		t.Fatalf("replace kept %d rows", len(again))
	}
	if err := r.DeleteAuthoring(ctx, "pi-test"); err != nil {
		t.Fatal(err)
	}
	if again, _ = r.ListProjectIllustrations(ctx, "pi-test"); len(again) != 0 {
		t.Fatal("project delete left its drawing list")
	}
}
