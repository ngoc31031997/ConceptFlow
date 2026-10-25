package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"authoring/internal/domain"
)

// scriptLLM answers by the task named in the prompt, and keeps every prompt it was sent.
type scriptLLM struct {
	replies map[string]string
	systems []string
	err     error
}

func (l *scriptLLM) Name() string { return "fake" }
func (l *scriptLLM) Chat(_ context.Context, req ChatRequest) (ChatResult, error) {
	l.systems = append(l.systems, req.System)
	if l.err != nil {
		return ChatResult{}, l.err
	}
	for marker, reply := range l.replies {
		if strings.Contains(req.System, marker) {
			return ChatResult{Content: reply}, nil
		}
	}
	return ChatResult{}, errors.New("no reply for this prompt")
}

type memVoices struct {
	rows map[string]domain.VoiceLine
}

func (m *memVoices) ListVoiceLines(context.Context, string) ([]domain.VoiceLine, error) {
	out := []domain.VoiceLine{}
	for _, r := range m.rows {
		out = append(out, r)
	}
	return out, nil
}
func (m *memVoices) RequestVoiceLine(_ context.Context, l domain.VoiceLine) error {
	l.Status, l.RequestedAt = domain.VoiceLinePending, time.Now()
	m.rows[l.LineID] = l
	return nil
}
func (m *memVoices) ClaimVoiceLines(context.Context, string, time.Duration) ([]domain.VoiceLine, error) {
	return nil, nil
}
func (m *memVoices) SaveVoiceResults(context.Context, string, []domain.VoiceLineResult) error {
	return nil
}
func (m *memVoices) FailVoiceLines(_ context.Context, _ string, ids []string, reason string) error {
	for _, id := range ids {
		l := m.rows[id]
		l.Status, l.Error = domain.VoiceLineFailed, reason
		m.rows[id] = l
	}
	return nil
}

type countingDispatcher struct {
	calls int
	err   error
}

func (d *countingDispatcher) DispatchDraftVoice(context.Context, string) error {
	d.calls++
	return d.err
}

const (
	scriptWrite = "Viết lời dẫn cho beat trên"
	scriptRedo  = "Viết lại TOÀN BỘ lời dẫn"
	scriptLine  = "Gợi ý câu thay cho câu"
	scriptJoin  = "Gợi ý câu thay cho CÂU ĐẦU"
)

type scriptFixture struct {
	beat       *beatFixture
	llm        *scriptLLM
	voices     *memVoices
	dispatcher *countingDispatcher
	uc         *ScriptDeskUseCase
}

// newScriptFixture opens the step: story approved, three beats written and approved.
func newScriptFixture(t *testing.T) *scriptFixture {
	t.Helper()
	b := newBeatFixture()
	b.generate(t)
	for _, key := range []string{"b1", "b2", "b3"} {
		b.patchBeat(t, key, func(sb *domain.StoredBeat) { sb.Status = domain.UnitApproved })
	}
	project := &briefProject{project: domain.Project{ContentLanguage: domain.LanguageVietnamese, VideoOutputMode: domain.ModeLongOnly, VoiceID: "vi-HoaiMy"}, status: domain.StatusDraft}
	f := &scriptFixture{
		beat:       b,
		llm:        &scriptLLM{replies: map[string]string{}},
		voices:     &memVoices{rows: map[string]domain.VoiceLine{}},
		dispatcher: &countingDispatcher{},
	}
	f.uc = NewScriptDeskUseCase(b.store, project, project, briefPrompts{}, f.llm, nil, 4000, b.uc).WithVoices(f.voices, f.dispatcher)
	return f
}

func (f *scriptFixture) script(t *testing.T, key string) domain.ScriptBeat {
	t.Helper()
	u, err := f.beat.units.GetUnit(context.Background(), "p1", domain.UnitKindScriptBeat, key)
	if err != nil {
		t.Fatalf("script %s: %v", key, err)
	}
	s, err := readScriptBeat(u)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func scriptReply(lines ...string) string {
	items := make([]string, len(lines))
	for i, l := range lines {
		items[i] = fmt.Sprintf(`{"text":%q,"facts":["F01"]}`, l)
	}
	return `{"lines":[` + strings.Join(items, ",") + `]}`
}

func TestGenerateWritesEachBeatWithoutDialogueReadingItsNeighbours(t *testing.T) {
	f := newScriptFixture(t)
	f.llm.replies[scriptWrite] = scriptReply("Quả sung có xác ong không?", "Câu chuyện bắt đầu từ một bông hoa.")
	if err := f.uc.Generate(context.Background(), "p1", "", ""); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(f.llm.systems) != 3 {
		t.Fatalf("%d model calls, want one per beat", len(f.llm.systems))
	}
	s := f.script(t, "b2")
	if len(s.Lines) != 2 || s.Lines[0].ID != "b2.1" || s.Lines[1].ID != "b2.2" || s.NextSeq != 3 {
		t.Errorf("b2 = %+v, want lines b2.1, b2.2", s)
	}
	if !strings.Contains(f.llm.systems[1], "b1.1: Quả sung có xác ong không?") {
		t.Errorf("beat 2 must be written reading beat 1's dialogue, prompt: %q", f.llm.systems[1])
	}
	if !strings.Contains(f.llm.systems[0], "(chưa có lời thoại)") && !strings.Contains(f.llm.systems[0], "beat này chưa có lời thoại") {
		t.Errorf("beat 1 has a neighbour with no dialogue yet; the prompt must say so: %q", f.llm.systems[0])
	}
}

func TestGenerateRefusesBeforeEveryBeatIsApproved(t *testing.T) {
	f := newScriptFixture(t)
	f.beat.patchBeat(t, "b3", func(sb *domain.StoredBeat) { sb.Status = domain.UnitPending })
	if err := f.uc.Generate(context.Background(), "p1", "", ""); !errors.Is(err, ErrScriptNeedsBeats) {
		t.Errorf("err = %v, want the beat sheet to be approved first", err)
	}
	if len(f.llm.systems) != 0 {
		t.Error("the model must not be called while the step is not open")
	}
}

func TestGenerateStoresNothingFromAnUnusableReplyAndStopsTheRun(t *testing.T) {
	f := newScriptFixture(t)
	f.llm.replies[scriptWrite] = `{"lines":[{"text":"Câu.","facts":["F77"]}]}`
	err := f.uc.Generate(context.Background(), "p1", "", "")
	if !errors.Is(err, domain.ErrScriptReplyUnusable) {
		t.Fatalf("err = %v, want ErrScriptReplyUnusable", err)
	}
	units, _ := f.beat.units.ListUnits(context.Background(), "p1", domain.UnitKindScriptBeat)
	if len(units) != 0 || len(f.llm.systems) != 1 {
		t.Errorf("%d dialogues stored and %d model calls; the run must stop at the first bad reply", len(units), len(f.llm.systems))
	}
}

func TestGenerateOneBeatRefusesToOverwriteDialogue(t *testing.T) {
	f := newScriptFixture(t)
	f.llm.replies[scriptWrite] = scriptReply("Một câu.")
	if err := f.uc.Generate(context.Background(), "p1", "b1", ""); err != nil {
		t.Fatal(err)
	}
	if err := f.uc.Generate(context.Background(), "p1", "b1", ""); !errors.Is(err, ErrScriptBeatHasLines) {
		t.Errorf("err = %v, want ErrScriptBeatHasLines", err)
	}
	if _, err := f.uc.State(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
}

func TestEditWritesDialogueByHandAndKeepsLineIDsStable(t *testing.T) {
	f := newScriptFixture(t)
	kind, ctx := f.uc.ScriptUnitKind(), context.Background()
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{ScriptFieldAddStart: "Câu đầu tiên?"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{"add:b1.1": "Câu thứ hai."}); err != nil {
		t.Fatalf("add after: %v", err)
	}
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{"facts:b1.2": "f01, F02"}); err != nil {
		t.Fatalf("facts: %v", err)
	}
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{"line:b1.1": ""}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{ScriptFieldAddStart: "Câu mới đứng đầu."}); err != nil {
		t.Fatal(err)
	}
	s := f.script(t, "b1")
	if len(s.Lines) != 2 || s.Lines[0].ID != "b1.3" || s.Lines[1].ID != "b1.2" || strings.Join(s.Lines[1].Facts, ",") != "F01,F02" {
		t.Errorf("lines = %+v: the deleted b1.1 must not be reused", s.Lines)
	}
	if u, _ := f.beat.units.GetUnit(ctx, "p1", domain.UnitKindScriptBeat, "b1"); !u.HasCreatorField("lines") || u.Status != domain.UnitPending {
		t.Errorf("a hand edit must be the Creator's and send the beat to review: %+v", u)
	}
}

func TestEditRejectsAnUnknownFactAndAnEmptyLine(t *testing.T) {
	f := newScriptFixture(t)
	kind, ctx := f.uc.ScriptUnitKind(), context.Background()
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{ScriptFieldAddStart: "Câu."}); err != nil {
		t.Fatal(err)
	}
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{"facts:b1.1": "F77"}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("unknown fact: err = %v", err)
	}
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{ScriptFieldAddStart: "  "}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("empty line: err = %v", err)
	}
	if _, err := kind.Edit(ctx, "p1", "b9", map[string]string{ScriptFieldAddStart: "Câu."}); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("a beat that does not exist: err = %v", err)
	}
}

func TestApprovalNeedsDialogue(t *testing.T) {
	f := newScriptFixture(t)
	kind, ctx := f.uc.ScriptUnitKind(), context.Background()
	if err := kind.CheckApprovable(ctx, domain.ReviewUnit{Content: []byte(`{"lines":[],"next_seq":1}`)}); !errors.Is(err, ErrScriptBeatEmpty) {
		t.Errorf("err = %v, want ErrScriptBeatEmpty", err)
	}
	if err := kind.CheckApprovable(ctx, domain.ReviewUnit{Content: []byte(`{"lines":[{"id":"b1.1","text":"x","facts":[]}],"next_seq":2}`)}); err != nil {
		t.Errorf("err = %v, want a beat with a line to be approvable", err)
	}
}

func TestRedoKeepsLineNumbersGrowing(t *testing.T) {
	f := newScriptFixture(t)
	f.llm.replies[scriptWrite] = scriptReply("Câu một.", "Câu hai.")
	if err := f.uc.Generate(context.Background(), "p1", "b1", ""); err != nil {
		t.Fatal(err)
	}
	f.llm.replies[scriptRedo] = scriptReply("Câu mới.")
	if _, err := f.uc.ScriptUnitKind().Redo(context.Background(), "p1", "b1", "ngắn hơn"); err != nil {
		t.Fatal(err)
	}
	s := f.script(t, "b1")
	if len(s.Lines) != 1 || s.Lines[0].ID != "b1.3" {
		t.Errorf("lines = %+v, want one line b1.3: ids are never reused, so a voice file cannot attach to the wrong text", s.Lines)
	}
	if !strings.Contains(f.llm.systems[1], "ngắn hơn") {
		t.Error("the Creator's note must reach the model")
	}
}

func TestStateFlagsLinesJoinsAndChecksTheOpening(t *testing.T) {
	f := newScriptFixture(t)
	ctx, kind := context.Background(), f.uc.ScriptUnitKind()
	long := strings.TrimSpace(strings.Repeat("từ ", 30)) + "."
	for key, lines := range map[string][]string{
		"b1": {"Quả sung có xác ong không?", "Đó là câu hỏi lạ."},
		"b2": {long},
		"b3": {"Đó là câu kết với DNA."},
	} {
		for _, text := range lines {
			if _, err := kind.Edit(ctx, "p1", key, map[string]string{"add:" + lastID(f, key): text}); err != nil {
				t.Fatalf("%s: %v", key, err)
			}
		}
	}
	state, err := f.uc.State(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if !state.BeatsApproved || state.Stats.Lines != 4 || state.Stats.Written != 3 {
		t.Fatalf("stats = %+v", state.Stats)
	}
	if state.Beats[1].Lines[0].Flag == nil || state.Beats[1].Lines[0].Flag.Kind != domain.LineFlagLong {
		t.Errorf("a 30-word line must be flagged long: %+v", state.Beats[1].Lines[0].Flag)
	}
	if state.Beats[2].Lines[0].Flag == nil || state.Beats[2].Lines[0].Flag.Kind != domain.LineFlagHard {
		t.Errorf("an acronym must be flagged hard to hear: %+v", state.Beats[2].Lines[0].Flag)
	}
	if state.Stats.OpenFlags != 2 || state.Checks[2].OK {
		t.Errorf("open flags = %d, check = %+v", state.Stats.OpenFlags, state.Checks[2])
	}
	if !state.Checks[0].OK || !state.Checks[1].OK {
		t.Errorf("opening and closing checks = %+v", state.Checks[:2])
	}
	if state.Stats.Estimated != 4 || state.Beats[0].Lines[0].Audio.State != "none" {
		t.Errorf("no voice file yet: every line is an estimate; stats=%+v audio=%+v", state.Stats, state.Beats[0].Lines[0].Audio)
	}
}

// lastID is the id of the last line of a beat's dialogue, or "start" when it has none.
func lastID(f *scriptFixture, key string) string {
	u, err := f.beat.units.GetUnit(context.Background(), "p1", domain.UnitKindScriptBeat, key)
	if err != nil {
		return "start"
	}
	s, _ := readScriptBeat(u)
	if len(s.Lines) == 0 {
		return "start"
	}
	return s.Lines[len(s.Lines)-1].ID
}

func TestStateUsesTheRealLengthOfALineOnlyWhileItsWordingIsUnchanged(t *testing.T) {
	f := newScriptFixture(t)
	ctx, kind := context.Background(), f.uc.ScriptUnitKind()
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{ScriptFieldAddStart: "Quả sung có xác ong."}); err != nil {
		t.Fatal(err)
	}
	f.voices.rows["b1.1"] = domain.VoiceLine{
		LineID: "b1.1", TextSHA: domain.LineTextSHA("Quả sung có xác ong."), VoiceID: "vi-HoaiMy", Status: domain.VoiceLineReady,
		AudioPath: "/shared/p1/audio/b1-1_abc.wav", DurationSeconds: 9.5,
	}
	state, _ := f.uc.State(ctx, "p1")
	l := state.Beats[0].Lines[0]
	if l.Audio.State != "ready" || l.Audio.File != "b1-1_abc.wav" || l.Seconds != 9.5 || l.Estimated {
		t.Errorf("line = %+v, want the voice file's length", l)
	}
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{"line:b1.1": "Quả sung có xác ong thật."}); err != nil {
		t.Fatal(err)
	}
	state, _ = f.uc.State(ctx, "p1")
	l = state.Beats[0].Lines[0]
	if l.Audio.State != "none" || !l.Estimated || l.Seconds == 9.5 {
		t.Errorf("line = %+v: edited wording must not keep the old file", l)
	}
}

func TestSpeakAsksOnlyForLinesWithoutACurrentFile(t *testing.T) {
	f := newScriptFixture(t)
	ctx, kind := context.Background(), f.uc.ScriptUnitKind()
	for _, text := range []string{"Câu một.", "Câu hai."} {
		if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{"add:" + lastID(f, "b1"): text}); err != nil {
			t.Fatal(err)
		}
	}
	f.voices.rows["b1.1"] = domain.VoiceLine{LineID: "b1.1", TextSHA: domain.LineTextSHA("Câu một."), VoiceID: "vi-HoaiMy", Status: domain.VoiceLineReady, AudioPath: "/shared/p1/audio/x.wav"}
	if err := f.uc.Speak(ctx, "p1", []string{"b1.1", "b1.2"}); err != nil {
		t.Fatalf("Speak: %v", err)
	}
	if f.voices.rows["b1.1"].Status != domain.VoiceLineReady {
		t.Error("a line that has its file must not be read again")
	}
	if r := f.voices.rows["b1.2"]; r.Status != domain.VoiceLinePending || r.VoiceID != "vi-HoaiMy" || r.Text != "Câu hai." {
		t.Errorf("b1.2 = %+v, want pending with the project's voice", r)
	}
	if f.dispatcher.calls != 1 {
		t.Errorf("%d dispatches, want 1", f.dispatcher.calls)
	}
	if err := f.uc.Speak(ctx, "p1", []string{"b1.1"}); err != nil || f.dispatcher.calls != 1 {
		t.Errorf("err=%v calls=%d: nothing to read, nothing to send", err, f.dispatcher.calls)
	}
}

func TestSpeakMarksLinesFailedWhenTheCommandCannotBeSent(t *testing.T) {
	f := newScriptFixture(t)
	ctx := context.Background()
	if _, err := f.uc.ScriptUnitKind().Edit(ctx, "p1", "b1", map[string]string{ScriptFieldAddStart: "Câu."}); err != nil {
		t.Fatal(err)
	}
	f.dispatcher.err = errors.New("orchestrator down")
	if err := f.uc.Speak(ctx, "p1", []string{"b1.1"}); err == nil {
		t.Fatal("a command that was not sent must be an error, not a silent success")
	}
	if r := f.voices.rows["b1.1"]; r.Status != domain.VoiceLineFailed || !strings.Contains(r.Error, "orchestrator down") {
		t.Errorf("row = %+v, want failed with the reason", r)
	}
	if err := f.uc.Speak(ctx, "p1", []string{"b9.9"}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("unknown line: err = %v", err)
	}
}

func TestSuggestRepliesOnlyForAFlaggedLineAndStoresNothing(t *testing.T) {
	f := newScriptFixture(t)
	ctx, kind := context.Background(), f.uc.ScriptUnitKind()
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{ScriptFieldAddStart: "Chất CO2 rất nhiều."}); err != nil {
		t.Fatal(err)
	}
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{"add:b1.1": "Một câu ổn."}); err != nil {
		t.Fatal(err)
	}
	f.llm.replies[scriptLine] = `{"text":"Chất các-bon đi-ô-xít rất nhiều."}`
	text, err := f.uc.Suggest(ctx, "p1", "line", "b1.1")
	if err != nil || text != "Chất các-bon đi-ô-xít rất nhiều." {
		t.Fatalf("text=%q err=%v", text, err)
	}
	if got := f.script(t, "b1").Lines[0].Text; got != "Chất CO2 rất nhiều." {
		t.Errorf("a suggestion must not change the line, got %q", got)
	}
	if _, err := f.uc.Suggest(ctx, "p1", "line", "b1.2"); !errors.Is(err, ErrScriptLineNotFlagged) {
		t.Errorf("unflagged line: err = %v", err)
	}
	if _, err := f.uc.Suggest(ctx, "p1", "join", "b1.1"); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("the first beat has no join: err = %v", err)
	}
}

func TestSuggestForAJoinReplacesTheFirstLineOfTheLaterBeat(t *testing.T) {
	f := newScriptFixture(t)
	ctx, kind := context.Background(), f.uc.ScriptUnitKind()
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{ScriptFieldAddStart: "Ong vào quả sung."}); err != nil {
		t.Fatal(err)
	}
	if _, err := kind.Edit(ctx, "p1", "b2", map[string]string{ScriptFieldAddStart: "Ong mất cánh ngay."}); err != nil {
		t.Fatal(err)
	}
	state, _ := f.uc.State(ctx, "p1")
	if j := state.Beats[1].Join; j == nil || j.OK {
		t.Fatalf("two lines opening with the same word must be flagged: %+v", state.Beats[1].Join)
	}
	f.llm.replies[scriptJoin] = `{"text":"Rồi con ong mất cánh ngay."}`
	text, err := f.uc.Suggest(ctx, "p1", "join", "b2.1")
	if err != nil || text == "" {
		t.Fatalf("text=%q err=%v", text, err)
	}
}

func TestSpeakWithoutVoicesIsRefused(t *testing.T) {
	f := newScriptFixture(t)
	f.uc.WithVoices(nil, nil)
	if err := f.uc.Speak(context.Background(), "p1", []string{"b1.1"}); !errors.Is(err, ErrVoiceNotConfigured) {
		t.Errorf("err = %v, want ErrVoiceNotConfigured", err)
	}
}

func TestSpokenWordingIsWhatTheVoiceReadsAndKeepsTheShownText(t *testing.T) {
	f := newScriptFixture(t)
	ctx, kind := context.Background(), f.uc.ScriptUnitKind()
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{ScriptFieldAddStart: "D = (1 - m) / p"}); err != nil {
		t.Fatal(err)
	}
	if err := f.uc.Speak(ctx, "p1", []string{"b1.1"}); err != nil {
		t.Fatal(err)
	}
	if r := f.voices.rows["b1.1"]; r.Text != "D = (1 - m) / p" {
		t.Fatalf("with no spoken wording the voice reads the text, got %q", r.Text)
	}
	f.voices.rows["b1.1"] = domain.VoiceLine{LineID: "b1.1", TextSHA: domain.LineTextSHA("D = (1 - m) / p"), VoiceID: "vi-HoaiMy", Status: domain.VoiceLineReady, AudioPath: "/shared/p1/audio/x.wav", DurationSeconds: 4}

	const spoken = "Đê bằng một trừ mờ, rồi tất cả chia cho bê."
	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{"spoken:b1.1": spoken}); err != nil {
		t.Fatal(err)
	}
	state, _ := f.uc.State(ctx, "p1")
	l := state.Beats[0].Lines[0]
	if l.Text != "D = (1 - m) / p" || l.Spoken != spoken || l.Audio.State != "none" {
		t.Errorf("line = %+v: the shown text stays, and the file read from the old wording is no longer current", l)
	}
	if err := f.uc.Speak(ctx, "p1", []string{"b1.1"}); err != nil {
		t.Fatal(err)
	}
	if r := f.voices.rows["b1.1"]; r.Status != domain.VoiceLinePending || r.Text != spoken || r.TextSHA != domain.LineTextSHA(spoken) {
		t.Errorf("row = %+v, want the line asked again with the spoken wording", r)
	}

	if _, err := kind.Edit(ctx, "p1", "b1", map[string]string{"spoken:b1.1": ""}); err != nil {
		t.Fatal(err)
	}
	if got := f.script(t, "b1").Lines[0]; got.Spoken != "" || got.SpokenText() != "D = (1 - m) / p" {
		t.Errorf("line = %+v, want an empty spoken wording to read the text again", got)
	}
}
