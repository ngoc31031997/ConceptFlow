package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"visual/internal/domain"
)

type seqModels struct{ fakeModels }

func (seqModels) StepDefaultModel(_ context.Context, step string) (string, error) {
	if step != SequenceStep {
		return "", ErrLLMFailed
	}
	return "ollama", nil
}

type seqAIFixture struct {
	*seqFixture
	llm *fakeLLM
}

func newSeqAIFixture(t *testing.T) *seqAIFixture {
	t.Helper()
	f := &seqAIFixture{seqFixture: newSeqFixture(t), llm: &fakeLLM{replies: map[string][]string{}}}
	f.desk.WithAI(f.llm, seedPrompts{}, seqModels{}, 1000)
	return f
}

func (f *seqAIFixture) queue(role string, replies ...string) { f.llm.replies[role] = replies }

func motifJSON(sameAs, name, anchor string, units ...string) string {
	q := make([]string, len(units))
	for i, u := range units {
		q[i] = `"` + u + `"`
	}
	return fmt.Sprintf(`{"same_as":%q,"name":%q,"description":"lặp lại","anchor_ref":%q,"role":"neo","planned_units":[%s]}`,
		sameAs, name, anchor, strings.Join(q, ","))
}

func motifsReply(m ...string) string { return `{"motifs":[` + strings.Join(m, ",") + `]}` }

func seqItemJSON(unit string, first bool, tempo, via string, motifs ...string) string {
	q := make([]string, len(motifs))
	for i, m := range motifs {
		q[i] = `"` + m + `"`
	}
	tr := "null"
	if !first {
		tr = fmt.Sprintf(`{"type":"Cắt","description":"d","via_motif":%q}`, via)
	}
	return fmt.Sprintf(`{"unit_id":%q,"purpose_in_flow":"mở","pacing":{"name":"Đều","tempo":%q,"description":"d"},"transition_from_previous":%s,"motif_refs":[%s]}`,
		unit, tempo, tr, strings.Join(q, ","))
}

func itemsReply(items ...string) string { return `{"items":[` + strings.Join(items, ",") + `]}` }

// approvedMotifs has the AI set two motifs and the Creator approve them.
func (f *seqAIFixture) approvedMotifs() {
	f.t.Helper()
	f.queue(domain.PromptRoleSequenceMotifs, motifsReply(motifJSON("", "Ong", anchorBee, "VP-001", "VP-002"), motifJSON("", "Đồng hồ", anchorClock, "VP-002", "VP-003")))
	f.must(f.desk.GenerateMotifs(f.ctx, project, ""))
	f.must(f.desk.ApproveMotifs(f.ctx, project))
}

func TestSequenceAI_MotifsTakeNewNumbersAndKeepWhatTheCreatorWrote(t *testing.T) {
	f := newSeqAIFixture(t)
	f.queue(domain.PromptRoleSequenceMotifs, motifsReply(motifJSON("", "Ong", anchorBee, "VP-001", "VP-002")))
	st := f.must(f.desk.GenerateMotifs(f.ctx, project, "ít motif"))
	call := f.llm.calls[0]
	if call.Role != domain.PromptRoleSequenceMotifs || call.Step != SequenceStep || strings.Contains(call.System, "{{") ||
		!strings.Contains(call.System, anchorBee) || !strings.Contains(call.System, "VP-003") || !strings.Contains(call.System, "ít motif") {
		t.Fatalf("one turn with every variable filled:\n%s", call.System)
	}
	if len(st.Motifs.Motifs) != 1 || st.Motifs.Motifs[0].MotifID != "MOTIF-01" || st.Motifs.Status != domain.UnitPending || st.Motifs.Stale {
		t.Fatalf("the AI's motif takes MOTIF-01 and waits for review: %+v", st.Motifs)
	}
	f.must(f.desk.EditMotif(f.ctx, project, "MOTIF-01", fields(t, map[string]any{"name": "Ong của Creator"})))
	f.queue(domain.PromptRoleSequenceMotifs, motifsReply(motifJSON("MOTIF-01", "Ong AI", anchorBee, "VP-001"), motifJSON("", "Đồng hồ", anchorClock, "VP-003")))
	st = f.must(f.desk.GenerateMotifs(f.ctx, project, ""))
	if !strings.Contains(f.llm.calls[1].System, "MOTIF-01.name") {
		t.Errorf("the prompt names the Creator's fields")
	}
	m := st.Motifs.Motifs
	if len(m) != 2 || m[0].MotifID != "MOTIF-01" || m[0].Name != "Ong của Creator" || m[1].MotifID != "MOTIF-02" {
		t.Fatalf("same_as keeps the id and the Creator's name, a new motif takes the next number: %+v", m)
	}
	if strings.Join(m[0].CreatorFields, ",") != "name" {
		t.Errorf("the Creator's field still stands: %v", m[0].CreatorFields)
	}
}

func TestSequenceAI_AMotifReplyThatDoesNotFitStoresNothing(t *testing.T) {
	f := newSeqAIFixture(t)
	f.approvedMotifs()
	f.writeItem("VP-001", true, "MOTIF-01")
	before := fmt.Sprint(f.store.units)
	for name, reply := range map[string]string{
		"drops a motif in use": motifsReply(motifJSON("MOTIF-02", "Đồng hồ", anchorClock)),
		"unknown same_as":      motifsReply(motifJSON("MOTIF-01", "Ong", anchorBee), motifJSON("MOTIF-09", "x", anchorBee)),
		"same_as twice":        motifsReply(motifJSON("MOTIF-01", "Ong", anchorBee), motifJSON("MOTIF-01", "x", anchorBee)),
		"unknown unit":         motifsReply(motifJSON("MOTIF-01", "Ong", anchorBee, "VP-099")),
		"machine key":          `{"motifs":[{"motif_id":"MOTIF-07","same_as":"","name":"x","description":"d","anchor_ref":"a","role":"r","planned_units":[]}]}`,
		"empty":                `{"motifs":[]}`,
	} {
		f.queue(domain.PromptRoleSequenceMotifs, reply)
		if _, err := f.desk.GenerateMotifs(f.ctx, project, ""); !errors.Is(err, domain.ErrUnreadableReply) {
			t.Errorf("%s: want ErrUnreadableReply, got %v", name, err)
		}
	}
	if fmt.Sprint(f.store.units) != before {
		t.Error("a refused reply stores nothing")
	}
}

func TestSequenceAI_BeatNeedsApprovedMotifsAndWritesEveryUnitInOrder(t *testing.T) {
	f := newSeqAIFixture(t)
	if _, err := f.desk.GenerateBeat(f.ctx, project, "b1", ""); !errors.Is(err, domain.ErrMotifsNotApproved) {
		t.Fatalf("no approved motif: want ErrMotifsNotApproved, got %v", err)
	}
	f.approvedMotifs()
	f.queue(domain.PromptRoleSequenceBeat, itemsReply(seqItemJSON("VP-001", true, "slow", "", "MOTIF-01"), seqItemJSON("VP-002", false, "fast", "MOTIF-01", "MOTIF-01", "MOTIF-02")))
	st := f.must(f.desk.GenerateBeat(f.ctx, project, "b1", "nhanh dần"))
	call := f.llm.calls[len(f.llm.calls)-1]
	if call.Role != domain.PromptRoleSequenceBeat || strings.Contains(call.System, "{{") || !strings.Contains(call.System, "unit đầu video") ||
		!strings.Contains(call.System, "MOTIF-02") || !strings.Contains(call.System, "nhanh dần") {
		t.Fatalf("the beat prompt is filled:\n%s", call.System)
	}
	it := item(st, "VP-002")
	if it.Item == nil || it.Item.Pacing.Tempo != "fast" || it.Item.TransitionFromPrevious.ViaMotif != "MOTIF-01" || it.Item.Status != domain.UnitPending || len(it.Problems) != 0 {
		t.Fatalf("VP-002 is written and waits for review: %+v", it)
	}

	f.must(f.desk.Edit(f.ctx, project, "VP-001", fields(t, map[string]any{"purpose_in_flow": "của Creator"})))
	f.queue(domain.PromptRoleSequenceBeat, itemsReply(seqItemJSON("VP-003", false, "calm", "MOTIF-02", "MOTIF-02")))
	f.must(f.desk.GenerateBeat(f.ctx, project, "b2", ""))
	if s := f.llm.calls[len(f.llm.calls)-1].System; !strings.Contains(s, `"tempo":"fast"`) {
		t.Errorf("the next beat reads the last unit of the beat before and its item:\n%s", s)
	}

	f.queue(domain.PromptRoleSequenceBeat, itemsReply(seqItemJSON("VP-001", true, "medium", ""), seqItemJSON("VP-002", false, "medium", "")))
	st = f.must(f.desk.GenerateBeat(f.ctx, project, "b1", ""))
	if it := item(st, "VP-001"); it.Item.PurposeInFlow != "của Creator" || it.Item.Pacing.Tempo != "medium" {
		t.Errorf("a rewrite keeps the Creator's field and takes the AI's others: %+v", it.Item)
	}
}

func TestSequenceAI_ABeatReplyThatDoesNotFitStoresNothing(t *testing.T) {
	f := newSeqAIFixture(t)
	f.approvedMotifs()
	before := fmt.Sprint(f.store.units)
	for name, reply := range map[string]string{
		"missing unit":             itemsReply(seqItemJSON("VP-001", true, "slow", "")),
		"wrong order":              itemsReply(seqItemJSON("VP-002", false, "slow", ""), seqItemJSON("VP-001", true, "slow", "")),
		"tempo outside its list":   itemsReply(seqItemJSON("VP-001", true, "rapid", ""), seqItemJSON("VP-002", false, "slow", "")),
		"unknown motif":            itemsReply(seqItemJSON("VP-001", true, "slow", "", "MOTIF-09"), seqItemJSON("VP-002", false, "slow", "")),
		"opening has a transition": itemsReply(seqItemJSON("VP-001", false, "slow", ""), seqItemJSON("VP-002", false, "slow", "")),
		"no transition":            itemsReply(seqItemJSON("VP-001", true, "slow", ""), seqItemJSON("VP-002", true, "slow", "")),
		"via a motif not carried":  itemsReply(seqItemJSON("VP-001", true, "slow", ""), seqItemJSON("VP-002", false, "slow", "MOTIF-01")),
		"machine key":              `{"items":[{"unit_id":"VP-001","status":"approved"}]}`,
	} {
		f.queue(domain.PromptRoleSequenceBeat, reply)
		if _, err := f.desk.GenerateBeat(f.ctx, project, "b1", ""); !errors.Is(err, domain.ErrUnreadableReply) {
			t.Errorf("%s: want ErrUnreadableReply, got %v", name, err)
		}
	}
	if fmt.Sprint(f.store.units) != before {
		t.Error("a refused reply stores nothing")
	}
}

func TestSequenceAI_GenerateWritesEmptyBeatsInOrder(t *testing.T) {
	f := newSeqAIFixture(t)
	f.approvedMotifs()
	f.queue(domain.PromptRoleSequenceBeat,
		itemsReply(seqItemJSON("VP-001", true, "slow", "", "MOTIF-01"), seqItemJSON("VP-002", false, "fast", "", "MOTIF-01")),
		itemsReply(seqItemJSON("VP-003", false, "calm", "", "MOTIF-02")))
	st := f.must(f.desk.Generate(f.ctx, project, ""))
	if !st.Coverage.Complete() || len(f.llm.calls) != 3 {
		t.Fatalf("both beats are written in two turns: %+v, %d calls", st.Coverage, len(f.llm.calls))
	}
	if !strings.Contains(f.llm.calls[2].System, `"tempo":"fast"`) {
		t.Errorf("the second beat reads the item the first one just wrote")
	}
	f.must(f.desk.Generate(f.ctx, project, ""))
	if len(f.llm.calls) != 3 {
		t.Error("written beats are not written again")
	}
}

func TestSequenceAI_ReviewFindingsNameAUnitOrAMotif(t *testing.T) {
	f := newSeqAIFixture(t)
	if _, err := f.desk.Review(f.ctx, project, ""); !errors.Is(err, domain.ErrUnitNotWritten) {
		t.Fatalf("nothing to review: %v", err)
	}
	f.approvedMotifs()
	f.writeItem("VP-001", true, "MOTIF-01")
	f.queue(domain.PromptRoleSequenceReview, reviewReply(
		finding("clarity", "VP-001", "purpose_in_flow", "mơ hồ", ""),
		finding("consistency", "MOTIF-02", "description", "lệch cách vẽ", "")))
	st := f.must(f.desk.Review(f.ctx, project, "kỹ"))
	call := f.llm.calls[len(f.llm.calls)-1]
	if call.Role != domain.PromptRoleSequenceReview || strings.Contains(call.System, "{{") || !strings.Contains(call.System, "VP-002 · b1 · (chưa có chuỗi)") ||
		!strings.Contains(call.System, "lần rà soát đầu") {
		t.Fatalf("the review prompt is filled:\n%s", call.System)
	}
	if !st.Review.Reviewed || st.Review.OpenCount != 2 {
		t.Fatalf("two open findings: %+v", st.Review)
	}
	for name, reply := range map[string]string{
		"unknown key":         reviewReply(finding("clarity", "VP-099", "", "x", "")),
		"motif field on unit": reviewReply(finding("clarity", "VP-001", "role", "x", "")),
		"unit field on motif": reviewReply(finding("clarity", "MOTIF-01", "pacing", "x", "")),
	} {
		f.queue(domain.PromptRoleSequenceReview, reply)
		if _, err := f.desk.Review(f.ctx, project, ""); !errors.Is(err, domain.ErrUnreadableReply) {
			t.Errorf("%s: want ErrUnreadableReply, got %v", name, err)
		}
	}
}

func TestSequenceAI_ModelAndAIOutsideTheSystem(t *testing.T) {
	f := newSeqFixture(t)
	if _, err := f.desk.GenerateMotifs(f.ctx, project, ""); !errors.Is(err, ErrLLMNotConfigured) {
		t.Errorf("a desk with no AI: %v", err)
	}
	g := newSeqAIFixture(t)
	if _, err := g.desk.SetModel(g.ctx, project, "nope"); !errors.Is(err, ErrUnknownModel) {
		t.Errorf("an unknown model: %v", err)
	}
	st := g.must(g.desk.State(g.ctx, project))
	if st.DefaultModel != "ollama" {
		t.Errorf("the screen shows the step default: %q", st.DefaultModel)
	}

	llm := g.llm
	g.desk.WithAI(NewExternalAIProvider(llm, llm), seedPrompts{}, seqModels{}, 1000)
	run := NewExternalAIRun(ExternalAIPrompt, "")
	before := fmt.Sprint(g.store.units)
	_, err := g.desk.GenerateMotifs(WithExternalAI(context.Background(), run), project, "")
	prompt, ok := run.Prompt()
	if !errors.Is(err, ErrExternalAIPromptBuilt) || !ok || !strings.Contains(prompt, anchorClock) || fmt.Sprint(g.store.units) != before {
		t.Fatalf("prompt mode builds the prompt and stops: %v %v", err, ok)
	}
	run = NewExternalAIRun(ExternalAIPaste, motifsReply(motifJSON("", "Dán", anchorBee)))
	st, err = g.desk.GenerateMotifs(WithExternalAI(context.Background(), run), project, "")
	if err != nil || len(st.Motifs.Motifs) != 1 || st.Motifs.Motifs[0].Name != "Dán" {
		t.Fatalf("paste mode stores the pasted motifs: %v", err)
	}
}
