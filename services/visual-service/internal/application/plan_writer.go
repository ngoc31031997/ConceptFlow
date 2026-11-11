package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"visual/internal/domain"
)

// aiReady fails with ErrLLMNotConfigured when the desk has no AI wired in.
func (d *PlanDesk) aiReady() error {
	if d.llm == nil || d.prompts == nil || d.models == nil {
		return ErrLLMNotConfigured
	}
	return nil
}

// modelFor is the model the video's step 9 runs on.
func (d *PlanDesk) modelFor(ctx context.Context, projectID string) (string, error) {
	m, err := d.store.PlanModel(ctx, projectID)
	if err != nil || m != "" {
		return m, err
	}
	return d.models.StepDefaultModel(ctx, PlanStep)
}

// SetModel saves the model the video's step 9 runs on; "" is the step default.
func (d *PlanDesk) SetModel(ctx context.Context, projectID, model string) (*PlanState, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrNoProjectID
	}
	if err := d.aiReady(); err != nil {
		return nil, err
	}
	model = strings.TrimSpace(model)
	if model != "" {
		models, err := d.models.Models(ctx)
		if err != nil {
			return nil, err
		}
		found := false
		for _, m := range models {
			found = found || m.ID == model
		}
		if !found {
			return nil, ErrUnknownModel
		}
	}
	if err := d.store.SetPlanModel(ctx, projectID, model); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// ask runs one turn of role on the template's variables and returns the JSON
// object of the reply.
func (d *PlanDesk) ask(ctx context.Context, projectID, role string, vars map[string]string) (json.RawMessage, error) {
	model, err := d.modelFor(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return askJSON(ctx, d.llm, d.prompts, askCall{ProjectID: projectID, Role: role, Step: PlanStep, Model: model, MaxTokens: d.maxTokens}, vars)
}

// writeBeat has the AI write the units of beat key from the beat's lines, its
// neighbours and the locked visual concept, and stores them once the answer is
// in and fits. The units the Creator edited keep their id and their fields; the
// rest is the AI's, and every unit goes back to review.
func (d *PlanDesk) writeBeat(ctx context.Context, projectID string, data *planData, key, note string) error {
	if data.in.SecondsPerWord <= 0 {
		return domain.ErrNoNarrationPace
	}
	current, had := data.plans[key], len(data.plans[key].Units) > 0
	creatorFields := data.rows[key].CreatorFields
	vars := beatVars(data.in, data.plans, key, current, creatorFields, note)
	reply, err := d.ask(ctx, projectID, domain.PromptRolePlanBeat, vars)
	if err != nil {
		return err
	}
	fresh, err := domain.ParsePlanBeatReply(reply, data.in.Script.Lines[key])
	if err != nil {
		return err
	}
	merge, err := domain.MergeBeatReply(fresh, current, creatorFields)
	if err != nil {
		return err
	}
	if len(merge.Fresh) > 0 {
		first, err := d.store.ReserveUnitNumbers(ctx, projectID, len(merge.Fresh))
		if err != nil {
			return err
		}
		merge.Assign(first)
	}
	row := data.beatRow(projectID, key)
	row.InputHashes = data.in.HashesFor(key)
	if had {
		row.RedoCount++
	}
	data.rows[key] = row
	return d.saveBeat(ctx, data, projectID, key, domain.BeatPlan{Units: merge.Units}, merge.Fields)
}

// GenerateBeat has the AI write or rewrite the units of one beat, with the
// Creator's note. It runs one model turn, so an AI outside the system can
// answer it.
func (d *PlanDesk) GenerateBeat(ctx context.Context, projectID, beatKey, note string) (*PlanState, error) {
	if err := d.aiReady(); err != nil {
		return nil, err
	}
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if _, ok := data.in.Beat(beatKey); !ok {
		return nil, domain.ErrUnknownPlanBeat
	}
	if d.running.has(projectID, beatKey) {
		return nil, domain.ErrUnitRunning
	}
	finish := d.running.start(projectID, beatKey)
	err = d.writeBeat(ctx, projectID, data, beatKey, note)
	finish()
	if err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// Generate has the AI write the beats that have no unit yet, one after another
// in story order, each reading the beats written before it. A failure stops the
// run; the beats written before it stay.
func (d *PlanDesk) Generate(ctx context.Context, projectID, note string) (*PlanState, error) {
	if err := d.aiReady(); err != nil {
		return nil, err
	}
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, b := range data.in.Script.Beats {
		if d.running.has(projectID, b.Key) {
			return nil, domain.ErrUnitRunning
		}
		if len(data.plans[b.Key].Units) == 0 {
			keys = append(keys, b.Key)
		}
	}
	finish := d.running.start(projectID, keys...)
	defer finish()
	for _, k := range keys {
		if err := d.writeBeat(ctx, projectID, data, k, note); err != nil {
			b, _ := data.in.Beat(k)
			return nil, fmt.Errorf("beat %d: %w", b.Number, err)
		}
		d.running.done(projectID, k)
	}
	return d.State(ctx, projectID)
}
