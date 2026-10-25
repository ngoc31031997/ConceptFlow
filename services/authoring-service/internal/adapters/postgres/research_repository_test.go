package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

func researchRepo(t *testing.T, pid string) (*PromptTemplateRepository, context.Context) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := NewPool(ctx, url, 4)
	if err != nil {
		t.Fatal(err)
	}
	r := NewPromptTemplateRepository(pool)
	t.Cleanup(func() { _ = r.DeleteAuthoring(ctx, pid) })
	_ = r.DeleteAuthoring(ctx, pid)
	return r, ctx
}

func saveFact(t *testing.T, r *PromptTemplateRepository, ctx context.Context, pid, key string, f domain.Fact) {
	t.Helper()
	if err := r.SaveResearchFact(ctx, pid, domain.StoredFact{ID: key, Fact: f}); err != nil {
		t.Fatal(err)
	}
}

func readFact(t *testing.T, r *PromptTemplateRepository, ctx context.Context, pid, key string) domain.Fact {
	t.Helper()
	s, err := r.GetResearchFact(ctx, pid, key)
	if err != nil {
		t.Fatal(err)
	}
	return s.Fact
}

// A fact survives a round trip through project_research, lists in code order
// and replaces the row of the same code; a missing code is not found.
func TestResearchFactRoundTripInPostgres(t *testing.T) {
	const pid = "research-fact-test"
	r, ctx := researchRepo(t, pid)
	want := domain.Fact{
		Question: "Q1", ClaimVI: "Dữ kiện", Type: domain.FactTypeHypothesis, SourceTitle: "Tựa", Publisher: "NXB", Year: "2020",
		URL: "https://a.org/x", Quote: "đoạn trích", Confidence: "high", Note: "ghi chú", Must: true, Keep: true,
		Link: domain.FactLink{State: domain.LinkBad, Why: "vì sao", Nearest: "câu gần nhất", CheckedAt: "2026-10-24T10:00:00Z"},
	}
	if err := r.SaveResearchFact(ctx, pid, domain.StoredFact{ID: "F10", Fact: want, CreatorFields: []string{"claim_vi"}}); err != nil {
		t.Fatal(err)
	}
	saveFact(t, r, ctx, pid, "F02", domain.Fact{ClaimVI: "hai", Type: domain.FactTypeFact, Keep: true, Link: domain.FactLink{State: domain.LinkNone}})

	got, err := r.GetResearchFact(ctx, pid, "F10")
	if err != nil || got.Fact != want || len(got.CreatorFields) != 1 || got.CreatorFields[0] != "claim_vi" {
		t.Fatalf("round trip = %+v, %v, want %+v with the Creator's field", got, err, want)
	}
	list, err := r.ListResearchFacts(ctx, pid)
	if err != nil || len(list) != 2 || list[0].ID != "F02" || list[1].ID != "F10" {
		t.Fatalf("list = %+v, %v, want F02 before F10", list, err)
	}
	saveFact(t, r, ctx, pid, "F02", domain.Fact{ClaimVI: "hai, sửa", Type: domain.FactTypeFact, Keep: false, Link: domain.FactLink{State: domain.LinkNone}})
	if again := readFact(t, r, ctx, pid, "F02"); again.ClaimVI != "hai, sửa" || again.Keep {
		t.Fatalf("saving the same code must replace the row, got %+v", again)
	}
	if _, err := r.GetResearchFact(ctx, pid, "F99"); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Fatalf("a missing fact: err = %v, want ErrUnitNotFound", err)
	}
	if err := r.DeleteAuthoring(ctx, pid); err != nil {
		t.Fatal(err)
	}
	if list, _ := r.ListResearchFacts(ctx, pid); len(list) != 0 {
		t.Fatalf("deleting the project's authoring must remove its facts, left %+v", list)
	}
}

// A unit built from the brief is stale once the brief's content changes, and a
// unit built from nothing never is.
func TestUnitStaleAgainstItsSourceInPostgres(t *testing.T) {
	const pid = "research-stale-test"
	r, ctx := researchRepo(t, pid)

	brief, _ := json.Marshal(domain.CreativeBrief{Audience: "Q", Tone: "T"})
	if _, err := r.SaveUnit(ctx, domain.ReviewUnit{
		ProjectID: pid, Kind: domain.UnitKindBrief, Key: domain.BriefUnitKey, Status: domain.UnitApproved, Content: brief,
	}); err != nil {
		t.Fatal(err)
	}
	sha, err := r.UnitSHA(ctx, pid, domain.UnitKindBrief, domain.BriefUnitKey)
	if err != nil || sha == "" {
		t.Fatalf("UnitSHA = %q, %v, want a hash", sha, err)
	}
	if none, err := r.UnitSHA(ctx, pid, "nothing", "here"); err != nil || none != "" {
		t.Fatalf("UnitSHA of a missing unit = %q, %v, want empty", none, err)
	}
	text, _ := json.Marshal(domain.ResearchQuestion{Text: "Một câu hỏi?"})
	built, err := r.SaveUnit(ctx, domain.ReviewUnit{
		ProjectID: pid, Kind: domain.UnitKindResearchQuestion, Key: "Q1", Content: text,
		InputRef: "brief:main", InputSHA: sha,
	})
	if err != nil {
		t.Fatal(err)
	}
	if built.Stale || built.InputRef != "brief:main" {
		t.Fatalf("a unit built from the current brief is not stale: %+v", built)
	}
	handWritten, err := r.SaveUnit(ctx, domain.ReviewUnit{
		ProjectID: pid, Kind: domain.UnitKindResearchQuestion, Key: "Q2", Content: text,
	})
	if err != nil || handWritten.Stale {
		t.Fatalf("a unit built from nothing is never stale: %+v, %v", handWritten, err)
	}

	changed, _ := json.Marshal(domain.CreativeBrief{Audience: "Q2", Tone: "T"})
	if _, err := r.SaveUnit(ctx, domain.ReviewUnit{
		ProjectID: pid, Kind: domain.UnitKindBrief, Key: domain.BriefUnitKey, Status: domain.UnitApproved, Content: changed,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetUnit(ctx, pid, domain.UnitKindResearchQuestion, "Q1")
	if err != nil || !got.Stale {
		t.Fatalf("a unit whose source changed must be stale: %+v, %v", got, err)
	}
	if again, err := r.GetUnit(ctx, pid, domain.UnitKindResearchQuestion, "Q2"); err != nil || again.Stale {
		t.Fatalf("the hand-written unit stays fresh: %+v, %v", again, err)
	}
}

func TestListAndDeleteUnitsInPostgres(t *testing.T) {
	const pid = "research-list-test"
	r, ctx := researchRepo(t, pid)
	for _, key := range []string{"Q10", "Q2", "Q1"} {
		if _, err := r.SaveUnit(ctx, domain.ReviewUnit{
			ProjectID: pid, Kind: domain.UnitKindResearchQuestion, Key: key, Content: []byte(`{"text":"x"}`),
		}); err != nil {
			t.Fatal(err)
		}
	}
	units, err := r.ListUnits(ctx, pid, domain.UnitKindResearchQuestion)
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 3 || units[0].Key != "Q1" || units[1].Key != "Q2" || units[2].Key != "Q10" {
		t.Fatalf("units = %+v, want Q1, Q2, Q10 in that order", units)
	}
	if err := r.DeleteUnit(ctx, pid, domain.UnitKindResearchQuestion, "Q2"); err != nil {
		t.Fatal(err)
	}
	if err := r.DeleteUnit(ctx, pid, domain.UnitKindResearchQuestion, "Q2"); err != nil {
		t.Fatalf("deleting a missing unit must not fail: %v", err)
	}
	if units, _ := r.ListUnits(ctx, pid, domain.UnitKindResearchQuestion); len(units) != 2 {
		t.Fatalf("after delete: %d units, want 2", len(units))
	}
	if other, err := r.ListUnits(ctx, pid, domain.UnitKindFact); err != nil || len(other) != 0 {
		t.Fatalf("another kind lists empty, got %+v, %v", other, err)
	}
}

// A check result is stored only while the fact still has the url and quote the
// check started from.
func TestSaveFactLinkOnlyWhileFactUnchangedInPostgres(t *testing.T) {
	const pid = "research-link-test"
	r, ctx := researchRepo(t, pid)
	saveFact(t, r, ctx, pid, "F01", domain.Fact{
		Question: "Q1", ClaimVI: "Dữ kiện", Type: domain.FactTypeFact, URL: "https://a.org/x", Quote: "đoạn trích gốc", Keep: true,
		Link: domain.FactLink{State: domain.LinkRunning},
	})

	stored, err := r.SaveFactLink(ctx, pid, "F01", "https://a.org/x", "đoạn trích gốc", domain.FactLink{State: domain.LinkOK, CheckedAt: "2026-10-18T10:00:00Z"})
	if err != nil || !stored {
		t.Fatalf("a check of the unchanged fact must be stored: %v, %v", stored, err)
	}
	if got := readFact(t, r, ctx, pid, "F01"); got.Link.State != domain.LinkOK || got.ClaimVI != "Dữ kiện" || !got.Keep || !got.Must {
		t.Fatalf("only the link and the must-say flag may change, got %+v", got)
	}

	stored, err = r.SaveFactLink(ctx, pid, "F01", "https://old.example/", "đoạn trích gốc", domain.FactLink{State: domain.LinkBad})
	if err != nil || stored {
		t.Fatalf("a check of an older url must be dropped: %v, %v", stored, err)
	}
	stored, err = r.SaveFactLink(ctx, pid, "F01", "https://a.org/x", "đoạn trích cũ", domain.FactLink{State: domain.LinkBad})
	if err != nil || stored {
		t.Fatalf("a check of an older quote must be dropped: %v, %v", stored, err)
	}
	if got := readFact(t, r, ctx, pid, "F01"); got.Link.State != domain.LinkOK {
		t.Fatalf("dropped results must leave the link alone, got %+v", got.Link)
	}
}

// A failed or unreachable check never makes a fact must-say, and a check never
// clears the flag.
func TestSaveFactLinkSetsMustOnlyOnMatchInPostgres(t *testing.T) {
	const pid = "research-link-must-test"
	r, ctx := researchRepo(t, pid)
	saveFact(t, r, ctx, pid, "F01", domain.Fact{ClaimVI: "a", Type: domain.FactTypeFact, URL: "https://a.org/", Quote: "q", Link: domain.FactLink{State: domain.LinkRunning}})
	saveFact(t, r, ctx, pid, "F02", domain.Fact{ClaimVI: "b", Type: domain.FactTypeFact, URL: "https://b.org/", Quote: "q", Must: true, Link: domain.FactLink{State: domain.LinkRunning}})

	if _, err := r.SaveFactLink(ctx, pid, "F01", "https://a.org/", "q", domain.FactLink{State: domain.LinkBad}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.SaveFactLink(ctx, pid, "F02", "https://b.org/", "q", domain.FactLink{State: domain.LinkBad}); err != nil {
		t.Fatal(err)
	}
	if got := readFact(t, r, ctx, pid, "F01"); got.Must {
		t.Fatalf("a failed check must not tick must-say, got %+v", got)
	}
	if got := readFact(t, r, ctx, pid, "F02"); !got.Must {
		t.Fatalf("a failed check must not clear must-say, got %+v", got)
	}
}

func TestResetRunningFactLinksInPostgres(t *testing.T) {
	const pid = "research-reset-test"
	r, ctx := researchRepo(t, pid)
	saveFact(t, r, ctx, pid, "F01", domain.Fact{ClaimVI: "a", Type: domain.FactTypeFact, URL: "https://a.org/", Link: domain.FactLink{State: domain.LinkRunning}})
	saveFact(t, r, ctx, pid, "F02", domain.Fact{ClaimVI: "b", Type: domain.FactTypeFact, URL: "https://b.org/", Link: domain.FactLink{State: domain.LinkOK}})

	n, err := r.ResetRunningFactLinks(ctx)
	if err != nil || n < 1 {
		t.Fatalf("reset = %d, %v, want at least the one running fact", n, err)
	}
	if got := readFact(t, r, ctx, pid, "F01"); got.Link.State != domain.LinkUnreachable || got.Link.Why == "" {
		t.Fatalf("a leftover running check must read as unreachable with a reason, got %+v", got.Link)
	}
	if got := readFact(t, r, ctx, pid, "F02"); got.Link.State != domain.LinkOK {
		t.Fatalf("a finished check must be left alone, got %+v", got.Link)
	}
}

func TestPasteLogInPostgres(t *testing.T) {
	const pid = "research-paste-test"
	r, ctx := researchRepo(t, pid)
	first, err := r.AddPaste(ctx, pid, application.PasteLog{
		Added: []string{"F01", "F02"}, Rejected: []domain.PasteRejected{{Row: 3, Reason: "thiếu type"}},
	})
	if err != nil || first.N != 1 || first.At.IsZero() {
		t.Fatalf("first paste = %+v, %v, want number 1 with a time", first, err)
	}
	second, err := r.AddPaste(ctx, pid, application.PasteLog{Updated: []string{"F02"}})
	if err != nil || second.N != 2 {
		t.Fatalf("second paste = %+v, %v, want number 2", second, err)
	}
	logs, err := r.ListPastes(ctx, pid)
	if err != nil || len(logs) != 2 {
		t.Fatalf("logs = %+v, %v, want 2", logs, err)
	}
	if len(logs[0].Added) != 2 || len(logs[0].Rejected) != 1 || logs[0].Rejected[0].Reason != "thiếu type" || logs[0].Updated == nil {
		t.Fatalf("first log read back wrong: %+v", logs[0])
	}
	if logs[1].N != 2 || len(logs[1].Updated) != 1 || logs[1].Rejected == nil || logs[1].Added == nil {
		t.Fatalf("second log read back wrong: %+v", logs[1])
	}
	if err := r.DeleteAuthoring(ctx, pid); err != nil {
		t.Fatal(err)
	}
	if logs, err := r.ListPastes(ctx, pid); err != nil || len(logs) != 0 {
		t.Fatalf("deleting the project must delete its paste log, got %+v, %v", logs, err)
	}
}
