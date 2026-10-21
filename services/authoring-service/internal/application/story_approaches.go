package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"authoring/internal/domain"
)

// approachKey names the nth proposed approach.
func approachKey(n int) string { return "A" + strconv.Itoa(n) }

// approachNumber is the number in an approach key ("A3" is 3), or 0.
func approachNumber(key string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(key, "A"))
	if err != nil || !strings.HasPrefix(key, "A") {
		return 0
	}
	return n
}

// approachViews loads every proposed approach in key order.
func (uc *StoryDeskUseCase) approachViews(ctx context.Context, projectID, selected string) ([]StoryApproachView, error) {
	units, err := uc.units.ListUnits(ctx, projectID, domain.UnitKindStoryApproach)
	if err != nil {
		return nil, err
	}
	out := make([]StoryApproachView, 0, len(units))
	for _, u := range units {
		var a domain.StoryApproach
		if err := json.Unmarshal(u.Content, &a); err != nil {
			return nil, fmt.Errorf("read stored approach %s: %w", u.Key, err)
		}
		if a.Refs == nil {
			a.Refs = []string{}
		}
		out = append(out, StoryApproachView{Key: u.Key, StoryApproach: a, Selected: u.Key == selected})
	}
	return out, nil
}

// archetypesText lists the Kiểu video table for the approaches prompt.
func archetypesText(list []domain.VideoArchetype) string {
	if len(list) == 0 {
		return "(bảng chưa có kiểu nào)"
	}
	lines := make([]string, 0, len(list))
	for _, a := range list {
		lines = append(lines, a.Code+": "+a.Name+" — "+a.WhenToUse)
	}
	return strings.Join(lines, "\n")
}

// GenerateApproaches has the AI propose ways to tell the story from the brief
// and the fact table; note is the Creator's instruction for this run. The
// approach the Creator chose stays; the others are replaced by the new ones.
// A reply that cannot be read stores nothing.
func (uc *StoryDeskUseCase) GenerateApproaches(ctx context.Context, projectID, note string) ([]StoryApproachView, error) {
	if err := uc.requireAI(); err != nil {
		return nil, err
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return nil, err
	}
	sc, err := uc.context(ctx, projectID)
	if err != nil {
		return nil, err
	}
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleStoryApproaches)
	if err != nil {
		return nil, fmt.Errorf("load story approaches prompt: %w", err)
	}
	table, err := uc.archetypes.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("load video archetypes: %w", err)
	}
	system := domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"topic":          sc.topic,
		"brief_language": briefLanguageName(sc.language),
		"brief":          sc.brief,
		"facts":          factsText(sc.facts),
		"archetypes":     archetypesText(table),
		"refs_rule":      domain.StoryRefsRule(len(sc.facts) > 0),
		"refs_example":   domain.StoryRefsExample(len(sc.facts) > 0),
		"redo_note":      noteText(note),
	})
	reply, err := uc.chat(ctx, domain.RoleStoryApproaches, domain.UnitKindStoryApproach, projectID, sc.model, system)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(table))
	for _, a := range table {
		codes = append(codes, a.Code)
	}
	written, err := domain.ParseApproachesReply(reply, codes, factKnownTo(sc.facts))
	if err != nil {
		return nil, err
	}

	// The new approaches are stored before the old ones are removed, so a
	// failure part way leaves too many approaches rather than none.
	_, gate, err := uc.gate(ctx, projectID)
	if err != nil {
		return nil, err
	}
	existing, err := uc.approachViews(ctx, projectID, gate.ApproachKey)
	if err != nil {
		return nil, err
	}
	next := 1
	for _, a := range existing {
		if n := approachNumber(a.Key); n >= next {
			next = n + 1
		}
	}
	sha, err := uc.units.UnitSHA(ctx, projectID, domain.UnitKindBrief, domain.BriefUnitKey)
	if err != nil {
		return nil, err
	}
	for _, a := range written {
		raw, err := json.Marshal(a)
		if err != nil {
			return nil, err
		}
		if _, err := uc.units.SaveUnit(ctx, domain.ReviewUnit{
			ProjectID: projectID, Kind: domain.UnitKindStoryApproach, Key: approachKey(next),
			Status: domain.UnitPending, Content: raw, InputRef: storyBriefRef, InputSHA: sha,
		}); err != nil {
			return nil, err
		}
		next++
	}
	for _, a := range existing {
		if a.Selected {
			continue
		}
		if err := uc.units.DeleteUnit(ctx, projectID, domain.UnitKindStoryApproach, a.Key); err != nil {
			return nil, err
		}
	}
	return uc.approachViews(ctx, projectID, gate.ApproachKey)
}

// SelectApproach records the approach the story follows; "" lets the AI choose.
// Choosing sends an approved story back to review, since the story no longer
// follows the approach it was approved under. The fields are rewritten by a
// separate GenerateFrame run.
func (uc *StoryDeskUseCase) SelectApproach(ctx context.Context, projectID, key string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	if key != "" {
		if _, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindStoryApproach, key); err != nil {
			return err
		}
	}
	unit, gate, err := uc.gate(ctx, projectID)
	if err != nil {
		return err
	}
	if gate.ApproachKey == key {
		return nil
	}
	gate.ApproachKey = key
	unit.Status = domain.UnitPending
	_, err = uc.saveGate(ctx, unit, gate)
	return err
}

// SaveApproachAsArchetype saves a new kind of video the AI proposed into the
// Kiểu video table, so that it can be chosen for other videos too.
func (uc *StoryDeskUseCase) SaveApproachAsArchetype(ctx context.Context, projectID, key string) (StoryApproachView, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return StoryApproachView{}, err
	}
	unit, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindStoryApproach, key)
	if err != nil {
		return StoryApproachView{}, err
	}
	var a domain.StoryApproach
	if err := json.Unmarshal(unit.Content, &a); err != nil {
		return StoryApproachView{}, fmt.Errorf("read stored approach %s: %w", key, err)
	}
	if !a.IsNew {
		return StoryApproachView{}, fmt.Errorf("%w: kiểu %s đã có trong bảng Kiểu video", domain.ErrInvalidUnitEdit, a.Code)
	}
	if a.ArchetypeID != "" {
		return StoryApproachView{}, fmt.Errorf("%w: kiểu %s đã được lưu vào bảng Kiểu video", domain.ErrInvalidUnitEdit, a.Code)
	}
	saved, err := uc.archetypes.Create(ctx, domain.VideoArchetype{Code: a.Code, Name: a.Name, WhenToUse: a.WhenToUse, Playbook: a.Playbook})
	if err != nil {
		return StoryApproachView{}, err
	}
	a.ArchetypeID = saved.ID
	if unit.Content, err = json.Marshal(a); err != nil {
		return StoryApproachView{}, err
	}
	if _, err := uc.units.SaveUnit(ctx, unit); err != nil {
		return StoryApproachView{}, err
	}
	_, gate, err := uc.gate(ctx, projectID)
	if err != nil {
		return StoryApproachView{}, err
	}
	return StoryApproachView{Key: key, StoryApproach: a, Selected: gate.ApproachKey == key}, nil
}

// approachText describes the chosen approach for the frame prompt: the AI's
// own choice when none is chosen, else the approach with the playbook of its kind.
func (uc *StoryDeskUseCase) approachText(ctx context.Context, projectID, key string) (string, error) {
	if key == "" {
		return "AI tự chọn góc hợp nhất với đề tài và dữ kiện.", nil
	}
	unit, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindStoryApproach, key)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return "AI tự chọn góc hợp nhất với đề tài và dữ kiện.", nil
	}
	if err != nil {
		return "", err
	}
	var a domain.StoryApproach
	if err := json.Unmarshal(unit.Content, &a); err != nil {
		return "", fmt.Errorf("read stored approach %s: %w", key, err)
	}
	playbook := a.Playbook
	if !a.IsNew || playbook == "" {
		table, err := uc.archetypes.List(ctx)
		if err != nil {
			return "", fmt.Errorf("load video archetypes: %w", err)
		}
		for _, t := range table {
			if strings.EqualFold(t.Code, a.Code) {
				playbook = t.Playbook
			}
		}
	}
	text := a.Name + " (" + a.Code + ")\nGóc kể: " + a.Angle + "\nVì sao hợp: " + a.Why + "\nCâu mở mẫu: " + a.Open
	if playbook != "" {
		text += "\nCách kể của kiểu này:\n" + playbook
	}
	return text, nil
}
