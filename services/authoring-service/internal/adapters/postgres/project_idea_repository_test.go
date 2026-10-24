package postgres

import (
	"errors"
	"testing"
	"time"

	"authoring/internal/domain"
)

func TestProjectIdeaRoundTripsAndGoesWithTheProject(t *testing.T) {
	r, ctx := researchRepo(t, "idea-test")
	if _, err := r.GetProjectIdea(ctx, "idea-test"); !errors.Is(err, domain.ErrIdeaNotFound) {
		t.Fatalf("err = %v, want ErrIdeaNotFound", err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	idea := domain.StoredIdea{
		ProjectID: "idea-test",
		Idea: domain.ProjectIdea{
			CentralQuestion: "Người rơi thấy gì?",
			CuriosityPoints: []string{"Chân trời sự kiện?", "Có bị kéo dài không?"}, Scope: "Không toán.",
		},
		CreatorFields: []string{domain.IdeaFieldScope}, Status: domain.IdeaApproved, RedoCount: 2,
		Language: "vi", InputHash: "h", ApprovedAt: &now, UpdatedAt: now,
	}
	if err := r.SaveProjectIdea(ctx, idea); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetProjectIdea(ctx, "idea-test")
	if err != nil {
		t.Fatal(err)
	}
	if got.Idea.CentralQuestion != idea.Idea.CentralQuestion || len(got.Idea.CuriosityPoints) != 2 ||
		got.Status != domain.IdeaApproved || got.RedoCount != 2 || got.ApprovedAt == nil || !got.HasCreatorField(domain.IdeaFieldScope) {
		t.Fatalf("round trip = %+v", got)
	}

	idea.Idea.Scope = "Phạm vi khác"
	idea.Status, idea.ApprovedAt = domain.IdeaPending, nil
	if err := r.SaveProjectIdea(ctx, idea); err != nil {
		t.Fatal(err)
	}
	if got, _ = r.GetProjectIdea(ctx, "idea-test"); got.Idea.Scope != "Phạm vi khác" || got.Status != domain.IdeaPending || got.ApprovedAt != nil {
		t.Errorf("update = %+v", got)
	}

	if _, err := r.GetIdeaReview(ctx, "idea-test"); !errors.Is(err, domain.ErrIdeaNotFound) {
		t.Fatalf("review before any: err = %v", err)
	}
	review := domain.IdeaReview{
		Clarity:         domain.IdeaAspect{Rating: domain.RatingGood, Comment: "Rõ."},
		Researchability: domain.IdeaAspect{Rating: domain.RatingFair, Comment: "Cần nguồn."},
		ScopeFit:        domain.IdeaAspect{Rating: domain.RatingWeak, Comment: "Rộng."},
		Unclear:         []string{"Ai xem?"}, Suggestions: []string{}, IdeaHash: idea.Idea.Hash(), CreatedAt: now,
	}
	if err := r.SaveIdeaReview(ctx, "idea-test", review); err != nil {
		t.Fatal(err)
	}
	gotReview, err := r.GetIdeaReview(ctx, "idea-test")
	if err != nil || gotReview.ScopeFit.Rating != domain.RatingWeak || gotReview.IdeaHash != review.IdeaHash {
		t.Fatalf("review round trip = %+v err %v", gotReview, err)
	}

	if err := r.DeleteAuthoring(ctx, "idea-test"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetProjectIdea(ctx, "idea-test"); !errors.Is(err, domain.ErrIdeaNotFound) {
		t.Errorf("the idea outlived its project: %v", err)
	}
	if _, err := r.GetIdeaReview(ctx, "idea-test"); !errors.Is(err, domain.ErrIdeaNotFound) {
		t.Errorf("the review outlived its project: %v", err)
	}
}
