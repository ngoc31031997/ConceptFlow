package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"visual/internal/application"
	"visual/internal/domain"
)

func TestConceptRepository(t *testing.T) {
	data := openTestPool(t)
	repo := NewConceptRepository(data.pool)
	ctx := context.Background()
	const p, r1, r2 = "test-cr112b-repo", "TEST112B1", "TEST112B2"
	t.Cleanup(func() {
		_ = data.DeleteProjectData(ctx, p)
		_, _ = data.pool.Exec(ctx, "DELETE FROM reference_styles WHERE id = ANY($1)", []string{r1, r2})
	})
	// The library owns the detail column; a database that does not have it yet gets it here.
	if _, err := data.pool.Exec(ctx, `ALTER TABLE reference_styles ADD COLUMN IF NOT EXISTS detail JSONB`); err != nil {
		t.Fatal(err)
	}
	for i, id := range []string{r1, r2} {
		if _, err := data.pool.Exec(ctx, `INSERT INTO reference_styles (id, seq, name, description, tags) VALUES ($1, $2, 'n', 'd', '{a}')`, id, -1120-i); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := data.pool.Exec(ctx, `UPDATE reference_styles SET detail = '{"drawing":{"shape":"tròn","light":"a"},"composition":{"layering":"ba lớp","optional_devices":["inset"]},"avoid":["sharp corners"],"source":{"url":"u"}}' WHERE id = $1`, r1); err != nil {
		t.Fatal(err)
	}

	u := domain.ReviewUnit{ProjectID: p, Kind: domain.UnitKindConcept, Key: domain.UnitTypography, Status: domain.UnitPending,
		Content: []byte(`{"font":"A"}`), CreatorFields: []string{"font"}, InputHashes: map[string]string{"brief": "h"}}
	if err := repo.SaveUnit(ctx, u); err != nil {
		t.Fatal(err)
	}
	u.Status, u.RedoCount = domain.UnitApproved, 2
	if err := repo.SaveUnit(ctx, u); err != nil {
		t.Fatal(err)
	}
	units, err := repo.ListUnits(ctx, p, domain.UnitKindConcept)
	if err != nil || len(units) != 1 || units[0].Status != domain.UnitApproved || units[0].RedoCount != 2 ||
		units[0].InputHashes["brief"] != "h" || units[0].CreatorFields[0] != "font" {
		t.Fatalf("%+v %v", units, err)
	}

	if m, _ := repo.ConceptModel(ctx, p); m != "" {
		t.Error("no model saved reads as ''")
	}
	if err := repo.SetConceptModel(ctx, p, "ollama"); err != nil {
		t.Fatal(err)
	}
	if m, _ := repo.ConceptModel(ctx, p); m != "ollama" {
		t.Error(m)
	}

	if err := repo.SetPicks(ctx, p, []string{r1, "TEST112B-none"}); !errors.Is(err, domain.ErrUnknownReference) {
		t.Errorf("unknown: %v", err)
	}
	if err := repo.SetPicks(ctx, p, []string{r2, r1}); err != nil {
		t.Fatal(err)
	}
	picked, _ := repo.PickedReferences(ctx, p)
	if len(picked) != 2 || picked[0].ID != r2 || picked[0].Tags[0] != "a" || picked[0].Intent != "d" || picked[0].Drawing.Shape != "" || picked[0].Avoid == nil || picked[0].CoreRules == nil {
		t.Errorf("picks in library order: %+v", picked)
	}
	if picked[1].Drawing.Shape != "tròn" || picked[1].Drawing.Light != "a" || picked[1].Composition.Layering != "ba lớp" || picked[1].Composition.OptionalDevices[0] != "inset" || picked[1].Avoid[0] != "sharp corners" || picked[1].Source.URL != "u" {
		t.Errorf("detail read: %+v", picked[1])
	}
	if err := repo.SetPicks(ctx, p, []string{r1}); err != nil {
		t.Fatal(err)
	}
	if picked, _ = repo.PickedReferences(ctx, p); len(picked) != 1 {
		t.Error("picks are replaced")
	}

	at := time.Now().UTC().Truncate(time.Second)
	v := application.VersionRecord{ProjectID: p, Version: 1, ScriptVersion: 2, Document: []byte(`{"a":1}`), Note: "n", LockedAt: at}
	if err := repo.InsertVersion(ctx, v); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertVersion(ctx, v); !errors.Is(err, domain.ErrVersionExists) {
		t.Errorf("duplicate: %v", err)
	}
	vs, _ := repo.Versions(ctx, p)
	got, err := repo.Version(ctx, p, 1)
	if len(vs) != 1 || vs[0].Document != nil || err != nil || string(got.Document) != `{"a": 1}` || got.ScriptVersion != 2 {
		t.Errorf("%+v %+v %v", vs, got, err)
	}
	if _, err := repo.Version(ctx, p, 9); !errors.Is(err, domain.ErrNoVersion) {
		t.Error(err)
	}
}
