package postgres

import (
	"context"
	"os"
	"testing"
)

// openTestPool connects to TEST_DATABASE_URL and skips the test without it.
func openTestPool(t *testing.T) *ProjectDataRepository {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	pool, err := NewPool(context.Background(), url, 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return NewProjectDataRepository(pool)
}

func TestDeleteProjectData_RemovesOnlyThatProjectAndKeepsTheLibrary(t *testing.T) {
	repo := openTestPool(t)
	ctx := context.Background()
	const gone, kept, ref = "test-cr112a-gone", "test-cr112a-kept", "TEST112A"
	t.Cleanup(func() {
		for _, p := range []string{gone, kept} {
			_ = repo.DeleteProjectData(ctx, p)
		}
		_, _ = repo.pool.Exec(ctx, "DELETE FROM reference_styles WHERE id = $1", ref)
	})
	if _, err := repo.pool.Exec(ctx, `INSERT INTO reference_styles (id, seq, name, description) VALUES ($1, -112, 'n', 'd')`, ref); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{gone, kept} {
		for _, q := range []string{
			`INSERT INTO review_units (project_id, kind, unit_key) VALUES ($1, 'visual_concept', 'color_system')`,
			`INSERT INTO visual_concept_versions (project_id, version, script_version, document) VALUES ($1, 1, 1, '{}')`,
			`INSERT INTO visual_plan_versions (project_id, version, script_version, concept_version, document) VALUES ($1, 1, 1, 1, '{}')`,
			`INSERT INTO visual_plan_seq (project_id, next) VALUES ($1, 2)`,
			`INSERT INTO project_visual (project_id) VALUES ($1)`,
			`INSERT INTO project_reference_picks (project_id, ref_id) VALUES ($1, '` + ref + `')`,
		} {
			if _, err := repo.pool.Exec(ctx, q, p); err != nil {
				t.Fatal(err)
			}
		}
	}

	if err := repo.DeleteProjectData(ctx, gone); err != nil {
		t.Fatal(err)
	}
	for _, table := range projectTables {
		var g, k int
		_ = repo.pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE project_id = $1", gone).Scan(&g)
		_ = repo.pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE project_id = $1", kept).Scan(&k)
		if g != 0 || k != 1 {
			t.Errorf("%s: deleted project has %d rows, other project has %d", table, g, k)
		}
	}
	var refs int
	_ = repo.pool.QueryRow(ctx, "SELECT count(*) FROM reference_styles WHERE id = $1", ref).Scan(&refs)
	if refs != 1 {
		t.Error("the reference library must not be touched")
	}
	if err := repo.DeleteProjectData(ctx, "test-cr112a-never-seen"); err != nil {
		t.Errorf("a project with no data is not an error: %v", err)
	}
}
