package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"authoring/internal/domain"
)

const factQuote = "Each fig species is typically pollinated by one or a few species of wasp."

// factsFixture is a research desk with a brief, one question (Q1), a verifier
// whose pages all hold factQuote, and the generic unit API on top.
type factsFixture struct {
	*researchFixture
	fetcher *fakeFetcher
	units   *ReviewUnitsUseCase
}

func newFactsFixture(t *testing.T) *factsFixture {
	t.Helper()
	f := &factsFixture{researchFixture: newResearchFixture()}
	f.fetcher = &fakeFetcher{pages: map[string]string{}}
	for i := 1; i <= 20; i++ {
		f.fetcher.pages[fmt.Sprintf("https://example.org/%d", i)] = "Intro. " + factQuote + " End."
	}
	verifier := NewSourceVerifier(f.store, f.fetcher)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	verifier.Start(ctx)
	f.uc.WithVerifier(verifier)
	f.units = NewReviewUnitsUseCase(f.store, f.project).
		Register(domain.UnitKindResearchQuestion, f.uc.QuestionUnitKind()).
		Register(domain.UnitKindFact, f.uc.FactUnitKind()).
		Register(domain.UnitKindResearch, f.uc.GateUnitKind())
	f.seedBrief(t, domain.UnitApproved)
	if _, err := f.uc.AddQuestion(ctx, "p1", "Mỗi loài sung có ong riêng không?"); err != nil {
		t.Fatal(err)
	}
	return f
}

func claimRow(id string, n int) string {
	return fmt.Sprintf(`{"id":%q,"question":"Q1","claim_vi":"Dữ kiện số %d","type":"fact","source_title":"T","publisher":"P","year":2003,"url":"https://example.org/%d","quote":%q,"confidence":"high","note":""}`,
		id, n, n, factQuote)
}

func answer(rows ...string) string {
	return "Kết quả:\n```json\n{\"schema\":\"conceptflow.research.v1\",\"claims\":[" + strings.Join(rows, ",") + "]}\n```"
}

func (f *factsFixture) fact(t *testing.T, id string) FactView {
	t.Helper()
	v, err := f.uc.factView(context.Background(), "p1", id)
	if err != nil {
		t.Fatalf("fact %s: %v", id, err)
	}
	return v
}

func (f *factsFixture) waitForLinks(t *testing.T, want domain.LinkState, ids ...string) {
	t.Helper()
	waitUntil(t, func() bool {
		for _, id := range ids {
			v, err := f.uc.factView(context.Background(), "p1", id)
			if err != nil || v.Link.State != want {
				return false
			}
		}
		return true
	})
}

func (f *factsFixture) approve(t *testing.T) {
	t.Helper()
	if _, err := f.units.Approve(context.Background(), "p1", domain.UnitKindResearch, domain.ResearchUnitKey); err != nil {
		t.Fatalf("approve: %v", err)
	}
}

func (f *factsFixture) gateStatus(t *testing.T) domain.UnitStatus {
	t.Helper()
	state, err := f.uc.State(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	return state.Gate.Status
}

func TestIngestPasteReadsTwelveFactsAndLogsTheBadRow(t *testing.T) {
	f := newFactsFixture(t)
	var rows []string
	var ids []string
	for i := 1; i <= 12; i++ {
		id := fmt.Sprintf("F%02d", i)
		ids = append(ids, id)
		rows = append(rows, claimRow(id, i))
	}
	rows = append(rows, `{"id":"F13","question":"Q1","claim_vi":"thiếu type"}`)

	result, err := f.uc.IngestPaste(context.Background(), "p1", answer(rows...))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Added) != 12 || len(result.Updated) != 0 {
		t.Fatalf("added %d, updated %d, want 12 and 0", len(result.Added), len(result.Updated))
	}
	if len(result.Rejected) != 1 || result.Rejected[0].Row != 13 || !strings.Contains(result.Rejected[0].Reason, "type") {
		t.Fatalf("rejected = %+v, want row 13 and its reason", result.Rejected)
	}
	if len(f.store.pastes) != 1 || f.store.pastes[0].N != 1 || len(f.store.pastes[0].Rejected) != 1 || len(f.store.pastes[0].Added) != 12 {
		t.Fatalf("the paste must be logged with what was read and left out, got %+v", f.store.pastes)
	}
	state, err := f.uc.State(context.Background(), "p1")
	if err != nil || len(state.Facts) != 12 {
		t.Fatalf("facts in the table = %d, %v, want 12", len(state.Facts), err)
	}
	if got := f.fact(t, "F03"); got.ClaimVI != "Dữ kiện số 3" || !got.Keep || got.Year != "2003" || got.Mine == nil || len(got.Mine) != 0 {
		t.Errorf("F03 = %+v, want the pasted fact, kept, and none of its fields the Creator's", got)
	}
	f.waitForLinks(t, domain.LinkOK, ids...)
}

func TestIngestPasteUpdatesTheSameCodesAndKeepsTheCreatorsChoices(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F01", 1), claimRow("F02", 2))); err != nil {
		t.Fatal(err)
	}
	f.waitForLinks(t, domain.LinkOK, "F01", "F02")
	must, drop := true, false
	if _, err := f.uc.SetFactFlags(ctx, "p1", "F01", FactFlags{Must: &must}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.SetFactFlags(ctx, "p1", "F02", FactFlags{Keep: &drop}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.units.Edit(ctx, "p1", domain.UnitKindFact, "F01", map[string]string{"claim_vi": "Bản của Creator"}); err != nil {
		t.Fatal(err)
	}
	callsBefore := f.fetcher.callCount()

	second := strings.Replace(claimRow("F01", 1), "Dữ kiện số 1", "Bản dán lần hai", 1)
	second2 := strings.Replace(claimRow("F02", 2), "Dữ kiện số 2", "F02 lần hai", 1)
	result, err := f.uc.IngestPaste(ctx, "p1", answer(second, second2, claimRow("F03", 3)))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(result.Updated, ",") != "F01,F02" || strings.Join(result.Added, ",") != "F03" {
		t.Fatalf("updated %v added %v, want F01,F02 updated and F03 added", result.Updated, result.Added)
	}
	state, _ := f.uc.State(ctx, "p1")
	if len(state.Facts) != 3 {
		t.Fatalf("a second paste must not duplicate codes, got %d facts", len(state.Facts))
	}
	f1, f2 := f.fact(t, "F01"), f.fact(t, "F02")
	if f1.ClaimVI != "Bản của Creator" || !f1.Must {
		t.Errorf("F01 = %+v, want the Creator's wording and must-say kept", f1)
	}
	if f2.ClaimVI != "F02 lần hai" || f2.Keep {
		t.Errorf("F02 = %+v, want the new wording but still dropped", f2)
	}
	f.waitForLinks(t, domain.LinkOK, "F03")
	if got := f.fetcher.callCount() - callsBefore; got != 1 {
		t.Errorf("only the new fact's page is fetched: %d fetches since the first paste, want 1", got)
	}
	if len(f.store.pastes) != 2 || f.store.pastes[1].N != 2 {
		t.Errorf("each paste has its own log line: %+v", f.store.pastes)
	}
}

func TestIngestPasteChecksAgainWhenTheSourceChanges(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F01", 1))); err != nil {
		t.Fatal(err)
	}
	f.waitForLinks(t, domain.LinkOK, "F01")
	moved := strings.Replace(claimRow("F01", 1), "example.org/1", "example.org/2", 1)
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(moved)); err != nil {
		t.Fatal(err)
	}
	if got := f.fact(t, "F01"); got.URL != "https://example.org/2" {
		t.Fatalf("URL = %q, want the new link", got.URL)
	}
	f.waitForLinks(t, domain.LinkOK, "F01")
	if f.fetcher.callCount() != 2 {
		t.Errorf("a new link means a new check: %d fetches, want 2", f.fetcher.callCount())
	}
}

func TestIngestPasteWithoutResearchJSONReadsAndLogsNothing(t *testing.T) {
	f := newFactsFixture(t)
	if _, err := f.uc.IngestPaste(context.Background(), "p1", "Chỉ là một đoạn văn, không có khối JSON."); !errors.Is(err, domain.ErrNoResearchJSON) {
		t.Fatalf("err = %v, want ErrNoResearchJSON", err)
	}
	if len(f.store.pastes) != 0 {
		t.Fatalf("nothing was read, nothing is logged: %+v", f.store.pastes)
	}
	state, _ := f.uc.State(context.Background(), "p1")
	if len(state.Facts) != 0 {
		t.Fatalf("facts = %d, want none", len(state.Facts))
	}
}

func TestIngestPasteRejectsUnknownQuestionsAndLockedProjects(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	other := strings.Replace(claimRow("F01", 1), `"question":"Q1"`, `"question":"Q9"`, 1)
	result, err := f.uc.IngestPaste(ctx, "p1", answer(other))
	if err != nil || len(result.Added) != 0 || len(result.Rejected) != 1 || !strings.Contains(result.Rejected[0].Reason, "Q9") {
		t.Fatalf("result = %+v, %v, want the Q9 row rejected by name", result, err)
	}
	f.project.status = domain.StatusAwaitingReview
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F01", 1))); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("locked project: err = %v, want ErrInvalidStatus", err)
	}
}

func TestChangingTheTableSendsAnApprovedTableBackToReview(t *testing.T) {
	ctx := context.Background()
	drop := false
	changes := map[string]func(f *factsFixture) error{
		"paste a new fact": func(f *factsFixture) error {
			_, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F02", 2)))
			return err
		},
		"add a fact by hand": func(f *factsFixture) error {
			_, err := f.uc.AddFact(ctx, "p1", FactInput{Question: "Q1", ClaimVI: "Tay", Type: domain.FactTypeFact})
			return err
		},
		"drop a fact": func(f *factsFixture) error {
			_, err := f.uc.SetFactFlags(ctx, "p1", "F01", FactFlags{Keep: &drop})
			return err
		},
		"change a type": func(f *factsFixture) error {
			typ := domain.FactTypeHypothesis
			_, err := f.uc.SetFactFlags(ctx, "p1", "F01", FactFlags{Type: &typ})
			return err
		},
		"edit a fact": func(f *factsFixture) error {
			_, err := f.units.Edit(ctx, "p1", domain.UnitKindFact, "F01", map[string]string{"claim_vi": "Đổi chữ"})
			return err
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			f := newFactsFixture(t)
			if _, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F01", 1))); err != nil {
				t.Fatal(err)
			}
			f.approve(t)
			if f.gateStatus(t) != domain.UnitApproved {
				t.Fatal("setup: the table must be approved")
			}
			if err := change(f); err != nil {
				t.Fatal(err)
			}
			if f.gateStatus(t) != domain.UnitPending {
				t.Fatalf("%s must send the approved table back to review", name)
			}
		})
	}
}

func TestLinkResultsDoNotReopenAnApprovedTable(t *testing.T) {
	f := newFactsFixture(t)
	if _, err := f.uc.IngestPaste(context.Background(), "p1", answer(claimRow("F01", 1))); err != nil {
		t.Fatal(err)
	}
	f.approve(t)
	f.waitForLinks(t, domain.LinkOK, "F01")
	if f.gateStatus(t) != domain.UnitApproved {
		t.Fatal("a finished link check must leave the approval alone")
	}
}

func TestAddFactNumbersItAndMarksItTheCreators(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F01", 1), claimRow("F07", 7))); err != nil {
		t.Fatal(err)
	}
	got, err := f.uc.AddFact(ctx, "p1", FactInput{
		Question: "Q1", ClaimVI: "  Dữ kiện tay  ", Type: domain.FactTypeMisconception,
		SourceTitle: "Bách khoa", URL: "https://example.org/3", Quote: factQuote,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "F08" || got.ClaimVI != "Dữ kiện tay" || !got.Keep {
		t.Fatalf("added = %+v, want F08 after the highest code, trimmed and kept", got)
	}
	for _, name := range []string{"claim_vi", "question", "type", "source_title", "url", "quote"} {
		found := false
		for _, m := range got.Mine {
			found = found || m == name
		}
		if !found {
			t.Errorf("field %q the Creator filled must be marked theirs, Mine = %v", name, got.Mine)
		}
	}
	f.waitForLinks(t, domain.LinkOK, "F08")
	noLink, err := f.uc.AddFact(ctx, "p1", FactInput{Question: "Q1", ClaimVI: "Không nguồn", Type: domain.FactTypeUnknown})
	if err != nil || noLink.Link.State != domain.LinkNone {
		t.Fatalf("a fact with no link has no source: %+v, %v", noLink, err)
	}
}

func TestAddFactValidation(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	cases := map[string]FactInput{
		"blank claim":      {Question: "Q1", ClaimVI: "  ", Type: domain.FactTypeFact},
		"unknown type":     {Question: "Q1", ClaimVI: "x", Type: "rumor"},
		"unknown question": {Question: "Q9", ClaimVI: "x", Type: domain.FactTypeFact},
		"bad link":         {Question: "Q1", ClaimVI: "x", Type: domain.FactTypeFact, URL: "javascript:alert(1)"},
		"link with login":  {Question: "Q1", ClaimVI: "x", Type: domain.FactTypeFact, URL: "https://u:p@example.org/"},
	}
	for name, in := range cases {
		if _, err := f.uc.AddFact(ctx, "p1", in); !errors.Is(err, domain.ErrInvalidUnitEdit) {
			t.Errorf("%s: err = %v, want ErrInvalidUnitEdit", name, err)
		}
	}
	state, _ := f.uc.State(ctx, "p1")
	if len(state.Facts) != 0 {
		t.Fatalf("a refused fact must not be stored, got %d", len(state.Facts))
	}
}

func TestEditFactQueuesTheCheckOnlyWhenTheSourceChanges(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F01", 1))); err != nil {
		t.Fatal(err)
	}
	f.waitForLinks(t, domain.LinkOK, "F01")

	if _, err := f.units.Edit(ctx, "p1", domain.UnitKindFact, "F01", map[string]string{"claim_vi": "Chỉ đổi chữ", "note": "ghi chú"}); err != nil {
		t.Fatal(err)
	}
	if got := f.fact(t, "F01"); got.Link.State != domain.LinkOK || got.ClaimVI != "Chỉ đổi chữ" || got.Note != "ghi chú" {
		t.Fatalf("F01 = %+v, want the text changed and the link result kept", got)
	}
	if f.fetcher.callCount() != 1 {
		t.Fatalf("a text edit does not fetch again: %d fetches", f.fetcher.callCount())
	}

	if _, err := f.units.Edit(ctx, "p1", domain.UnitKindFact, "F01", map[string]string{"url": "https://example.org/5"}); err != nil {
		t.Fatal(err)
	}
	f.waitForLinks(t, domain.LinkOK, "F01")
	if f.fetcher.callCount() != 2 {
		t.Fatalf("a new link is fetched: %d fetches, want 2", f.fetcher.callCount())
	}
	if _, err := f.units.Edit(ctx, "p1", domain.UnitKindFact, "F01", map[string]string{"url": ""}); err != nil {
		t.Fatal(err)
	}
	if got := f.fact(t, "F01"); got.Link.State != domain.LinkNone {
		t.Fatalf("removing the link leaves a fact with no source, got %+v", got.Link)
	}
}

func TestEditFactRefusals(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F01", 1))); err != nil {
		t.Fatal(err)
	}
	for name, fields := range map[string]map[string]string{
		"nothing to save":  {},
		"blank claim":      {"claim_vi": " "},
		"type here":        {"type": "hypothesis"},
		"unknown field":    {"must": "true"},
		"bad link":         {"url": "ftp://example.org/"},
		"unknown question": {"question": "Q9"},
	} {
		if _, err := f.units.Edit(ctx, "p1", domain.UnitKindFact, "F01", fields); !errors.Is(err, domain.ErrInvalidUnitEdit) {
			t.Errorf("%s: err = %v, want ErrInvalidUnitEdit", name, err)
		}
	}
	if _, err := f.units.Edit(ctx, "p1", domain.UnitKindFact, "F99", map[string]string{"claim_vi": "x"}); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("missing fact: err = %v, want ErrUnitNotFound", err)
	}
	if _, err := f.units.Redo(ctx, "p1", domain.UnitKindFact, "F01", "ghi chú"); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("redo of a fact: err = %v, want ErrInvalidUnitEdit", err)
	}
}

func TestSetFactFlagsMarksATypeChangeTheCreators(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F01", 1))); err != nil {
		t.Fatal(err)
	}
	typ := domain.FactTypeHypothesis
	got, err := f.uc.SetFactFlags(ctx, "p1", "F01", FactFlags{Type: &typ})
	if err != nil || got.Type != domain.FactTypeHypothesis {
		t.Fatalf("got %+v, %v, want the type changed", got, err)
	}
	// A later paste says "fact" again; the Creator's type stays.
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F01", 1))); err != nil {
		t.Fatal(err)
	}
	if again := f.fact(t, "F01"); again.Type != domain.FactTypeHypothesis {
		t.Fatalf("type = %q, want the Creator's choice kept", again.Type)
	}
	bad := domain.FactType("rumor")
	if _, err := f.uc.SetFactFlags(ctx, "p1", "F01", FactFlags{Type: &bad}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("unknown type: err = %v, want ErrInvalidUnitEdit", err)
	}
	if _, err := f.uc.SetFactFlags(ctx, "p1", "F77", FactFlags{}); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("missing fact: err = %v, want ErrUnitNotFound", err)
	}
}

func TestVerifyChecksTheNamedFactsOrAllKeptOnes(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F01", 1), claimRow("F02", 2))); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.AddFact(ctx, "p1", FactInput{Question: "Q1", ClaimVI: "Không link", Type: domain.FactTypeUnknown}); err != nil {
		t.Fatal(err)
	}
	f.waitForLinks(t, domain.LinkOK, "F01", "F02")
	before := f.fetcher.callCount()

	if err := f.uc.Verify(ctx, "p1", []string{"F02"}); err != nil {
		t.Fatal(err)
	}
	f.waitForLinks(t, domain.LinkOK, "F02")
	waitUntil(t, func() bool { return f.fetcher.callCount() == before+1 })

	if err := f.uc.Verify(ctx, "p1", nil); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, func() bool { return f.fetcher.callCount() == before+3 })
	if got := f.fact(t, "F03"); got.Link.State != domain.LinkNone {
		t.Errorf("a fact with no link stays with no source: %+v", got.Link)
	}
	if err := f.uc.Verify(ctx, "p1", []string{"F55"}); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Errorf("missing fact: err = %v, want ErrUnitNotFound", err)
	}
}

func TestFactWithoutAVerifierSaysItCannotBeChecked(t *testing.T) {
	f := newResearchFixture()
	ctx := context.Background()
	f.seedBrief(t, domain.UnitApproved)
	if _, err := f.uc.AddQuestion(ctx, "p1", "Câu hỏi?"); err != nil {
		t.Fatal(err)
	}
	got, err := f.uc.AddFact(ctx, "p1", FactInput{Question: "Q1", ClaimVI: "x", Type: domain.FactTypeFact, URL: "https://example.org/1", Quote: factQuote})
	if err != nil || got.Link.State != domain.LinkUnreachable || got.Link.Why == "" {
		t.Fatalf("link = %+v, %v, want an explained failure rather than a check that never ends", got.Link, err)
	}
}

func TestApprovingTheTableAndSkippingTheStep(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	if _, err := f.units.Approve(ctx, "p1", domain.UnitKindResearch, domain.ResearchUnitKey); !errors.Is(err, domain.ErrResearchEmpty) {
		t.Fatalf("approving an empty table: err = %v, want ErrResearchEmpty", err)
	}
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F01", 1))); err != nil {
		t.Fatal(err)
	}
	drop := false
	if _, err := f.uc.SetFactFlags(ctx, "p1", "F01", FactFlags{Keep: &drop}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.units.Approve(ctx, "p1", domain.UnitKindResearch, domain.ResearchUnitKey); !errors.Is(err, domain.ErrResearchEmpty) {
		t.Fatalf("a table of dropped facts is empty: err = %v", err)
	}
	keep := true
	if _, err := f.uc.SetFactFlags(ctx, "p1", "F01", FactFlags{Keep: &keep}); err != nil {
		t.Fatal(err)
	}
	f.approve(t)
	if state, _ := f.uc.State(ctx, "p1"); !state.Gate.Resolved || state.Gate.Skipped {
		t.Fatalf("an approved table resolves the step: %+v", state.Gate)
	}
	if _, err := f.units.Unapprove(ctx, "p1", domain.UnitKindResearch, domain.ResearchUnitKey); err != nil {
		t.Fatal(err)
	}
	if state, _ := f.uc.State(ctx, "p1"); state.Gate.Resolved {
		t.Fatal("an unapproved table does not resolve the step")
	}

	view, err := f.uc.SetSkipped(ctx, "p1", true)
	if err != nil || !view.Skipped || !view.Resolved {
		t.Fatalf("skipping = %+v, %v, want a resolved step", view, err)
	}
	if _, err := f.units.Approve(ctx, "p1", domain.UnitKindResearch, domain.ResearchUnitKey); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("approving while skipped: err = %v, want ErrInvalidStatus", err)
	}
	state, _ := f.uc.State(ctx, "p1")
	if len(state.Facts) != 1 || len(state.Questions) != 1 {
		t.Errorf("skipping keeps the questions and facts: %d facts, %d questions", len(state.Facts), len(state.Questions))
	}
	view, err = f.uc.SetSkipped(ctx, "p1", false)
	if err != nil || view.Skipped || view.Resolved {
		t.Fatalf("coming back = %+v, %v, want an unresolved step", view, err)
	}
}

func TestGateRefusesEditAndRedoAndLockedChanges(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	if _, err := f.units.Edit(ctx, "p1", domain.UnitKindResearch, domain.ResearchUnitKey, map[string]string{"x": "y"}); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("edit of the gate: err = %v", err)
	}
	if _, err := f.units.Redo(ctx, "p1", domain.UnitKindResearch, domain.ResearchUnitKey, ""); !errors.Is(err, domain.ErrInvalidUnitEdit) {
		t.Errorf("redo of the gate: err = %v", err)
	}
	f.project.status = domain.StatusAwaitingReview
	if _, err := f.uc.SetSkipped(ctx, "p1", true); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("locked project: err = %v, want ErrInvalidStatus", err)
	}
}

func TestFactImpactComesFromTheRegisteredFinders(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F01", 1))); err != nil {
		t.Fatal(err)
	}
	items, err := f.units.Impact(ctx, "p1", domain.UnitKindFact, "F01")
	if err != nil || len(items) != 0 {
		t.Fatalf("no later step uses facts yet: items = %v, err = %v", items, err)
	}
	f.uc.RegisterFactDependents(func(_ context.Context, _, factID string) ([]string, error) {
		return []string{"Beat B2 dùng " + factID}, nil
	})
	items, err = f.units.Impact(ctx, "p1", domain.UnitKindFact, "F01")
	if err != nil || len(items) != 1 || items[0] != "Beat B2 dùng F01" {
		t.Fatalf("items = %v, err = %v, want the finder's line", items, err)
	}
}

func TestStateCoversEachQuestionByIndependentSources(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	a := claimRow("F01", 1)
	b := strings.Replace(claimRow("F02", 2), "https://example.org/2", "https://other.example.net/2", 1)
	f.fetcher.pages["https://other.example.net/2"] = factQuote
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(a, b)); err != nil {
		t.Fatal(err)
	}
	f.waitForLinks(t, domain.LinkOK, "F01", "F02")
	state, err := f.uc.State(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Coverage) != 1 || state.Coverage[0].Question != "Q1" || state.Coverage[0].Sources != 2 || state.Coverage[0].Weak {
		t.Fatalf("coverage = %+v, want two independent sources for Q1", state.Coverage)
	}
	raw, _ := json.Marshal(state.Gate)
	if !strings.Contains(string(raw), `"resolved":false`) {
		t.Errorf("gate JSON = %s", raw)
	}
}

// The Creator may approve the table while a source is unchecked or its quote was
// not found: approval needs one kept fact, not a verified link.
func TestApproveAllowsFactsWithUnverifiedSources(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	f.seedFact(t, "F01", domain.Fact{
		Question: "Q1", ClaimVI: "Dữ kiện chưa xác minh", Type: domain.FactTypeFact, URL: "https://example.org/1",
		Quote: "không có trên trang", Keep: true, Link: domain.FactLink{State: domain.LinkBad, Why: "không thấy đoạn trích"},
	})
	if _, err := f.units.Approve(ctx, "p1", domain.UnitKindResearch, domain.ResearchUnitKey); err != nil {
		t.Fatalf("a table with an unverified source must be approvable: %v", err)
	}
}

// The facts live in project_research and the later steps read them there: a pasted
// answer is saved as rows and survives a read back.
func TestPastedFactsAreStoredForLaterSteps(t *testing.T) {
	f := newFactsFixture(t)
	ctx := context.Background()
	if _, err := f.uc.IngestPaste(ctx, "p1", answer(claimRow("F01", 1), claimRow("F02", 2))); err != nil {
		t.Fatal(err)
	}
	stored, err := f.store.ListResearchFacts(ctx, "p1")
	if err != nil || len(stored) != 2 || stored[0].ID != "F01" || stored[1].Fact.ClaimVI != "Dữ kiện số 2" || !stored[0].Fact.Keep {
		t.Fatalf("stored facts = %+v, %v, want F01 and F02 kept", stored, err)
	}
}
