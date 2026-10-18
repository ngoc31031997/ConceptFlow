package application_test

import (
	"context"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// testLanguages pins every project to the shipped language, which is also the active one.
type testLanguages struct{}

func testLanguagePort() application.LanguagePort { return testLanguages{} }

func (testLanguages) Active(context.Context) (domain.VisualLanguage, error) {
	return domain.VisualLanguage{
		ID: domain.PhatsangID, Name: "Khoa học phát sáng", KitKey: domain.PhatsangKitKey,
		IsSystem: true, IsActive: true, Version: 1, Rules: domain.PhatsangRules(),
	}, nil
}

func (l testLanguages) Pinned(ctx context.Context, _ string) (domain.PinnedLanguage, error) {
	a, _ := l.Active(ctx)
	return domain.PinnedLanguage{ID: a.ID, Version: a.Version, KitKey: a.KitKey, Name: a.Name, Rules: a.Rules}, nil
}
