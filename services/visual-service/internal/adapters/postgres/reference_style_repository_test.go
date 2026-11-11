package postgres

import (
	"context"
	"errors"
	"testing"

	"visual/internal/application"
	"visual/internal/domain"
)

const referenceTestPrefix = "test-cr112c"

// openReferenceRepository connects to TEST_DATABASE_URL (skipping without it)
// and removes, after the test, every description and pick it made.
func openReferenceRepository(t *testing.T) *ReferenceStyleRepository {
	t.Helper()
	data := openTestPool(t)
	repo := NewReferenceStyleRepository(data.pool)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = data.pool.Exec(ctx, "DELETE FROM project_reference_picks WHERE project_id LIKE $1", referenceTestPrefix+"%")
		_, _ = data.pool.Exec(ctx, "DELETE FROM reference_styles WHERE name LIKE $1", referenceTestPrefix+"%")
	})
	return repo
}

func TestReferenceStyles_CreateListUpdateDelete(t *testing.T) {
	repo := openReferenceRepository(t)
	ctx := context.Background()

	first, err := repo.CreateReferenceStyle(ctx, domain.ReferenceStyleInput{StyleName: referenceTestPrefix + "-a", Intent: "d", Tags: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.CreateReferenceStyle(ctx, domain.ReferenceStyleInput{StyleName: referenceTestPrefix + "-b", Intent: "d", Tags: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID || first.ID[0] != 'R' {
		t.Fatalf("ids %q and %q", first.ID, second.ID)
	}

	list, err := repo.ListReferenceStyles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	firstAt, secondAt := -1, -1
	for i, s := range list {
		switch s.ID {
		case first.ID:
			firstAt = i
		case second.ID:
			secondAt = i
		}
	}
	if firstAt < 0 || secondAt < firstAt {
		t.Fatalf("list must hold both in the order they were added: %d, %d", firstAt, secondAt)
	}

	updated, err := repo.UpdateReferenceStyle(ctx, first.ID, domain.ReferenceStyleInput{StyleName: referenceTestPrefix + "-a2", Intent: "d2", Tags: []string{"y", "z"}})
	if err != nil {
		t.Fatal(err)
	}
	if updated.StyleName != referenceTestPrefix+"-a2" || len(updated.Tags) != 2 || updated.UpdatedAt.Before(first.UpdatedAt) {
		t.Errorf("update returned %+v", updated)
	}

	if err := repo.DeleteReferenceStyle(ctx, second.ID); err != nil {
		t.Fatal(err)
	}
	third, err := repo.CreateReferenceStyle(ctx, domain.ReferenceStyleInput{StyleName: referenceTestPrefix + "-c", Intent: "d", Tags: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if third.ID == second.ID {
		t.Errorf("deleted id %s was handed out again", second.ID)
	}
}

func TestReferenceStyles_UnknownIDIsNotFound(t *testing.T) {
	repo := openReferenceRepository(t)
	ctx := context.Background()
	const unknown = "R-none"
	if _, err := repo.UpdateReferenceStyle(ctx, unknown, domain.ReferenceStyleInput{StyleName: "n", Intent: "d", Tags: []string{}}); !errors.Is(err, application.ErrReferenceStyleNotFound) {
		t.Errorf("update: %v", err)
	}
	if err := repo.DeleteReferenceStyle(ctx, unknown); !errors.Is(err, application.ErrReferenceStyleNotFound) {
		t.Errorf("delete: %v", err)
	}
	if _, err := repo.ReferenceStyleUsers(ctx, unknown); !errors.Is(err, application.ErrReferenceStyleNotFound) {
		t.Errorf("users: %v", err)
	}
}

func TestReferenceStyles_UsersAndCascadeOnDelete(t *testing.T) {
	repo := openReferenceRepository(t)
	ctx := context.Background()
	s, err := repo.CreateReferenceStyle(ctx, domain.ReferenceStyleInput{StyleName: referenceTestPrefix + "-used", Intent: "d", Tags: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	none, err := repo.ReferenceStyleUsers(ctx, s.ID)
	if err != nil || len(none) != 0 {
		t.Fatalf("unpicked style: users %v err %v", none, err)
	}
	for _, p := range []string{referenceTestPrefix + "-p2", referenceTestPrefix + "-p1"} {
		if _, err := repo.pool.Exec(ctx, "INSERT INTO project_reference_picks (project_id, ref_id) VALUES ($1, $2)", p, s.ID); err != nil {
			t.Fatal(err)
		}
	}
	users, err := repo.ReferenceStyleUsers(ctx, s.ID)
	if err != nil || len(users) != 2 || users[0] != referenceTestPrefix+"-p1" {
		t.Fatalf("users %v err %v", users, err)
	}
	if err := repo.DeleteReferenceStyle(ctx, s.ID); err != nil {
		t.Fatal(err)
	}
	var picks int
	_ = repo.pool.QueryRow(ctx, "SELECT count(*) FROM project_reference_picks WHERE ref_id = $1", s.ID).Scan(&picks)
	if picks != 0 {
		t.Errorf("%d picks left after deleting the style", picks)
	}
}

func TestReferenceStyles_DetailRoundTrips(t *testing.T) {
	repo := openReferenceRepository(t)
	ctx := context.Background()
	in := domain.ReferenceStyleInput{
		StyleName: referenceTestPrefix + "-detail", Intent: "tóm tắt", Tags: []string{},
		Source:    domain.ReferenceSource{URL: "https://example.com/v", Title: "t", Kind: "video"},
		Drawing:   domain.ReferenceDrawing{Shape: "tròn", Texture: "giấy"},
		CoreRules: []string{"no outlines"}, Avoid: []string{"pastel"},
		ColorRules:     domain.ReferenceColorRules{Background: "tối", Accents: "phát sáng"},
		Composition:    domain.ReferenceComposition{Structure: "lưới", OptionalDevices: []string{"inset"}},
		PromptTemplate: "{subject} in {setting}", NegativePrompt: "outline",
	}
	created, err := repo.CreateReferenceStyle(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if created.Intent != "tóm tắt" || created.Source != in.Source || created.Drawing != in.Drawing || created.ColorRules != in.ColorRules ||
		created.CoreRules[0] != "no outlines" || created.Avoid[0] != "pastel" || created.Composition.OptionalDevices[0] != "inset" ||
		created.PromptTemplate != in.PromptTemplate || created.NegativePrompt != in.NegativePrompt {
		t.Fatalf("created %+v", created)
	}
	in.Drawing.Shape = "vuông"
	in.Source = domain.ReferenceSource{}
	if _, err := repo.UpdateReferenceStyle(ctx, created.ID, in); err != nil {
		t.Fatal(err)
	}
	list, err := repo.ListReferenceStyles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range list {
		if s.ID == created.ID && (s.Drawing.Shape != "vuông" || s.Source.URL != "" || s.Composition.Structure != "lưới") {
			t.Errorf("listed %+v", s)
		}
	}
}
