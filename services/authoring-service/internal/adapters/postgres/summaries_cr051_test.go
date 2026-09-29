package postgres

import (
	"context"
	"os"
	"testing"

	"authoring/internal/domain"
)

// CR-051: Summaries' illustrations_ready is the code step's gate in SQL. Each
// case checks the SQL answer against ProjectIllustration.Ready on the same rows,
// so the two cannot drift apart silently.
// Runs against a real, disposable database: TEST_DATABASE_URL=postgres://...
func TestSummariesIllustrationsReadyMatchesTheCodeGate(t *testing.T) {
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
	draft, err := r.CreateIllustration(ctx, domain.Illustration{
		Name: "Cr051Draft", Title: "nháp", FolderID: "co-the-suc-khoe", Code: "x", Status: domain.IllustrationDraft, Version: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.DeleteIllustration(ctx, draft.ID, nil) }()

	cases := []struct {
		name    string
		planned bool
		rows    []domain.ProjectIllustration
		want    bool
	}{
		{"never planned", false, nil, false},
		{"planned, nothing needed", true, nil, true},
		{"builtin reused", true, []domain.ProjectIllustration{
			{Position: 1, Name: "Tooth", State: domain.PIReused, IllustrationID: "builtin-Tooth", FolderID: "co-the-suc-khoe"},
		}, true},
		{"skipped", true, []domain.ProjectIllustration{
			{Position: 1, Name: "Bike", State: domain.PISkipped, FolderID: "phuong-tien"},
		}, true},
		{"one still planned", true, []domain.ProjectIllustration{
			{Position: 1, Name: "Tooth", State: domain.PIReused, IllustrationID: "builtin-Tooth", FolderID: "co-the-suc-khoe"},
			{Position: 2, Name: "Bike", State: domain.PIPlanned, FolderID: "phuong-tien"},
		}, false},
		{"drawn but not approved", true, []domain.ProjectIllustration{
			{Position: 1, Name: "Cr051Draft", State: domain.PIDrawn, IllustrationID: draft.ID, FolderID: "co-the-suc-khoe"},
		}, false},
		{"rows without planned_at", false, []domain.ProjectIllustration{
			{Position: 1, Name: "Bike", State: domain.PISkipped, FolderID: "phuong-tien"},
		}, true},
	}
	for i, c := range cases {
		id := "cr051-sum-" + string(rune('a'+i))
		if _, err := pool.Exec(ctx, `INSERT INTO project_authoring (project_id) VALUES ($1) ON CONFLICT DO NOTHING`, id); err != nil {
			t.Fatal(err)
		}
		if c.planned {
			if err := r.MarkIllustrationsPlanned(ctx, id); err != nil {
				t.Fatal(err)
			}
		}
		if err := r.ReplaceProjectIllustrations(ctx, id, c.rows); err != nil {
			t.Fatal(err)
		}
		sums, err := r.Summaries(ctx, []string{id})
		if err != nil {
			t.Fatal(err)
		}
		if got := sums[id].IllustrationsReady; got != c.want {
			t.Errorf("%s: illustrations_ready = %v, want %v", c.name, got, c.want)
		}
		// Same rows through the Go rule the code step uses.
		listed, err := r.ListProjectIllustrations(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		goReady := c.planned || len(listed) > 0
		for _, row := range listed {
			goReady = goReady && row.Ready()
		}
		if goReady != c.want {
			t.Errorf("%s: Ready() rule says %v, SQL case expects %v", c.name, goReady, c.want)
		}
		if err := r.DeleteAuthoring(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
}
