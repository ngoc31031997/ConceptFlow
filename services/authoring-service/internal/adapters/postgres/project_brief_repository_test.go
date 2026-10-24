package postgres

import (
	"errors"
	"testing"
	"time"

	"authoring/internal/domain"
)

func TestProjectBriefRoundTripsAndGoesWithTheProject(t *testing.T) {
	r, ctx := researchRepo(t, "brief-test")
	if _, err := r.GetProjectBrief(ctx, "brief-test"); !errors.Is(err, domain.ErrBriefNotFound) {
		t.Fatalf("err = %v, want ErrBriefNotFound", err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	brief := domain.StoredBrief{
		ProjectID: "brief-test",
		Brief: domain.ProjectBrief{
			Audience: "Khán giả phổ thông", DurationSeconds: 270, Tone: "Tò mò",
			LearningObjectives: []string{"Giải thích chân trời sự kiện", "Hiểu lực thủy triều"},
			Avoid:              []string{"Mô tả chân trời sự kiện như vỏ cứng"},
		},
		CreatorFields: []string{domain.BriefFieldTone}, Status: domain.BriefApproved, RedoCount: 2,
		Language: "vi", IdeaHash: "h", ApprovedAt: &now, UpdatedAt: now,
	}
	if err := r.SaveProjectBrief(ctx, brief); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetProjectBrief(ctx, "brief-test")
	if err != nil {
		t.Fatal(err)
	}
	if got.Brief.DurationSeconds != 270 || len(got.Brief.LearningObjectives) != 2 || len(got.Brief.Avoid) != 1 ||
		got.Status != domain.BriefApproved || got.RedoCount != 2 || got.ApprovedAt == nil || !got.HasCreatorField(domain.BriefFieldTone) {
		t.Fatalf("round trip = %+v", got)
	}

	brief.Brief.DurationSeconds = 300
	brief.Status, brief.ApprovedAt = domain.BriefPending, nil
	if err := r.SaveProjectBrief(ctx, brief); err != nil {
		t.Fatal(err)
	}
	if got, _ = r.GetProjectBrief(ctx, "brief-test"); got.Brief.DurationSeconds != 300 || got.Status != domain.BriefPending || got.ApprovedAt != nil {
		t.Errorf("update = %+v", got)
	}

	if _, err := r.GetBriefReview(ctx, "brief-test"); !errors.Is(err, domain.ErrBriefNotFound) {
		t.Fatalf("review before any: err = %v", err)
	}
	review := domain.BriefReview{
		AudienceFit: domain.IdeaAspect{Rating: domain.RatingGood, Comment: "Rõ."},
		DurationFit: domain.IdeaAspect{Rating: domain.RatingFair, Comment: "Hơi dài."},
		Objectives:  domain.IdeaAspect{Rating: domain.RatingGood, Comment: "Cụ thể."},
		AvoidList:   domain.IdeaAspect{Rating: domain.RatingWeak, Comment: "Thiếu."},
		Unclear:     []string{}, Suggestions: []string{"Thêm"}, BriefHash: brief.Brief.Hash(), CreatedAt: now,
	}
	if err := r.SaveBriefReview(ctx, "brief-test", review); err != nil {
		t.Fatal(err)
	}
	gotReview, err := r.GetBriefReview(ctx, "brief-test")
	if err != nil || gotReview.AvoidList.Rating != domain.RatingWeak || gotReview.BriefHash != review.BriefHash {
		t.Fatalf("review round trip = %+v err %v", gotReview, err)
	}

	if err := r.DeleteAuthoring(ctx, "brief-test"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetProjectBrief(ctx, "brief-test"); !errors.Is(err, domain.ErrBriefNotFound) {
		t.Errorf("the brief outlived its project: %v", err)
	}
	if _, err := r.GetBriefReview(ctx, "brief-test"); !errors.Is(err, domain.ErrBriefNotFound) {
		t.Errorf("the review outlived its project: %v", err)
	}
}
