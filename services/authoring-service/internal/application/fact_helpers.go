package application

import (
	"context"
	"strings"

	"authoring/internal/domain"
)

// storyFact is a kept fact with the content the prompts and the screen need.
type storyFact struct {
	ID string
	domain.Fact
}

// loadKeptFacts loads the research facts of a project the Creator has not dropped, in code order.
func loadKeptFacts(ctx context.Context, store ResearchFactStorePort, projectID string) ([]storyFact, error) {
	stored, err := store.ListResearchFacts(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]storyFact, 0, len(stored))
	for _, f := range stored {
		if f.Fact.Keep {
			out = append(out, storyFact{ID: f.ID, Fact: f.Fact})
		}
	}
	return out, nil
}

func factKnownTo(facts []storyFact) func(string) bool {
	set := make(map[string]bool, len(facts))
	for _, f := range facts {
		set[f.ID] = true
	}
	return func(id string) bool { return set[id] }
}

func toSet(ids []string) map[string]bool {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// factsText lists the kept facts for a prompt: code, kind, whether it must be
// stated, whether it has a source, and the claim.
func factsText(facts []storyFact) string {
	if len(facts) == 0 {
		return "(chưa có dữ kiện nào)"
	}
	var sb strings.Builder
	for i, f := range facts {
		if i > 0 {
			sb.WriteString("\n")
		}
		tags := []string{string(f.Type)}
		if f.Must {
			tags = append(tags, "bắt buộc nói")
		}
		if f.Link.State == domain.LinkNone {
			tags = append(tags, "chưa có nguồn")
		}
		sb.WriteString(f.ID + " [" + strings.Join(tags, " · ") + "]: " + f.ClaimVI)
		if f.Note != "" {
			sb.WriteString(" (ghi chú: " + f.Note + ")")
		}
	}
	return sb.String()
}

// mergeFactCodes returns the codes of a followed by those of b that a lacks.
func mergeFactCodes(a, b []string) []string {
	seen := toSet(a)
	out := append([]string{}, a...)
	for _, id := range b {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
