package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// Runs against a real, disposable database: TEST_DATABASE_URL=postgres://...
// Skipped otherwise. The schema is applied by NewPool, as at startup.
func TestIllustrationRepositoryAgainstPostgres(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `DELETE FROM illustrations; DELETE FROM illustration_folders`); err != nil {
		t.Fatal(err)
	}
	r := NewPromptTemplateRepository(pool)
	for range 2 { // seeding is repeated on every start
		if err := r.SeedIllustrations(ctx); err != nil {
			t.Fatal(err)
		}
	}
	all, _ := r.ListIllustrations(ctx, application.IllustrationFilter{})
	want := len(domain.BuiltinIllustrations()) + len(domain.ExemplarIllustrations())
	if len(all) != want {
		t.Fatalf("seeded %d rows, want %d", len(all), want)
	}
	cat, err := r.GetIllustration(ctx, "exemplar-Cat")
	if err != nil || !cat.Exemplar || cat.Code == "" || cat.Version != 1 {
		t.Fatalf("exemplar row: %+v %v", cat, err)
	}
	// An exemplar whose code changed in a new image gets a new version on seed.
	if _, err := pool.Exec(ctx, `UPDATE illustrations SET code = 'old' WHERE id = 'exemplar-Cat'`); err != nil {
		t.Fatal(err)
	}
	if err := r.SeedIllustrations(ctx); err != nil {
		t.Fatal(err)
	}
	if cat, _ = r.GetIllustration(ctx, "exemplar-Cat"); cat.Version != 2 {
		t.Fatalf("exemplar code change did not bump the version: v%d", cat.Version)
	}
	health, _ := r.ListIllustrations(ctx, application.IllustrationFilter{FolderID: "co-the-suc-khoe"})
	byTag, _ := r.ListIllustrations(ctx, application.IllustrationFilter{Query: "SÂU RĂNG"})
	if len(health) != 5 || len(byTag) != 1 || byTag[0].Name != "Tooth" {
		t.Fatalf("folder/tag filters: %d health, %v", len(health), byTag)
	}

	line := 7
	bus, err := r.CreateIllustration(ctx, domain.Illustration{Name: "Bus", Title: "Xe buýt", FolderID: "phuong-tien",
		Tags: []string{"xe"}, Code: "export function Bus() {}", Status: domain.IllustrationDraft, Version: 1,
		Warnings: []domain.CodeFinding{{Message: "[S9] màu #123456", Line: &line}}})
	if err != nil || bus.HasPreview {
		t.Fatalf("create: %+v %v", bus, err)
	}
	if len(bus.Warnings) != 1 || *bus.Warnings[0].Line != 7 {
		t.Fatalf("warnings not stored: %+v", bus.Warnings)
	}
	if _, err := r.CreateIllustration(ctx, domain.Illustration{Name: "Tooth", Title: "x", FolderID: "do-vat", Code: "x", Status: "draft", Version: 1}); !errors.Is(err, application.ErrIllustrationNameTaken) {
		t.Fatalf("duplicate name: %v", err)
	}
	if _, err := r.CreateIllustration(ctx, domain.Illustration{Name: "Ghost", Title: "x", FolderID: "khong-co", Code: "x", Status: "draft", Version: 1}); !errors.Is(err, application.ErrFolderNotFound) {
		t.Fatalf("unknown folder: %v", err)
	}

	if err := r.SaveIllustrationPreview(ctx, bus.ID, 1, []byte("png1"), []byte("gif1")); err != nil {
		t.Fatal(err)
	}
	if png, _, _ := r.GetIllustrationPreview(ctx, bus.ID); string(png) != "png1" {
		t.Fatalf("preview v1: %q", png)
	}
	bus.Version = 2
	bus, _ = r.UpdateIllustration(ctx, bus)
	if png, _, _ := r.GetIllustrationPreview(ctx, bus.ID); png != nil || bus.HasPreview {
		t.Fatal("a preview of version 1 was served for version 2")
	}
	if _, err := r.UpdateIllustration(ctx, domain.Illustration{ID: "builtin-Tooth", Name: "Tooth", Title: "hacked", FolderID: "do-vat", Status: "draft", Version: 9}); !errors.Is(err, application.ErrIllustrationNotFound) {
		t.Fatalf("built-in row was updated: %v", err)
	}

	f, err := r.CreateIllustrationFolder(ctx, domain.IllustrationFolder{ID: "do-choi", Name: "Đồ chơi"})
	if err != nil || f.Position != len(domain.SystemIllustrationFolders())+1 {
		t.Fatalf("folder: %+v %v", f, err)
	}
	if _, err := r.CreateIllustrationFolder(ctx, domain.IllustrationFolder{ID: "do-choi", Name: "x"}); !errors.Is(err, application.ErrFolderTaken) {
		t.Fatalf("duplicate folder: %v", err)
	}
}
