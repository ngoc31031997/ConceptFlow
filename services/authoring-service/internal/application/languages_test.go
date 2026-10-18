package application

import (
	"context"

	"authoring/internal/domain"
)

// fakeLanguages is a language store with the shipped language active. A
// project is pinned to it unless `unpinned` names it; `pinned` overrides the
// pin of a project.
type fakeLanguages struct {
	active   domain.VisualLanguage
	pinned   map[string]domain.PinnedLanguage
	unpinned map[string]bool
	err      error
}

func newFakeLanguages() *fakeLanguages {
	return &fakeLanguages{
		active: domain.VisualLanguage{
			ID: domain.PhatsangID, Name: "Khoa học phát sáng", Description: "ánh sáng kể chuyện",
			KitKey: domain.PhatsangKitKey, IsSystem: true, IsActive: true, Version: 1, Rules: domain.PhatsangRules(),
		},
		pinned:   map[string]domain.PinnedLanguage{},
		unpinned: map[string]bool{},
	}
}

func (f *fakeLanguages) Active(context.Context) (domain.VisualLanguage, error) {
	return f.active, f.err
}

func (f *fakeLanguages) Pinned(_ context.Context, projectID string) (domain.PinnedLanguage, error) {
	if f.err != nil {
		return domain.PinnedLanguage{}, f.err
	}
	if p, ok := f.pinned[projectID]; ok {
		return p, nil
	}
	if f.unpinned[projectID] {
		return domain.PinnedLanguage{}, domain.ErrProjectNotPinned
	}
	return domain.PinnedLanguage{ID: f.active.ID, Version: f.active.Version, KitKey: f.active.KitKey,
		Name: f.active.Name, Rules: f.active.Rules}, nil
}
