package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"visual/internal/domain"
)

// userTurnNudge is the user turn of every call; the task is in the system prompt.
const userTurnNudge = "Hãy thực hiện nhiệm vụ trên và chỉ trả về kết quả theo đúng định dạng đã mô tả."

// modelFor is the model the video's step 8 runs on.
func (d *ConceptDesk) modelFor(ctx context.Context, projectID string) (string, error) {
	m, err := d.store.ConceptModel(ctx, projectID)
	if err != nil || m != "" {
		return m, err
	}
	return d.models.StepDefaultModel(ctx, ConceptStep)
}

// ask runs one turn of role on template vars and returns the JSON object of the reply.
func (d *ConceptDesk) ask(ctx context.Context, projectID, role string, vars map[string]string) (json.RawMessage, error) {
	model, err := d.modelFor(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return askJSON(ctx, d.llm, d.prompts, askCall{ProjectID: projectID, Role: role, Step: ConceptStep, Model: model, MaxTokens: d.maxTokens}, vars)
}

// askCall is what one JSON turn of a step needs besides its prompt variables.
type askCall struct {
	ProjectID, Role, Step, Model string
	MaxTokens                    int
}

// askJSON fills the template of call.Role with vars, runs one turn on the
// model and returns the JSON object of the reply.
func askJSON(ctx context.Context, llm LLMPort, prompts PromptLibraryPort, call askCall, vars map[string]string) (json.RawMessage, error) {
	template, err := prompts.ActiveTemplate(ctx, call.Role)
	if err != nil {
		return nil, err
	}
	res, err := llm.Chat(ctx, ChatRequest{
		System: renderTemplate(template, vars), User: userTurnNudge, Model: call.Model, MaxTokens: call.MaxTokens,
		ProjectID: call.ProjectID, Role: call.Role, Step: call.Step, ExpectJSON: true,
	})
	if err != nil {
		return nil, err
	}
	if len(res.JSON) == 0 {
		return nil, fmt.Errorf("%w: không có khối JSON", domain.ErrUnreadableReply)
	}
	return res.JSON, nil
}

// writeUnit has the AI write part key from the inputs and the parts already
// written, and stores it once the answer is in. The Creator's fields keep
// their values; the part goes back to review.
func (d *ConceptDesk) writeUnit(ctx context.Context, projectID string, data *deskData, key, note string) error {
	if key == domain.UnitTypography && data.fontsErr != nil {
		return fmt.Errorf("không viết được phần chữ vì chưa có danh sách font: %w", data.fontsErr)
	}
	prev, had := data.units[key]
	vars := inputVars(data.in, data.fonts)
	vars["parts"] = partsText(data.units, key)
	vars["current"], vars["creator_fields"] = "(chưa có)", none
	if had && len(prev.Content) > 0 {
		vars["current"] = string(prev.Content)
		if len(prev.CreatorFields) > 0 {
			vars["creator_fields"] = strings.Join(prev.CreatorFields, ", ")
		}
	}
	vars["note"] = orNone(note)
	reply, err := d.ask(ctx, projectID, domain.PromptRoleOf(key), vars)
	if err != nil {
		return err
	}
	if _, _, err := domain.DecodePart(key, reply); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrUnreadableReply, err)
	}
	content := []byte(reply)
	if had && len(prev.Content) > 0 && len(prev.CreatorFields) > 0 {
		if content, err = domain.OverlayFields(content, prev.Content, prev.CreatorFields); err != nil {
			return fmt.Errorf("%w: %v", domain.ErrUnreadableReply, err)
		}
	}
	v, canon, err := domain.DecodePart(key, content)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrUnreadableReply, err)
	}
	if f, ok := v.(*domain.Feasibility); ok {
		domain.MachineScope(f, len(data.in.Beats), data.parts.RepresentationRules)
		canon, _ = json.Marshal(f)
	}
	u := domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindConcept, Key: key, Status: domain.UnitPending,
		Content: canon, CreatorFields: []string{}, InputHashes: data.hashesFor(key)}
	if had {
		u.CreatorFields, u.RedoCount, u.CreatedAt = prev.CreatorFields, prev.RedoCount+1, prev.CreatedAt
	}
	if err := d.store.SaveUnit(ctx, u); err != nil {
		return err
	}
	data.units[key] = u
	data.parts.Set(key, v)
	return nil
}

// GenerateUnit has the AI write or rewrite one part, with the Creator's note.
// It runs one model turn, so an AI outside the system can answer it.
func (d *ConceptDesk) GenerateUnit(ctx context.Context, projectID, key, note string) (*ConceptState, error) {
	if _, ok := domain.UnitSpecOf(key); !ok {
		return nil, domain.ErrUnknownUnit
	}
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if d.running.has(projectID, key) {
		return nil, domain.ErrUnitRunning
	}
	finish := d.running.start(projectID, key)
	err = d.writeUnit(ctx, projectID, data, key, note)
	finish()
	if err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Generate has the AI write the parts one after another in part order, each
// reading the ones before it: the parts not written yet, or with rewriteAll
// every part (the Creator's fields keep their values). A failure stops the
// run; the parts written before it stay.
func (d *ConceptDesk) Generate(ctx context.Context, projectID, note string, rewriteAll bool) (*ConceptState, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, spec := range domain.ConceptUnits {
		if d.running.has(projectID, spec.Key) {
			return nil, domain.ErrUnitRunning
		}
		if u, ok := data.units[spec.Key]; rewriteAll || !ok || len(u.Content) == 0 {
			keys = append(keys, spec.Key)
		}
	}
	finish := d.running.start(projectID, keys...)
	defer finish()
	for _, k := range keys {
		if err := d.writeUnit(ctx, projectID, data, k, note); err != nil {
			return nil, fmt.Errorf("phần %d: %w", domain.UnitNumber(k), err)
		}
		d.running.done(projectID, k)
	}
	return d.State(ctx, projectID)
}
