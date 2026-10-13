package postgres

import (
	"context"
	"os"
	"testing"
)

// Runs against a real, disposable database: TEST_DATABASE_URL=postgres://...
// Skipped otherwise. The schema is applied by NewPool, as at startup.
func TestPurgeLegacyPromptsDeletesEveryRowOfARetiredRole(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `DELETE FROM prompts WHERE role IN ('short_script', 'manim_engineer', 'thumbnail_design')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO prompts (id, role, name, template_text, is_system, is_active) VALUES
		    ('retired-system', 'short_script', 'Mặc định', 'x', true, true),
		    ('retired-creator', 'short_script', 'Của tôi', 'y', false, false),
		    ('retired-manim', 'manim_engineer', 'Mặc định', 'm', true, true),
		    ('kept-system', 'thumbnail_design', 'Mặc định', 'z', true, true)`); err != nil {
		t.Fatal(err)
	}

	r := NewPromptTemplateRepository(pool)
	n, err := r.PurgeLegacyPrompts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n < 3 {
		t.Errorf("purge reported %d deleted rows, want at least the 2 short_script rows and the Manim row", n)
	}
	var retired, kept int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM prompts WHERE role IN ('short_script', 'manim_engineer')`).Scan(&retired); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM prompts WHERE id = 'kept-system'`).Scan(&kept); err != nil {
		t.Fatal(err)
	}
	if retired != 0 {
		t.Errorf("%d rows of retired roles left after purge, want 0", retired)
	}
	if kept != 1 {
		t.Errorf("a prompt of a live role was deleted by the purge")
	}
}
