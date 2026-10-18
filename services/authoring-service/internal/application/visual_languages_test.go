package application

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"authoring/internal/domain"
)

// memLanguageRepo is an in-memory language store with the shipped language active.
type memLanguageRepo struct {
	langs   map[string]domain.VisualLanguage
	samples map[string]domain.LanguageSample // kit|scene|frame
	pinned  map[string]string                // project -> language id
	nextID  int
}

func newMemLanguageRepo() *memLanguageRepo {
	r := &memLanguageRepo{langs: map[string]domain.VisualLanguage{}, samples: map[string]domain.LanguageSample{}, pinned: map[string]string{}}
	r.langs[domain.PhatsangID] = domain.VisualLanguage{
		ID: domain.PhatsangID, Name: "Khoa học phát sáng", KitKey: domain.PhatsangKitKey,
		IsSystem: true, IsActive: true, Version: 1, Rules: domain.PhatsangRules(),
	}
	return r
}

func (r *memLanguageRepo) ListVisualLanguages(context.Context) ([]domain.VisualLanguage, error) {
	var out []domain.VisualLanguage
	for _, l := range r.langs {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *memLanguageRepo) GetVisualLanguage(_ context.Context, id string) (domain.VisualLanguage, error) {
	l, ok := r.langs[id]
	if !ok {
		return l, domain.ErrVisualLanguageNotFound
	}
	return l, nil
}

func (r *memLanguageRepo) ActiveVisualLanguage(context.Context) (domain.VisualLanguage, error) {
	for _, l := range r.langs {
		if l.IsActive {
			return l, nil
		}
	}
	return domain.VisualLanguage{}, domain.ErrNoActiveVisualLanguage
}

func (r *memLanguageRepo) CreateVisualLanguage(_ context.Context, l domain.VisualLanguage) (domain.VisualLanguage, error) {
	for _, other := range r.langs {
		if strings.EqualFold(other.Name, l.Name) {
			return l, domain.ErrVisualLanguageNameTaken
		}
	}
	r.nextID++
	l.ID = "lang-" + string(rune('a'+r.nextID))
	l.Version, l.IsSystem, l.IsActive = 1, false, false
	r.langs[l.ID] = l
	return l, nil
}

func (r *memLanguageRepo) AddVisualLanguageVersion(_ context.Context, l domain.VisualLanguage) (domain.VisualLanguage, error) {
	cur := r.langs[l.ID]
	l.Version, l.IsSystem, l.IsActive, l.KitKey = cur.Version+1, cur.IsSystem, cur.IsActive, cur.KitKey
	r.langs[l.ID] = l
	return l, nil
}

func (r *memLanguageRepo) ActivateVisualLanguage(_ context.Context, id string) error {
	for k, l := range r.langs {
		l.IsActive = k == id
		r.langs[k] = l
	}
	return nil
}

func (r *memLanguageRepo) DeleteVisualLanguage(_ context.Context, id string) error {
	if r.langs[id].IsActive {
		return domain.ErrVisualLanguageActive
	}
	for _, lang := range r.pinned {
		if lang == id {
			return domain.ErrVisualLanguageInUse
		}
	}
	delete(r.langs, id)
	return nil
}

func (r *memLanguageRepo) PinnedLanguage(_ context.Context, projectID string) (domain.PinnedLanguage, error) {
	id, ok := r.pinned[projectID]
	if !ok {
		return domain.PinnedLanguage{}, domain.ErrProjectNotPinned
	}
	l := r.langs[id]
	return domain.PinnedLanguage{ID: l.ID, Version: l.Version, KitKey: l.KitKey, Name: l.Name, Rules: l.Rules}, nil
}

func (r *memLanguageRepo) ListLanguageSamples(_ context.Context, kitKey string) ([]domain.LanguageSample, error) {
	var out []domain.LanguageSample
	for _, s := range r.samples {
		if s.KitKey == kitKey {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *memLanguageRepo) SaveLanguageSample(_ context.Context, s domain.LanguageSample) error {
	r.samples[s.KitKey+"|"+s.Scene+"|"+s.Frame] = s
	return nil
}

// fakeStills renders sample stills by name and counts what it was asked for.
type fakeStills struct {
	kits     map[string]int
	rendered []string
	pruned   struct {
		keep []string
		age  float64
	}
	err error
}

func (f *fakeStills) Kits(context.Context) (map[string]int, error) { return f.kits, f.err }
func (f *fakeStills) Still(_ context.Context, composition, kitKey string) (string, bool, error) {
	if f.err != nil {
		return "", false, f.err
	}
	f.rendered = append(f.rendered, composition)
	return kitKey + "-" + composition + ".png", false, nil
}
func (f *fakeStills) Prune(_ context.Context, keep []string, minAgeHours float64) ([]string, error) {
	f.pruned.keep, f.pruned.age = keep, minAgeHours
	return []string{"old.png"}, nil
}

func TestACopyIsAnEditableLanguageWithAUniqueName(t *testing.T) {
	ctx := context.Background()
	uc := NewVisualLanguagesUseCase(newMemLanguageRepo(), nil)
	first, err := uc.Copy(ctx, domain.PhatsangID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := uc.Copy(ctx, domain.PhatsangID)
	if err != nil {
		t.Fatal(err)
	}
	if first.IsSystem || first.IsActive || first.Rules != domain.PhatsangRules() || first.KitKey != domain.PhatsangKitKey {
		t.Fatalf("copy: %+v", first)
	}
	if first.Name == second.Name {
		t.Fatalf("two copies share the name %q", first.Name)
	}
}

func TestEditingALanguageAddsAVersionAndTheShippedOneIsReadOnly(t *testing.T) {
	ctx := context.Background()
	repo := newMemLanguageRepo()
	uc := NewVisualLanguagesUseCase(repo, nil)
	copyOf, _ := uc.Copy(ctx, domain.PhatsangID)
	rules := domain.PhatsangRules()
	rules.Scene = "- [C1] luật cảnh mới"
	out, err := uc.Update(ctx, copyOf.ID, VisualLanguageInput{Name: "Của tôi", Rules: rules})
	if err != nil {
		t.Fatal(err)
	}
	if out.Version != copyOf.Version+1 || out.Rules.Scene != "- [C1] luật cảnh mới" || out.Name != "Của tôi" {
		t.Fatalf("update: %+v", out)
	}
	if _, err := uc.Update(ctx, domain.PhatsangID, VisualLanguageInput{Name: "x", Rules: rules}); !errors.Is(err, domain.ErrVisualLanguageReadOnly) {
		t.Fatalf("the shipped language must be read-only, got %v", err)
	}
	rules.Backdrop = " "
	if _, err := uc.Update(ctx, copyOf.ID, VisualLanguageInput{Name: "Của tôi", Rules: rules}); !errors.Is(err, domain.ErrInvalidVisualLanguage) {
		t.Fatalf("empty rules must be refused, got %v", err)
	}
	if _, err := uc.Update(ctx, copyOf.ID, VisualLanguageInput{Name: " ", Rules: domain.PhatsangRules()}); !errors.Is(err, domain.ErrInvalidVisualLanguage) {
		t.Fatalf("an empty name must be refused, got %v", err)
	}
}

func TestOnlyOneLanguageIsActiveAndAnActiveOrPinnedOneCannotBeDeleted(t *testing.T) {
	ctx := context.Background()
	repo := newMemLanguageRepo()
	uc := NewVisualLanguagesUseCase(repo, nil)
	mine, _ := uc.Copy(ctx, domain.PhatsangID)
	if err := uc.Activate(ctx, mine.ID); err != nil {
		t.Fatal(err)
	}
	active, _ := uc.Active(ctx)
	if active.ID != mine.ID {
		t.Fatalf("active is %q", active.ID)
	}
	var count int
	all, _ := uc.List(ctx)
	for _, l := range all {
		if l.IsActive {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("%d languages are active", count)
	}
	if err := uc.Delete(ctx, mine.ID); !errors.Is(err, domain.ErrVisualLanguageActive) {
		t.Fatalf("deleting the active one: %v", err)
	}
	if err := uc.Activate(ctx, domain.PhatsangID); err != nil {
		t.Fatal(err)
	}
	repo.pinned["p1"] = mine.ID
	if err := uc.Delete(ctx, mine.ID); !errors.Is(err, domain.ErrVisualLanguageInUse) {
		t.Fatalf("deleting a pinned one: %v", err)
	}
	delete(repo.pinned, "p1")
	if err := uc.Delete(ctx, mine.ID); err != nil {
		t.Fatal(err)
	}
	if err := uc.Delete(ctx, domain.PhatsangID); !errors.Is(err, domain.ErrVisualLanguageReadOnly) {
		t.Fatalf("deleting the shipped one: %v", err)
	}
	if err := uc.Activate(ctx, "nope"); !errors.Is(err, domain.ErrVisualLanguageNotFound) {
		t.Fatalf("activating a missing one: %v", err)
	}
}

func TestAProjectKeepsTheLanguageVersionItPinned(t *testing.T) {
	ctx := context.Background()
	repo := newMemLanguageRepo()
	uc := NewVisualLanguagesUseCase(repo, nil)
	if _, err := uc.Pinned(ctx, "p1"); !errors.Is(err, domain.ErrProjectNotPinned) {
		t.Fatalf("a project with no pin must be an error, got %v", err)
	}
	repo.pinned["p1"] = domain.PhatsangID
	pinned, err := uc.Pinned(ctx, "p1")
	if err != nil || pinned.KitKey != domain.PhatsangKitKey || pinned.Version != 1 {
		t.Fatalf("pinned: %+v, %v", pinned, err)
	}
}

func TestRebuildingSamplesRendersEveryScenePerFrameForTheKitVersion(t *testing.T) {
	ctx := context.Background()
	repo := newMemLanguageRepo()
	stills := &fakeStills{kits: map[string]int{domain.PhatsangKitKey: 3}}
	uc := NewVisualLanguagesUseCase(repo, stills)
	out, err := uc.RebuildSamples(ctx, domain.PhatsangID)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"sample-fig-wide", "sample-fig-tall", "sample-space-wide", "sample-space-tall"}
	if strings.Join(stills.rendered, ",") != strings.Join(want, ",") {
		t.Fatalf("rendered %v, want %v", stills.rendered, want)
	}
	if len(out) != 4 || out[0].KitVersion != 3 || !strings.HasSuffix(out[0].File, ".png") {
		t.Fatalf("samples: %+v", out)
	}
	saved, _ := uc.Samples(ctx, domain.PhatsangID)
	if len(saved) != 4 {
		t.Fatalf("recorded %d samples", len(saved))
	}
}

func TestSamplesAreRebuiltOnlyWhenMissingOrOutOfDateWithTheKit(t *testing.T) {
	ctx := context.Background()
	repo := newMemLanguageRepo()
	stills := &fakeStills{kits: map[string]int{domain.PhatsangKitKey: 1}}
	uc := NewVisualLanguagesUseCase(repo, stills)
	if err := uc.EnsureSamples(ctx); err != nil || len(stills.rendered) != 4 {
		t.Fatalf("first run: %v, rendered %d", err, len(stills.rendered))
	}
	if err := uc.EnsureSamples(ctx); err != nil || len(stills.rendered) != 4 {
		t.Fatalf("current samples were rendered again: %v, rendered %d", err, len(stills.rendered))
	}
	stills.kits[domain.PhatsangKitKey] = 2
	if err := uc.EnsureSamples(ctx); err != nil || len(stills.rendered) != 8 {
		t.Fatalf("a new kit version must re-render: %v, rendered %d", err, len(stills.rendered))
	}
}

func TestAKitRenderingDoesNotHaveIsAnError(t *testing.T) {
	ctx := context.Background()
	stills := &fakeStills{kits: map[string]int{"other": 1}}
	uc := NewVisualLanguagesUseCase(newMemLanguageRepo(), stills)
	if _, err := uc.RebuildSamples(ctx, domain.PhatsangID); !errors.Is(err, domain.ErrUnknownKitKey) {
		t.Fatalf("unknown kit: %v", err)
	}
	stills.err = errors.New("rendering down")
	if err := uc.EnsureSamples(ctx); err == nil || !strings.Contains(err.Error(), "rendering down") {
		t.Fatalf("an unreachable rendering must be reported, got %v", err)
	}
	if _, err := NewVisualLanguagesUseCase(newMemLanguageRepo(), nil).RebuildSamples(ctx, domain.PhatsangID); err == nil {
		t.Fatal("no stills port must be an error")
	}
}

func TestPruningKeepsTheStillsOfEveryLanguagePage(t *testing.T) {
	ctx := context.Background()
	repo := newMemLanguageRepo()
	stills := &fakeStills{kits: map[string]int{domain.PhatsangKitKey: 1}}
	uc := NewVisualLanguagesUseCase(repo, stills)
	if _, err := uc.RebuildSamples(ctx, domain.PhatsangID); err != nil {
		t.Fatal(err)
	}
	removed, err := uc.PruneStills(ctx, 36*time.Hour)
	if err != nil || len(removed) != 1 {
		t.Fatalf("prune: %v, %v", removed, err)
	}
	if len(stills.pruned.keep) != 4 || stills.pruned.age != 36 {
		t.Fatalf("prune asked to keep %v older than %v h", stills.pruned.keep, stills.pruned.age)
	}
}
