package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"authoring/internal/domain"
)

// ErrScriptNeedsBeats is returned when the script is asked for before every beat is approved.
var ErrScriptNeedsBeats = errors.New("duyệt beat sheet ở bước Beat sheet trước")

// ErrScriptBeatHasLines is returned when a beat that already has dialogue is written from scratch.
var ErrScriptBeatHasLines = errors.New("beat đã có lời thoại: dùng Làm lại để viết lại")

// ErrScriptLineNotFlagged is returned when a suggestion is asked for a line or join the checks pass.
var ErrScriptLineNotFlagged = errors.New("câu này không có cờ nào để gợi ý")

// ScriptAudio is the reading of one line as the screen shows it.
type ScriptAudio struct {
	// State is none (never asked), pending, running, ready or failed. A line
	// whose wording changed since the file was read is none again.
	State string `json:"state"`
	// File is the name api-gateway serves the voice file under; set when ready.
	File  string `json:"file,omitempty"`
	Error string `json:"error,omitempty"`
}

// ScriptLineView is one line of dialogue as the screen shows it.
type ScriptLineView struct {
	domain.ScriptLine
	// Seconds is how long the line takes to say: the voice file's length when
	// there is one for the current wording, else an estimate.
	Seconds   float64          `json:"seconds"`
	Estimated bool             `json:"estimated"`
	Flag      *domain.LineFlag `json:"flag"`
	Audio     ScriptAudio      `json:"audio"`
}

// ScriptJoinView is the join between a beat and the one before it.
type ScriptJoinView struct {
	OK  bool   `json:"ok"`
	Msg string `json:"msg"`
}

// ScriptBeatView is one beat's dialogue as the screen shows it.
type ScriptBeatView struct {
	Key    string `json:"key"`
	Number int    `json:"number"`
	Fn     string `json:"fn"`
	Title  string `json:"title"`
	Goal   string `json:"goal"`
	// TargetSeconds is the length the beat sheet gave the beat.
	TargetSeconds int `json:"target_seconds"`
	// Written is true once the beat has dialogue.
	Written bool              `json:"written"`
	Status  domain.UnitStatus `json:"status"`
	Mine    []string          `json:"mine"`
	// Stale is true when the beat changed since its dialogue was written.
	Stale   bool             `json:"stale"`
	Lines   []ScriptLineView `json:"lines"`
	Seconds float64          `json:"seconds"`
	// Join is nil for the first beat and for a beat that has no dialogue to join.
	Join *ScriptJoinView `json:"join"`
}

// ScriptStats are the counts shown over the whole script.
type ScriptStats struct {
	Lines   int `json:"lines"`
	Beats   int `json:"beats"`
	Written int `json:"written"`
	// Estimated is how many lines have no voice file for their current wording.
	Estimated    int     `json:"estimated"`
	TotalSeconds float64 `json:"total_seconds"`
	OpenFlags    int     `json:"open_flags"`
	OpenJoins    int     `json:"open_joins"`
}

// ScriptState is everything the script screen shows.
type ScriptState struct {
	// Status is approved when every beat has dialogue and every beat is approved.
	Status        domain.UnitStatus    `json:"status"`
	BeatsApproved bool                 `json:"beats_approved"`
	Beats         []ScriptBeatView     `json:"beats"`
	Checks        []domain.ScriptCheck `json:"checks"`
	Stats         ScriptStats          `json:"stats"`
	TargetLabel   string               `json:"target_label"`
	// InRange is true when the script's length is inside the video's target.
	InRange bool `json:"in_range"`
	// VoiceID is the voice the lines are read with; "" is the default voice.
	VoiceID string `json:"voice_id"`
}

// ScriptDeskUseCase backs the script step: the dialogue of each beat, written
// by the AI from the approved beat sheet, its checks, and the reading of each
// line. The dialogue of a beat is a review unit (domain.UnitKindScriptBeat)
// keyed by the beat's key and stored as a row of project_script.
type ScriptDeskUseCase struct {
	units           StepUnitStorePort
	projects        ProjectReaderPort
	locks           AuthoringLockPort
	prompts         PromptActivePort
	provider        LLMProviderPort
	maxOutputTokens int
	beats           *BeatDeskUseCase
	voices          VoiceLinePort
	dispatcher      VoiceDispatcherPort
	now             func() time.Time
}

// NewScriptDeskUseCase wires the use case. provider may be nil when no
// language model is configured: the Creator then writes the lines by hand.
// voices and dispatcher may be nil, which turns listening off.
func NewScriptDeskUseCase(
	units StepUnitStorePort, projects ProjectReaderPort, locks AuthoringLockPort, prompts PromptActivePort,
	provider LLMProviderPort, maxOutputTokens int, beats *BeatDeskUseCase,
) *ScriptDeskUseCase {
	return &ScriptDeskUseCase{
		units: units, projects: projects, locks: locks, prompts: prompts, provider: provider,
		maxOutputTokens: maxOutputTokens, beats: beats, now: time.Now,
	}
}

// WithVoices turns on listening: voices stores the reading of each line and
// dispatcher asks the orchestrator to have waiting lines read.
func (uc *ScriptDeskUseCase) WithVoices(voices VoiceLinePort, dispatcher VoiceDispatcherPort) *ScriptDeskUseCase {
	uc.voices, uc.dispatcher = voices, dispatcher
	return uc
}

// storedScript is a script_beat unit with its content read.
type storedScript struct {
	unit domain.ReviewUnit
	domain.ScriptBeat
}

func readScriptBeat(u domain.ReviewUnit) (domain.ScriptBeat, error) {
	var s domain.ScriptBeat
	if err := json.Unmarshal(u.Content, &s); err != nil {
		return domain.ScriptBeat{}, fmt.Errorf("read stored script %s: %w", u.Key, err)
	}
	if s.Lines == nil {
		s.Lines = []domain.ScriptLine{}
	}
	for i := range s.Lines {
		if s.Lines[i].Facts == nil {
			s.Lines[i].Facts = []string{}
		}
	}
	if s.NextSeq < 1 {
		s.NextSeq = len(s.Lines) + 1
	}
	return s, nil
}

// scripts loads every stored dialogue by beat key.
func (uc *ScriptDeskUseCase) scripts(ctx context.Context, projectID string) (map[string]storedScript, error) {
	units, err := uc.units.ListUnits(ctx, projectID, domain.UnitKindScriptBeat)
	if err != nil {
		return nil, err
	}
	out := make(map[string]storedScript, len(units))
	for _, u := range units {
		s, err := readScriptBeat(u)
		if err != nil {
			return nil, err
		}
		out[u.Key] = storedScript{unit: u, ScriptBeat: s}
	}
	return out, nil
}

func (uc *ScriptDeskUseCase) saveScript(ctx context.Context, projectID, key string, s domain.ScriptBeat, base domain.ReviewUnit) (domain.ReviewUnit, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	base.ProjectID, base.Kind, base.Key, base.Content = projectID, domain.UnitKindScriptBeat, key, raw
	return uc.units.SaveUnit(ctx, base)
}

// beatsApproved reports whether the beat sheet has beats and all are approved.
func beatsApproved(beats []storedBeat) bool {
	for _, b := range beats {
		if b.unit.Status != domain.UnitApproved {
			return false
		}
	}
	return len(beats) > 0
}

// voiceOf returns the voice and language the project reads its lines with.
func (uc *ScriptDeskUseCase) voiceOf(ctx context.Context, projectID string) (voiceID string, lang domain.ContentLanguage, err error) {
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return "", "", fmt.Errorf("load project: %w", err)
	}
	return project.VoiceID, project.ContentLanguage, nil
}

func audioOf(l domain.VoiceLine, ok bool, text, voiceID string) ScriptAudio {
	if !ok || !audioIsCurrent(l, text, voiceID) {
		return ScriptAudio{State: "none"}
	}
	switch l.Status {
	case domain.VoiceLineReady:
		return ScriptAudio{State: "ready", File: audioFileName(l)}
	case domain.VoiceLineFailed:
		return ScriptAudio{State: "failed", Error: l.Error}
	default:
		return ScriptAudio{State: string(l.Status)}
	}
}

// State returns the whole script screen.
func (uc *ScriptDeskUseCase) State(ctx context.Context, projectID string) (ScriptState, error) {
	state := ScriptState{Status: domain.UnitPending, Beats: []ScriptBeatView{}, Checks: []domain.ScriptCheck{}}
	beats, _, err := uc.beats.beats(ctx, projectID)
	if err != nil {
		return ScriptState{}, err
	}
	state.BeatsApproved = beatsApproved(beats)
	scripts, err := uc.scripts(ctx, projectID)
	if err != nil {
		return ScriptState{}, err
	}
	voiceID, lang, err := uc.voiceOf(ctx, projectID)
	if err != nil {
		return ScriptState{}, err
	}
	state.VoiceID = voiceID
	voiceRows := map[string]domain.VoiceLine{}
	if uc.voices != nil {
		rows, err := uc.voices.ListVoiceLines(ctx, projectID)
		if err != nil {
			return ScriptState{}, err
		}
		for _, r := range rows {
			voiceRows[r.LineID] = r
		}
	}

	approved, written := 0, 0
	var firstBeatLines []domain.ScriptLine
	var firstSecs []float64
	var lastLine *domain.ScriptLine
	var prevLast string
	for i, b := range beats {
		sc, has := scripts[b.unit.Key]
		view := ScriptBeatView{
			Key: b.unit.Key, Number: i + 1, Fn: b.Fn, Title: b.Title, Goal: b.Goal, TargetSeconds: b.Dur,
			Written: has && len(sc.Lines) > 0, Status: domain.UnitPending, Mine: []string{}, Lines: []ScriptLineView{},
		}
		if has {
			view.Status, view.Stale = sc.unit.Status, sc.unit.Stale
			view.Mine = append(view.Mine, sc.unit.CreatorFields...)
		}
		if view.Written {
			written++
			if view.Status == domain.UnitApproved {
				approved++
			}
		}
		secs := make([]float64, 0, len(sc.Lines))
		for _, l := range sc.Lines {
			lv := ScriptLineView{ScriptLine: l, Flag: domain.FlagLine(l.SpokenText())}
			row, ok := voiceRows[l.ID]
			lv.Audio = audioOf(row, ok, l.SpokenText(), voiceID)
			if lv.Audio.State == "ready" {
				lv.Seconds = row.DurationSeconds
			} else {
				lv.Seconds, lv.Estimated = domain.EstimateNarrationDuration(l.SpokenText(), lang), true
				state.Stats.Estimated++
			}
			if lv.Flag != nil {
				state.Stats.OpenFlags++
			}
			view.Seconds += lv.Seconds
			secs = append(secs, lv.Seconds)
			view.Lines = append(view.Lines, lv)
		}
		if view.Written {
			if i > 0 && prevLast != "" {
				if j := domain.FlagJoin(prevLast, sc.Lines[0].SpokenText()); j != nil {
					view.Join = &ScriptJoinView{Msg: j.Msg}
					state.Stats.OpenJoins++
				} else {
					view.Join = &ScriptJoinView{OK: true, Msg: "nối mượt với beat trước"}
				}
			}
			last := sc.Lines[len(sc.Lines)-1]
			prevLast = last.SpokenText()
			lastLine = &sc.Lines[len(sc.Lines)-1]
			if firstBeatLines == nil {
				firstBeatLines, firstSecs = sc.Lines, secs
			}
		} else {
			prevLast = ""
		}
		state.Stats.Lines += len(view.Lines)
		state.Stats.TotalSeconds += view.Seconds
		state.Beats = append(state.Beats, view)
	}
	state.Stats.Beats, state.Stats.Written = len(beats), written
	if len(beats) > 0 && approved == len(beats) {
		state.Status = domain.UnitApproved
	}

	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return ScriptState{}, fmt.Errorf("load project: %w", err)
	}
	duration, err := uc.units.GetTargetDuration(ctx, projectID)
	if err != nil {
		return ScriptState{}, err
	}
	lo, hi := domain.TargetSeconds(project.VideoOutputMode, duration)
	state.TargetLabel = domain.TargetDurationLabel(project.VideoOutputMode, duration, "vi")
	total := int(state.Stats.TotalSeconds + 0.5)
	state.InRange = written > 0 && total >= lo && total <= hi

	// The opening is the first beat only when that beat is the first of the sheet.
	var opening []domain.ScriptLine
	var openingSecs []float64
	if len(beats) > 0 && state.Beats[0].Written {
		opening, openingSecs = firstBeatLines, firstSecs
	}
	state.Checks = append(state.Checks, domain.CheckOpening(opening, openingSecs))
	var closing *domain.ScriptLine
	if n := len(state.Beats); n > 0 && state.Beats[n-1].Written {
		closing = lastLine
	}
	state.Checks = append(state.Checks, domain.CheckClosing(closing))
	state.Checks = append(state.Checks,
		domain.ScriptCheck{OK: state.Stats.OpenFlags == 0, Text: flagCheckText(state.Stats.OpenFlags)},
		domain.ScriptCheck{OK: state.Stats.OpenJoins == 0, Text: joinCheckText(state.Stats.OpenJoins)},
	)
	return state, nil
}

func flagCheckText(n int) string {
	if n == 0 {
		return "Không còn câu dài hay khó nghe khi đọc"
	}
	return fmt.Sprintf("%d câu dài hoặc khó nghe khi đọc", n)
}

func joinCheckText(n int) string {
	if n == 0 {
		return "Các beat nối nhau mượt"
	}
	return fmt.Sprintf("%d chỗ nối giữa hai beat cần xem", n)
}

// FactDependents lists, for the research step, the dialogue that uses a fact.
func (uc *ScriptDeskUseCase) FactDependents() FactDependentsFinder {
	return func(ctx context.Context, projectID, factID string) ([]string, error) {
		beats, _, err := uc.beats.beats(ctx, projectID)
		if err != nil {
			return nil, err
		}
		scripts, err := uc.scripts(ctx, projectID)
		if err != nil {
			return nil, err
		}
		var items []string
		for i, b := range beats {
			for _, l := range scripts[b.unit.Key].Lines {
				for _, id := range l.Facts {
					if id == factID {
						items = append(items, fmt.Sprintf("Viết kịch bản · B%d câu %s dùng %s: beat được duyệt lại", i+1, l.ID, factID))
					}
				}
			}
		}
		return items, nil
	}
}

// beatIndex finds the beat with key in beats.
func beatIndex(beats []storedBeat, key string) int { return indexOfBeat(beats, key) }

// sortedKeys returns the keys of fields in a fixed order, so an edit with
// several fields applies the same way every time.
func sortedKeys(fields map[string]string) []string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// lineIndex finds the line with id in lines.
func lineIndex(lines []domain.ScriptLine, id string) int {
	for i, l := range lines {
		if l.ID == id {
			return i
		}
	}
	return -1
}

func linesText(lines []domain.ScriptLine) string {
	if len(lines) == 0 {
		return "(chưa có)"
	}
	rows := make([]string, 0, len(lines))
	for _, l := range lines {
		rows = append(rows, fmt.Sprintf("%s: %s", l.ID, l.Text))
	}
	return strings.Join(rows, "\n")
}
