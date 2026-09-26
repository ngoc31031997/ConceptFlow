package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"authoring/internal/domain"
)

var (
	ErrArchetypeNotFound  = errors.New("video archetype not found")
	ErrArchetypeReadOnly  = errors.New("system video archetype is read-only")
	ErrArchetypeCodeTaken = errors.New("video archetype code is already used")
)

// VideoArchetypePort is the persistence the archetype table needs (CR-041).
type VideoArchetypePort interface {
	ListArchetypes(ctx context.Context) ([]domain.VideoArchetype, error)
	GetArchetype(ctx context.Context, id string) (domain.VideoArchetype, error)
	CreateArchetype(ctx context.Context, a domain.VideoArchetype) (domain.VideoArchetype, error)
	UpdateArchetype(ctx context.Context, a domain.VideoArchetype) (domain.VideoArchetype, error)
	DeleteArchetype(ctx context.Context, id string) error
}

// VideoArchetypesUseCase manages the kinds of video the Creator can choose from.
type VideoArchetypesUseCase struct{ repo VideoArchetypePort }

func NewVideoArchetypesUseCase(repo VideoArchetypePort) *VideoArchetypesUseCase {
	return &VideoArchetypesUseCase{repo: repo}
}

func (uc *VideoArchetypesUseCase) List(ctx context.Context) ([]domain.VideoArchetype, error) {
	return uc.repo.ListArchetypes(ctx)
}

// maxArchetypeCodeLen keeps the code short enough to type as "kiểu: X" and to
// read in the dòng KIỂU VIDEO.
const maxArchetypeCodeLen = 8

func normalize(a domain.VideoArchetype) (domain.VideoArchetype, error) {
	a.Code = strings.ToUpper(strings.TrimSpace(a.Code))
	a.Name = strings.TrimSpace(a.Name)
	a.WhenToUse = strings.TrimSpace(a.WhenToUse)
	a.Playbook = strings.TrimSpace(a.Playbook)
	switch {
	case a.Name == "":
		return a, fmt.Errorf("name is required")
	case a.WhenToUse == "":
		return a, fmt.Errorf("when_to_use is required — the model picks a kind by it")
	case a.Playbook == "":
		return a, fmt.Errorf("playbook is required")
	case len([]rune(a.Code)) > maxArchetypeCodeLen:
		return a, fmt.Errorf("code must be at most %d characters", maxArchetypeCodeLen)
	case strings.ContainsAny(a.Code, " \t\n:—"):
		return a, fmt.Errorf("code must not contain spaces, ':' or '—'")
	}
	return a, nil
}

// nextCode is the first unused capital letter, so a Creator who leaves the code
// blank gets E after A–D.
func nextCode(existing []domain.VideoArchetype) (string, error) {
	used := map[string]bool{}
	for _, e := range existing {
		used[strings.ToUpper(e.Code)] = true
	}
	for c := 'A'; c <= 'Z'; c++ {
		if !used[string(c)] {
			return string(c), nil
		}
	}
	return "", fmt.Errorf("all single-letter codes are used — give this kind a code")
}

func (uc *VideoArchetypesUseCase) Create(ctx context.Context, a domain.VideoArchetype) (domain.VideoArchetype, error) {
	a, err := normalize(a)
	if err != nil {
		return a, err
	}
	if a.Code == "" {
		existing, err := uc.repo.ListArchetypes(ctx)
		if err != nil {
			return a, err
		}
		if a.Code, err = nextCode(existing); err != nil {
			return a, err
		}
	}
	return uc.repo.CreateArchetype(ctx, a)
}

// Copy duplicates any row — including a system one — into an editable row with
// the next free code.
func (uc *VideoArchetypesUseCase) Copy(ctx context.Context, id string) (domain.VideoArchetype, error) {
	src, err := uc.repo.GetArchetype(ctx, id)
	if err != nil {
		return src, err
	}
	src.Name = "Bản sao của " + src.Name
	src.Code, src.IsSystem = "", false
	return uc.Create(ctx, src)
}

func (uc *VideoArchetypesUseCase) Update(ctx context.Context, id string, a domain.VideoArchetype) (domain.VideoArchetype, error) {
	existing, err := uc.repo.GetArchetype(ctx, id)
	if err != nil {
		return existing, err
	}
	if existing.IsSystem {
		return existing, ErrArchetypeReadOnly
	}
	a.ID = id
	if a.Code == "" {
		a.Code = existing.Code
	}
	if a, err = normalize(a); err != nil {
		return a, err
	}
	return uc.repo.UpdateArchetype(ctx, a)
}

func (uc *VideoArchetypesUseCase) Delete(ctx context.Context, id string) error {
	existing, err := uc.repo.GetArchetype(ctx, id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return ErrArchetypeReadOnly
	}
	return uc.repo.DeleteArchetype(ctx, id)
}
