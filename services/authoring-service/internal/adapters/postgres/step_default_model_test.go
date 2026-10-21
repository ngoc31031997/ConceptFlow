package postgres

import (
	"context"
	"os"
	"testing"

	"authoring/internal/domain"
)

// Runs against a real, disposable database: TEST_DATABASE_URL=postgres://...
// Skipped otherwise. The schema is applied by NewPool, as at startup.
func TestStepDefaultModelsAgainstPostgres(t *testing.T) {
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

	const projectID = "step-default-model-test"
	clearSettings := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app_settings WHERE key LIKE $1`, stepModelKeyPrefix+"%")
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM project_authoring WHERE project_id = $1`, projectID)
		clearSettings()
	})
	clearSettings()

	// With nothing saved, every step follows the first model of the catalog.
	steps, err := r.GetStepDefaultModels(ctx)
	if err != nil || len(steps) != len(domain.ConfigurableModelSteps) {
		t.Fatalf("GetStepDefaultModels = %v, %v; want one entry per configurable step", steps, err)
	}
	for step, model := range steps {
		if model != domain.AuthoringModelCatalog[0].ID {
			t.Errorf("step %q = %q, want the first catalog model", step, model)
		}
	}

	if err := r.SaveStepDefaultModel(ctx, "research", "deepseek-ai/deepseek-v4.1-flash"); err != nil {
		t.Fatal(err)
	}
	if err := r.SaveStepDefaultModel(ctx, "brief", "zai-org/glm-5.3-flash"); err != nil {
		t.Fatal(err)
	}
	if err := r.SaveStepDefaultModel(ctx, "brief", "ollama"); err != nil {
		t.Fatal(err)
	}
	steps, _ = r.GetStepDefaultModels(ctx)
	if steps["research"] != "deepseek-ai/deepseek-v4.1-flash" || steps["brief"] != "ollama" || steps["code"] != "ollama" {
		t.Fatalf("saved defaults not read back, got %v", steps)
	}

	// The project picked a model for story only.
	if err := r.SaveAuthoringModels(ctx, projectID, domain.AuthoringStepModels{Story: "zai-org/glm-5.3-flash"}); err != nil {
		t.Fatal(err)
	}
	chosen, err := r.GetChosenAuthoringModels(ctx, projectID)
	if err != nil || chosen.Research != "" || chosen.Story != "zai-org/glm-5.3-flash" {
		t.Fatalf("GetChosenAuthoringModels = %+v, %v; must hold only what the project picked", chosen, err)
	}
	filled, err := r.GetAuthoringModels(ctx, projectID)
	if err != nil || filled.Story != "zai-org/glm-5.3-flash" || filled.Research != "deepseek-ai/deepseek-v4.1-flash" || filled.Code != "ollama" {
		t.Fatalf("GetAuthoringModels = %+v, %v; want the project's pick, else the step default", filled, err)
	}

	// A project with no authoring row follows the step defaults.
	none, err := r.GetAuthoringModels(ctx, "no-such-project")
	if err != nil || none.Brief != "ollama" || none.Research != "deepseek-ai/deepseek-v4.1-flash" {
		t.Fatalf("GetAuthoringModels for a new project = %+v, %v", none, err)
	}
}
