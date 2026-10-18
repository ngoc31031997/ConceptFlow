package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"authoring/internal/domain"
)

// VisualLanguageRepoPort is the language store's persistence.
type VisualLanguageRepoPort interface {
	ListVisualLanguages(ctx context.Context) ([]domain.VisualLanguage, error)
	// GetVisualLanguage returns a language at its newest version.
	GetVisualLanguage(ctx context.Context, id string) (domain.VisualLanguage, error)
	// ActiveVisualLanguage returns the one active language (ErrNoActiveVisualLanguage if none).
	ActiveVisualLanguage(ctx context.Context) (domain.VisualLanguage, error)
	// CreateVisualLanguage stores a new language at version 1 (ErrVisualLanguageNameTaken
	// when the name is used).
	CreateVisualLanguage(ctx context.Context, l domain.VisualLanguage) (domain.VisualLanguage, error)
	// AddVisualLanguageVersion stores edited fields and rules as the next version.
	AddVisualLanguageVersion(ctx context.Context, l domain.VisualLanguage) (domain.VisualLanguage, error)
	// ActivateVisualLanguage makes id the only active language, in one transaction.
	ActivateVisualLanguage(ctx context.Context, id string) error
	// DeleteVisualLanguage deletes a Creator language that is neither active nor
	// pinned by a project (ErrVisualLanguageActive, ErrVisualLanguageInUse).
	DeleteVisualLanguage(ctx context.Context, id string) error
	// PinnedLanguage is the language version a project is pinned to, with that
	// version's rules (ErrProjectNotPinned when the project has none).
	PinnedLanguage(ctx context.Context, projectID string) (domain.PinnedLanguage, error)
	ListLanguageSamples(ctx context.Context, kitKey string) ([]domain.LanguageSample, error)
	SaveLanguageSample(ctx context.Context, s domain.LanguageSample) error
}

// StillsPort is the rendering service's shared stills: the kits it can draw
// with and sample scenes rendered once and cached.
type StillsPort interface {
	// Kits returns the version of every kit the rendering service draws with.
	Kits(ctx context.Context) (map[string]int, error)
	// Still renders (or finds cached) one sample composition and returns its file name.
	Still(ctx context.Context, composition, kitKey string) (file string, cacheHit bool, err error)
	// Prune deletes stills not in keep and older than minAgeHours; returns the names removed.
	Prune(ctx context.Context, keep []string, minAgeHours float64) ([]string, error)
}

// VisualLanguageInput is what a Creator edits on a language of their own.
type VisualLanguageInput struct {
	Name        string
	Note        string
	Description string
	Rules       domain.VisualLanguageRules
}

// VisualLanguagesUseCase manages the language store, the project pin and the
// sample scenes of a language.
type VisualLanguagesUseCase struct {
	repo   VisualLanguageRepoPort
	stills StillsPort // nil = samples cannot be rendered
}

func NewVisualLanguagesUseCase(repo VisualLanguageRepoPort, stills StillsPort) *VisualLanguagesUseCase {
	return &VisualLanguagesUseCase{repo: repo, stills: stills}
}

func (uc *VisualLanguagesUseCase) List(ctx context.Context) ([]domain.VisualLanguage, error) {
	return uc.repo.ListVisualLanguages(ctx)
}

func (uc *VisualLanguagesUseCase) Get(ctx context.Context, id string) (domain.VisualLanguage, error) {
	return uc.repo.GetVisualLanguage(ctx, id)
}

// Active returns the active language.
func (uc *VisualLanguagesUseCase) Active(ctx context.Context) (domain.VisualLanguage, error) {
	return uc.repo.ActiveVisualLanguage(ctx)
}

// Activate switches the active language. Projects already created keep the
// language they pinned.
func (uc *VisualLanguagesUseCase) Activate(ctx context.Context, id string) error {
	if _, err := uc.repo.GetVisualLanguage(ctx, id); err != nil {
		return err
	}
	return uc.repo.ActivateVisualLanguage(ctx, id)
}

// Copy makes an editable language of the Creator from any language.
func (uc *VisualLanguagesUseCase) Copy(ctx context.Context, id string) (domain.VisualLanguage, error) {
	src, err := uc.repo.GetVisualLanguage(ctx, id)
	if err != nil {
		return domain.VisualLanguage{}, err
	}
	clone := domain.VisualLanguage{
		Name: src.Name + " (bản sao)", Note: src.Note, Description: src.Description,
		KitKey: src.KitKey, Rules: src.Rules,
	}
	for n := 2; ; n++ {
		out, err := uc.repo.CreateVisualLanguage(ctx, clone)
		if errors.Is(err, domain.ErrVisualLanguageNameTaken) {
			clone.Name = fmt.Sprintf("%s (bản sao %d)", src.Name, n)
			continue
		}
		return out, err
	}
}

// Update stores the edited fields of a Creator language as a new version; a
// project that pinned an earlier version keeps it.
func (uc *VisualLanguagesUseCase) Update(ctx context.Context, id string, in VisualLanguageInput) (domain.VisualLanguage, error) {
	cur, err := uc.repo.GetVisualLanguage(ctx, id)
	if err != nil {
		return domain.VisualLanguage{}, err
	}
	if cur.IsSystem {
		return domain.VisualLanguage{}, domain.ErrVisualLanguageReadOnly
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return domain.VisualLanguage{}, fmt.Errorf("%w: thiếu tên", domain.ErrInvalidVisualLanguage)
	}
	if err := in.Rules.Validate(); err != nil {
		return domain.VisualLanguage{}, err
	}
	cur.Name, cur.Note, cur.Description, cur.Rules = name, strings.TrimSpace(in.Note), strings.TrimSpace(in.Description), in.Rules
	return uc.repo.AddVisualLanguageVersion(ctx, cur)
}

// Delete removes a Creator language that no project pinned and that is not active.
func (uc *VisualLanguagesUseCase) Delete(ctx context.Context, id string) error {
	cur, err := uc.repo.GetVisualLanguage(ctx, id)
	if err != nil {
		return err
	}
	if cur.IsSystem {
		return domain.ErrVisualLanguageReadOnly
	}
	return uc.repo.DeleteVisualLanguage(ctx, id)
}

// Pinned is the language version a project is pinned to; a project with none
// is an error, never a silent fall back to the active language.
func (uc *VisualLanguagesUseCase) Pinned(ctx context.Context, projectID string) (domain.PinnedLanguage, error) {
	return uc.repo.PinnedLanguage(ctx, projectID)
}

// Samples returns the rendered samples of a language's kit.
func (uc *VisualLanguagesUseCase) Samples(ctx context.Context, id string) ([]domain.LanguageSample, error) {
	l, err := uc.repo.GetVisualLanguage(ctx, id)
	if err != nil {
		return nil, err
	}
	return uc.repo.ListLanguageSamples(ctx, l.KitKey)
}

// RebuildSamples renders every sample scene of the language's kit (each is
// cached by the rendering service, so unchanged ones return at once) and
// records the files. It returns the samples now current.
func (uc *VisualLanguagesUseCase) RebuildSamples(ctx context.Context, id string) ([]domain.LanguageSample, error) {
	if uc.stills == nil {
		return nil, errors.New("rendering stills are not configured")
	}
	l, err := uc.repo.GetVisualLanguage(ctx, id)
	if err != nil {
		return nil, err
	}
	kits, err := uc.stills.Kits(ctx)
	if err != nil {
		return nil, fmt.Errorf("đọc bộ kit của rendering: %w", err)
	}
	version, ok := kits[l.KitKey]
	if !ok {
		return nil, fmt.Errorf("%w: %q", domain.ErrUnknownKitKey, l.KitKey)
	}
	var out []domain.LanguageSample
	for _, scene := range domain.SampleScenes {
		for _, frame := range domain.SampleFrames {
			file, _, err := uc.stills.Still(ctx, domain.SampleComposition(scene, frame), l.KitKey)
			if err != nil {
				return nil, fmt.Errorf("dựng ảnh mẫu %s/%s: %w", scene, frame, err)
			}
			s := domain.LanguageSample{KitKey: l.KitKey, KitVersion: version, Scene: scene, Frame: frame, File: file}
			if err := uc.repo.SaveLanguageSample(ctx, s); err != nil {
				return nil, err
			}
			out = append(out, s)
		}
	}
	return out, nil
}

// EnsureSamples renders the samples of every language whose kit has none, or
// whose recorded samples were made by an older kit version.
func (uc *VisualLanguagesUseCase) EnsureSamples(ctx context.Context) error {
	if uc.stills == nil {
		return nil
	}
	kits, err := uc.stills.Kits(ctx)
	if err != nil {
		return fmt.Errorf("đọc bộ kit của rendering: %w", err)
	}
	langs, err := uc.repo.ListVisualLanguages(ctx)
	if err != nil {
		return err
	}
	done := map[string]bool{}
	for _, l := range langs {
		if done[l.KitKey] {
			continue
		}
		done[l.KitKey] = true
		version, ok := kits[l.KitKey]
		if !ok {
			slog.Warn("visual language names a kit rendering does not have", "language", l.ID, "kit_key", l.KitKey)
			continue
		}
		have, err := uc.repo.ListLanguageSamples(ctx, l.KitKey)
		if err != nil {
			return err
		}
		if samplesCurrent(have, version) {
			continue
		}
		if _, err := uc.RebuildSamples(ctx, l.ID); err != nil {
			return fmt.Errorf("ngôn ngữ %s: %w", l.ID, err)
		}
	}
	return nil
}

func samplesCurrent(have []domain.LanguageSample, version int) bool {
	want := len(domain.SampleScenes) * len(domain.SampleFrames)
	current := 0
	for _, s := range have {
		if s.KitVersion == version {
			current++
		}
	}
	return current == want
}

// PruneStills deletes stills no language page shows that are older than minAge.
func (uc *VisualLanguagesUseCase) PruneStills(ctx context.Context, minAge time.Duration) ([]string, error) {
	if uc.stills == nil {
		return nil, nil
	}
	langs, err := uc.repo.ListVisualLanguages(ctx)
	if err != nil {
		return nil, err
	}
	var keep []string
	seen := map[string]bool{}
	for _, l := range langs {
		if seen[l.KitKey] {
			continue
		}
		seen[l.KitKey] = true
		samples, err := uc.repo.ListLanguageSamples(ctx, l.KitKey)
		if err != nil {
			return nil, err
		}
		for _, s := range samples {
			keep = append(keep, s.File)
		}
	}
	return uc.stills.Prune(ctx, keep, minAge.Hours())
}
