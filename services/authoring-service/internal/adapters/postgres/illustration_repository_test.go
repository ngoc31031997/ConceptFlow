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
	// A database from before still holds the three exemplars as seeded
	// built-ins; the first start after the change turns them into Hình mẫu data.
	if err := r.SeedIllustrations(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO illustrations (id, name, title, folder_id, code, builtin, exemplar, status)
		VALUES ('exemplar-Cat', 'Cat', 'Con mèo', 'dong-vat', 'export function Cat() {}', true, true, 'approved')`); err != nil {
		t.Fatal(err)
	}
	for range 2 { // seeding is repeated on every start
		if err := r.SeedIllustrations(ctx); err != nil {
			t.Fatal(err)
		}
	}
	all, _ := r.ListIllustrations(ctx, application.IllustrationFilter{})
	if want := len(domain.BuiltinIllustrations()) + 1; len(all) != want {
		t.Fatalf("seeded %d rows, want %d", len(all), want)
	}
	cat, err := r.GetIllustration(ctx, "exemplar-Cat")
	if err != nil || !cat.Exemplar || cat.Builtin || cat.FolderID != domain.ExemplarFolderID || cat.HomeFolderID != "dong-vat" || !cat.ReadOnly() {
		t.Fatalf("migrated exemplar: %+v %v", cat, err)
	}
	// Taken out of the Hình mẫu, it stays out across restarts.
	if cat, err = r.ReleaseExemplar(ctx, "exemplar-Cat", "dong-vat"); err != nil || cat.Exemplar || cat.HomeFolderID != "" || cat.FolderID != "dong-vat" {
		t.Fatalf("release: %+v %v", cat, err)
	}
	if err := r.SeedIllustrations(ctx); err != nil {
		t.Fatal(err)
	}
	if cat, _ = r.GetIllustration(ctx, "exemplar-Cat"); cat.Exemplar || cat.FolderID != "dong-vat" {
		t.Fatalf("seeding brought the exemplar back: %+v", cat)
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

// The illustration library against a real database (TEST_DATABASE_URL): who uses a drawing, the
// delete that refuses a project nobody checked, and the Hình mẫu limit.
func TestIllustrationUsageAndExemplarsAgainstPostgres(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `DELETE FROM project_illustrations; DELETE FROM project_authoring; DELETE FROM illustrations; DELETE FROM illustration_folders`); err != nil {
		t.Fatal(err)
	}
	r := NewPromptTemplateRepository(pool)
	if err := r.SeedIllustrations(ctx); err != nil {
		t.Fatal(err)
	}
	mk := func(name string) domain.Illustration {
		i, err := r.CreateIllustration(ctx, domain.Illustration{Name: name, Title: name, FolderID: "phuong-tien",
			Code: "export function " + name + "() {}", Status: domain.IllustrationApproved, Version: 1})
		if err != nil {
			t.Fatal(err)
		}
		return i
	}
	bus, busStop := mk("Bus"), mk("BusStop")
	if _, err := pool.Exec(ctx, `
		INSERT INTO project_authoring (project_id, topic, code_content) VALUES
		    ('p-code', 'Giao thông', 'return <Bus color="red" />;'),
		    ('p-link', 'Trường học', ''),
		    ('p-stop', 'Trạm', 'return <BusStop />;')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO project_illustrations (project_id, position, name, illustration_id) VALUES ('p-link', 1, 'Bus', $1)
	`, bus.ID); err != nil {
		t.Fatal(err)
	}
	users, err := r.FindIllustrationUsers(ctx, bus.ID, bus.Name)
	if err != nil || len(users) != 2 || users[0].ProjectID != "p-code" || users[0].Topic != "Giao thông" || users[1].ProjectID != "p-link" {
		t.Fatalf("users of Bus (not BusStop): %+v %v", users, err)
	}
	if users, _ := r.FindIllustrationUsers(ctx, busStop.ID, busStop.Name); len(users) != 1 || users[0].ProjectID != "p-stop" {
		t.Fatalf("users of BusStop: %+v", users)
	}

	if err := r.DeleteIllustration(ctx, bus.ID, []string{"p-code"}); !errors.Is(err, application.ErrIllustrationLinked) {
		t.Fatalf("delete with an unchecked link: %v", err)
	}
	if err := r.DeleteIllustration(ctx, bus.ID, []string{"p-code", "p-link"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetIllustration(ctx, bus.ID); !errors.Is(err, application.ErrIllustrationNotFound) {
		t.Fatalf("Bus still there: %v", err)
	}
	if err := r.DeleteIllustration(ctx, "builtin-Tooth", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetIllustration(ctx, "builtin-Tooth"); err != nil {
		t.Fatal("the kit must never be deleted")
	}

	cp, err := r.CreateExemplar(ctx, domain.Illustration{Name: "BusStopMau", Title: "Trạm", Code: "export function BusStopMau() {}",
		SourceID: busStop.ID, Status: domain.IllustrationApproved, Version: 1}, domain.MaxExemplars)
	if err != nil || !cp.Exemplar || cp.SourceID != busStop.ID || cp.FolderID != domain.ExemplarFolderID {
		t.Fatalf("exemplar copy: %+v %v", cp, err)
	}
	if _, err := r.CreateExemplar(ctx, domain.Illustration{Name: "BusStopMau2", Title: "x", Code: "x", SourceID: busStop.ID,
		Status: domain.IllustrationApproved, Version: 1}, domain.MaxExemplars); !errors.Is(err, application.ErrAlreadyExemplar) {
		t.Fatalf("second copy of one drawing: %v", err)
	}
	if _, err := r.CreateExemplar(ctx, domain.Illustration{Name: "Other", Title: "x", Code: "x",
		Status: domain.IllustrationApproved, Version: 1}, 1); !errors.Is(err, application.ErrExemplarLimit) {
		t.Fatalf("limit: %v", err)
	}
	// Deleting the source keeps the copy, which no longer points anywhere.
	if err := r.DeleteIllustration(ctx, busStop.ID, nil); err != nil {
		t.Fatal(err)
	}
	if cp, _ = r.GetIllustration(ctx, cp.ID); cp.SourceID != "" || !cp.IsExemplarCopy() {
		t.Fatalf("copy after its source went: %+v", cp)
	}
}
