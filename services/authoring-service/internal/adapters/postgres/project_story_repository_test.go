package postgres

import (
	"errors"
	"testing"
	"time"

	"authoring/internal/domain"
)

// A story survives a round trip through project_story, a second save replaces
// it, and a project without one is not found.
func TestProjectStoryRoundTripInPostgres(t *testing.T) {
	const pid = "project-story-test"
	r, ctx := researchRepo(t, pid)
	if _, err := r.GetProjectStory(ctx, pid); !errors.Is(err, domain.ErrStoryNotFound) {
		t.Fatalf("err = %v, want ErrStoryNotFound before any save", err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	want := domain.StoredStory{
		ProjectID: pid, Status: domain.StoryApproved, CreatorFields: []string{domain.StoryFieldCoreAnswer},
		BriefHash: "bh", ResearchHash: "rh", CreatedAt: now, UpdatedAt: now,
		Story: domain.StoryStructure{
			CentralQuestion: "Vì sao?", CoreAnswer: "Vì thế.",
			NarrativeArc: []domain.ArcBeat{{Beat: "Hook", Text: "Mở", Facts: []string{"F01", "F02"}}, {Beat: "Payoff", Text: "Kết"}},
		},
	}
	if err := r.SaveProjectStory(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetProjectStory(ctx, pid)
	if err != nil {
		t.Fatal(err)
	}
	if got.Story.CentralQuestion != "Vì sao?" || got.Status != domain.StoryApproved || got.BriefHash != "bh" || got.ResearchHash != "rh" ||
		len(got.CreatorFields) != 1 || len(got.Story.NarrativeArc) != 2 || len(got.Story.NarrativeArc[0].Facts) != 2 {
		t.Fatalf("story = %+v, want what was saved", got)
	}
	if got.Story.NarrativeArc[1].Facts == nil || len(got.Story.NarrativeArc[1].Facts) != 0 {
		t.Errorf("facts of a beat citing none = %v, want an empty list", got.Story.NarrativeArc[1].Facts)
	}
	want.Story.CoreAnswer, want.Status = "Khác", domain.StoryPending
	if err := r.SaveProjectStory(ctx, want); err != nil {
		t.Fatal(err)
	}
	if got, _ = r.GetProjectStory(ctx, pid); got.Story.CoreAnswer != "Khác" || got.Status != domain.StoryPending {
		t.Errorf("story = %+v, want the second save to replace the first", got)
	}
	if err := r.DeleteAuthoring(ctx, pid); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetProjectStory(ctx, pid); !errors.Is(err, domain.ErrStoryNotFound) {
		t.Errorf("err = %v, want deleting a project's authoring to delete its story", err)
	}
}

// A unit that names the story as its input reads as stale once the story
// changes, and fresh while it does not.
func TestUnitBuiltFromTheStoryGoesStaleWhenTheStoryChanges(t *testing.T) {
	const pid = "story-input-test"
	r, ctx := researchRepo(t, pid)
	story := domain.StoredStory{
		ProjectID: pid, Status: domain.StoryApproved, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		Story: domain.StoryStructure{CentralQuestion: "Q", CoreAnswer: "A", NarrativeArc: []domain.ArcBeat{{Beat: "H", Text: "t"}}},
	}
	if err := r.SaveProjectStory(ctx, story); err != nil {
		t.Fatal(err)
	}
	sha, err := r.UnitSHA(ctx, pid, domain.StoryInputKind, domain.StoryInputKey)
	if err != nil || sha == "" {
		t.Fatalf("sha = %q err = %v, want a hash of the story", sha, err)
	}
	if _, err := r.SaveUnit(ctx, domain.ReviewUnit{
		ProjectID: pid, Kind: domain.UnitKindBeat, Key: "b1", Status: domain.UnitPending, Content: []byte(`{}`),
		InputRef: domain.StoryInputKind + ":" + domain.StoryInputKey, InputSHA: sha,
	}); err != nil {
		t.Fatal(err)
	}
	unit, err := r.GetUnit(ctx, pid, domain.UnitKindBeat, "b1")
	if err != nil || unit.Stale {
		t.Fatalf("stale = %v err = %v, want a unit built from the current story fresh", unit.Stale, err)
	}
	story.Story.NarrativeArc[0].Text = "đã đổi"
	if err := r.SaveProjectStory(ctx, story); err != nil {
		t.Fatal(err)
	}
	if unit, _ = r.GetUnit(ctx, pid, domain.UnitKindBeat, "b1"); !unit.Stale {
		t.Error("a unit built from a story that has changed since must read as stale")
	}
	if listed, _ := r.ListUnits(ctx, pid, domain.UnitKindBeat); len(listed) != 1 || !listed[0].Stale {
		t.Errorf("listed = %+v, want the same stale unit when listed", listed)
	}
	if sha, err := r.UnitSHA(ctx, "no-such-project", domain.StoryInputKind, domain.StoryInputKey); err != nil || sha != "" {
		t.Errorf("sha = %q err = %v, want \"\" for a project with no story", sha, err)
	}
}
