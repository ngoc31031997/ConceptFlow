package application

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"sort"
	"strings"
	"sync"

	"authoring/internal/domain"
)

// checks that write to it while a test reads.
type briefUnits struct {
	mu    sync.Mutex
	units map[string]domain.ReviewUnit
	saves int
}

func newBriefUnits() *briefUnits { return &briefUnits{units: map[string]domain.ReviewUnit{}} }

func (b *briefUnits) GetUnit(_ context.Context, projectID, kind, key string) (domain.ReviewUnit, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.get(projectID, kind, key)
}

// get reads a unit; the caller holds the lock.
func (b *briefUnits) get(projectID, kind, key string) (domain.ReviewUnit, error) {
	u, ok := b.units[projectID+"|"+kind+"|"+key]
	if !ok {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	u.Stale = u.InputRef != "" && u.InputSHA != b.sha(projectID, u.InputRef)
	return u, nil
}

// sha hashes the content of the unit named "kind:key", as the store does.
func (b *briefUnits) sha(projectID, ref string) string {
	u, ok := b.units[projectID+"|"+strings.Replace(ref, ":", "|", 1)]
	if !ok {
		return ""
	}
	sum := md5.Sum(u.Content)
	return hex.EncodeToString(sum[:])
}

func (b *briefUnits) UnitSHA(_ context.Context, projectID, kind, key string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sha(projectID, kind+":"+key), nil
}

func (b *briefUnits) ListUnits(_ context.Context, projectID, kind string) ([]domain.ReviewUnit, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []domain.ReviewUnit
	for k := range b.units {
		parts := strings.SplitN(k, "|", 3)
		if parts[0] == projectID && parts[1] == kind {
			u, _ := b.get(projectID, kind, parts[2])
			out = append(out, u)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Key) != len(out[j].Key) {
			return len(out[i].Key) < len(out[j].Key)
		}
		return out[i].Key < out[j].Key
	})
	return out, nil
}

func (b *briefUnits) DeleteUnit(_ context.Context, projectID, kind, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.units, projectID+"|"+kind+"|"+key)
	return nil
}

func (b *briefUnits) SaveUnit(_ context.Context, u domain.ReviewUnit) (domain.ReviewUnit, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.saves++
	if u.CreatorFields == nil {
		u.CreatorFields = []string{}
	}
	b.units[u.ProjectID+"|"+u.Kind+"|"+u.Key] = u
	return u, nil
}

// briefStore is an in-memory store of the topic and the model choices.
type briefStore struct {
	topic  string
	models domain.AuthoringStepModels
}

func (s *briefStore) GetAuthoringTopic(context.Context, string) (string, error) { return s.topic, nil }
func (s *briefStore) GetAuthoringModels(context.Context, string) (domain.AuthoringStepModels, error) {
	return s.models, nil
}

// briefProject answers both ProjectReaderPort and AuthoringLockPort.
type briefProject struct {
	project domain.Project
	status  domain.ProjectStatus
}

func (p *briefProject) Get(context.Context, string) (*domain.Project, error) {
	project := p.project
	return &project, nil
}
func (p *briefProject) GetStatus(context.Context, string) (domain.ProjectStatus, error) {
	return p.status, nil
}

// briefPrompts serves the shipped prompts, plus a story prompt written by the
// Creator when story is set (no story prompt ships).
type briefPrompts struct{ story string }

func (b briefPrompts) GetActive(_ context.Context, role domain.PromptRole) (domain.Prompt, error) {
	if role == domain.RoleStoryArchitect && b.story != "" {
		return domain.Prompt{Role: role, TemplateText: b.story}, nil
	}
	for _, p := range domain.SystemPrompts() {
		if p.Role == role {
			return p, nil
		}
	}
	return domain.Prompt{}, ErrPromptNotFound
}

// briefLLM replies with a fixed text and keeps the prompts it was sent.
type briefLLM struct {
	reply   string
	err     error
	systems []string
	models  []string
}

func (l *briefLLM) Name() string { return "fake" }
func (l *briefLLM) Chat(_ context.Context, req ChatRequest) (ChatResult, error) {
	l.systems = append(l.systems, req.System)
	l.models = append(l.models, req.Model)
	if l.err != nil {
		return ChatResult{}, l.err
	}
	return llmServiceReply(req, l.reply)
}
