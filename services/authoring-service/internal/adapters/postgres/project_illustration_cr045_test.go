package postgres

import (
	"context"
	"os"
	"testing"

	"authoring/internal/domain"
)

// CR-045 against a real database (TEST_DATABASE_URL): the "planned" mark, and
// the startup clean-up that drops stored out-of-palette (S9) warnings.
func TestIllustrationsPlannedMarkAndS9CleanupAgainstPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := NewPool(ctx, url, 4)
	if err != nil {
		t.Fatal(err)
	}
	r := NewPromptTemplateRepository(pool)
	if _, err := pool.Exec(ctx, `DELETE FROM project_authoring WHERE project_id IN ('cr045-a', 'cr045-b')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO project_authoring (project_id) VALUES ('cr045-a')`); err != nil {
		t.Fatal(err)
	}
	for _, pid := range []string{"cr045-a", "cr045-b"} { // with and without an authoring row
		if planned, err := r.IllustrationsPlanned(ctx, pid); err != nil || planned {
			t.Fatalf("%s before planning: planned=%v err=%v", pid, planned, err)
		}
		if err := r.MarkIllustrationsPlanned(ctx, pid); err != nil {
			t.Fatal(err)
		}
		if planned, err := r.IllustrationsPlanned(ctx, pid); err != nil || !planned {
			t.Fatalf("%s after planning: planned=%v err=%v", pid, planned, err)
		}
	}

	if err := r.SeedIllustrations(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM illustrations WHERE name = 'Cr045Palette'`); err != nil {
		t.Fatal(err)
	}
	line := 4
	saved, err := r.CreateIllustration(ctx, domain.Illustration{Name: "Cr045Palette", Title: "x", FolderID: "do-vat", Code: "x",
		Status: domain.IllustrationDraft, Version: 1, Warnings: []domain.CodeFinding{
			{Message: "[S9] màu #123456 không có trong bảng màu kênh", Line: &line},
			{Message: "[S3] <rect> không bo góc (thêm rx)", Line: &line},
		}})
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	pool, err = NewPool(ctx, url, 4) // the schema runs again, as at the next start
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	got, err := NewPromptTemplateRepository(pool).GetIllustration(ctx, saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Warnings) != 1 || got.Warnings[0].Message != "[S3] <rect> không bo góc (thêm rx)" {
		t.Fatalf("warnings after clean-up = %+v, want only the S3 one", got.Warnings)
	}
}
