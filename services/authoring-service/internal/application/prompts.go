package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"authoring/internal/domain"
)

var (
	// ErrPromptNotFound — no prompt row has this id.
	ErrPromptNotFound = errors.New("prompt not found")
	// ErrPromptReadOnly — the row ships with the system and cannot be edited
	// or deleted. Copy it to get a row that can.
	ErrPromptReadOnly = errors.New("system prompt is read-only")
)

// PromptPort is the persistence capability the prompt library needs (CR-031).
type PromptPort interface {
	// GetActive returns the row the pipeline should render for a role: the
	// active one, else the role's system row.
	GetActive(ctx context.Context, role domain.PromptRole) (domain.Prompt, error)
	// List returns every row, or just one role's when role is not empty.
	List(ctx context.Context, role domain.PromptRole) ([]domain.Prompt, error)
	Get(ctx context.Context, id string) (domain.Prompt, error)
	// Create inserts a Creator's row, inactive.
	Create(ctx context.Context, role domain.PromptRole, name, templateText string) (domain.Prompt, error)
	Update(ctx context.Context, id, name, templateText string) (domain.Prompt, error)
	Delete(ctx context.Context, id string) error
	// Activate makes one row the role's only active row.
	Activate(ctx context.Context, id string) (domain.Prompt, error)
	GetSystem(ctx context.Context, role domain.PromptRole) (domain.Prompt, error)
}

// PromptsUseCase backs the prompt library: a list of prompts per role, one of
// them active (CR-031).
type PromptsUseCase struct {
	prompts PromptPort
}

func NewPromptsUseCase(prompts PromptPort) *PromptsUseCase {
	return &PromptsUseCase{prompts: prompts}
}

// Active returns the wording the pipeline will actually use for a role.
func (uc *PromptsUseCase) Active(ctx context.Context, role domain.PromptRole) (domain.Prompt, error) {
	if !domain.ValidPromptRole(string(role)) {
		return domain.Prompt{}, fmt.Errorf("invalid role %q", role)
	}
	return uc.prompts.GetActive(ctx, role)
}

// List returns the library, optionally narrowed to one role.
func (uc *PromptsUseCase) List(ctx context.Context, role domain.PromptRole) ([]domain.Prompt, error) {
	if role != "" && !domain.ValidPromptRole(string(role)) {
		return nil, fmt.Errorf("invalid role %q", role)
	}
	return uc.prompts.List(ctx, role)
}

func validNameAndText(name, templateText string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	if strings.TrimSpace(templateText) == "" {
		return "", fmt.Errorf("template_text is required")
	}
	return name, nil
}

// Create adds a Creator-owned prompt. It starts inactive: adding a row must
// not change what the pipeline runs — activating is its own deliberate action.
func (uc *PromptsUseCase) Create(ctx context.Context, role domain.PromptRole, name, templateText string) (domain.Prompt, error) {
	if !domain.ValidPromptRole(string(role)) {
		return domain.Prompt{}, fmt.Errorf("invalid role %q", role)
	}
	name, err := validNameAndText(name, templateText)
	if err != nil {
		return domain.Prompt{}, err
	}
	return uc.prompts.Create(ctx, role, name, templateText)
}

// Copy duplicates any row — including a system one, which is the only way to
// start from it — into a new Creator-owned, inactive row.
func (uc *PromptsUseCase) Copy(ctx context.Context, id string) (domain.Prompt, error) {
	src, err := uc.prompts.Get(ctx, id)
	if err != nil {
		return domain.Prompt{}, err
	}
	return uc.prompts.Create(ctx, src.Role, "Bản sao của "+src.Name, src.TemplateText)
}

// Update edits a Creator-owned prompt.
func (uc *PromptsUseCase) Update(ctx context.Context, id, name, templateText string) (domain.Prompt, error) {
	existing, err := uc.prompts.Get(ctx, id)
	if err != nil {
		return domain.Prompt{}, err
	}
	if existing.IsSystem {
		return domain.Prompt{}, ErrPromptReadOnly
	}
	name, err = validNameAndText(name, templateText)
	if err != nil {
		return domain.Prompt{}, err
	}
	return uc.prompts.Update(ctx, id, name, templateText)
}

// Activate makes a row the one its role runs on. System rows can be activated
// too — that is how a Creator goes back to the default.
func (uc *PromptsUseCase) Activate(ctx context.Context, id string) (domain.Prompt, error) {
	return uc.prompts.Activate(ctx, id)
}

// Delete removes a Creator-owned prompt. Deleting the active one hands the
// role back to its system prompt, so a role is never left without wording.
func (uc *PromptsUseCase) Delete(ctx context.Context, id string) error {
	existing, err := uc.prompts.Get(ctx, id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return ErrPromptReadOnly
	}
	if err := uc.prompts.Delete(ctx, id); err != nil {
		return err
	}
	if existing.IsActive {
		system, err := uc.prompts.GetSystem(ctx, existing.Role)
		if err != nil {
			return err
		}
		_, err = uc.prompts.Activate(ctx, system.ID)
		return err
	}
	return nil
}
