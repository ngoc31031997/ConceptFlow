package application

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"authoring/internal/domain"
)

type fakeProjectRows struct {
	lib  *fakeIllustrationRepo
	rows map[string][]domain.ProjectIllustration
	n    int
}

func (f *fakeProjectRows) ListProjectIllustrations(_ context.Context, pid string) ([]domain.ProjectIllustration, error) {
	out := make([]domain.ProjectIllustration, 0, len(f.rows[pid]))
	for _, r := range f.rows[pid] {
		if r.IllustrationID != "" {
			if ill, ok := f.lib.rows[r.IllustrationID]; ok {
				r.Illustration = &ill
			}
		}
		out = append(out, r)
	}
	return out, nil
}
func (f *fakeProjectRows) ReplaceProjectIllustrations(_ context.Context, pid string, rows []domain.ProjectIllustration) error {
	f.rows[pid] = nil
	for _, r := range rows {
		f.n++
		r.ID = fmt.Sprintf("r%d", f.n)
		f.rows[pid] = append(f.rows[pid], r)
	}
	return nil
}
func (f *fakeProjectRows) UpdateProjectIllustration(_ context.Context, r domain.ProjectIllustration) error {
	for i, x := range f.rows[r.ProjectID] {
		if x.ID == r.ID {
			r.Illustration = nil
			f.rows[r.ProjectID][i] = r
		}
	}
	return nil
}

type storyboardOf string

func (s storyboardOf) GetAuthoringStoryboard(context.Context, string) (string, error) { return string(s), nil }

const planJSON = `Đây là danh sách:
{"reuse": ["Tooth", "Germ", "KhongCo"],
 "draw": [
   {"name": "Motorbike", "description": "xe máy đỏ nhìn ngang, có mặt", "folder_id": "phuong-tien", "shots": ["1.1", "2.3"]},
   {"name": "Candy", "description": "kẹo", "folder_id": "do-an-thuc-uong"},
   {"name": "Dentist", "description": "nha sĩ áo blouse", "folder_id": "khong-ton-tai"}
 ]}`

func stage(llm *scriptedLLM) (*ProjectIllustrationsUseCase, *fakeProjectRows, *fakeIllustrationRepo) {
	lib := newFakeIllustrationRepo()
	library := NewIllustrationsUseCase(lib, &fakeRenderer{}).WithDrawer(llm, nil, 1000)
	rows := &fakeProjectRows{lib: lib, rows: map[string][]domain.ProjectIllustration{}}
	return NewProjectIllustrationsUseCase(rows, library, storyboardOf(`{"scenes":[]}`), llm, nil, 1000), rows, lib
}

func TestPlanReusesWhatTheLibraryHasAndPlansTheRest(t *testing.T) {
	llm := &scriptedLLM{replies: []string{planJSON}}
	uc, _, _ := stage(llm)
	rows, err := uc.Plan(context.Background(), "p1", "")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]domain.ProjectIllustrationState{}
	for _, r := range rows {
		got[r.Name] = r.State
	}
	want := map[string]domain.ProjectIllustrationState{
		"Motorbike": domain.PIPlanned, "Candy": domain.PIReused, "Dentist": domain.PIPlanned,
		"Tooth": domain.PIReused, "Germ": domain.PIReused,
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("plan = %v, want %v (an unknown reuse name is dropped, a draw of an existing name becomes reuse)", got, want)
	}
	for _, r := range rows {
		if r.Name == "Dentist" && r.FolderID != "do-vat" {
			t.Errorf("unknown folder not replaced: %q", r.FolderID)
		}
	}
	if sys := llm.calls[0].System; !strings.Contains(sys, "- Tooth — Chiếc răng — co-the-suc-khoe") || !strings.Contains(sys, "- phuong-tien — Phương tiện") {
		t.Error("planner prompt lacks the library catalog or the folders")
	}
}

func TestEnsurePlansDrawsAndHoldsTheCodeStepUntilReviewed(t *testing.T) {
	llm := &scriptedLLM{replies: []string{planJSON, reply("Motorbike", motoCode)}}
	uc, rows, lib := stage(llm)
	ctx := context.Background()
	pending, err := uc.Ensure(ctx, "p1", "")
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, p := range pending {
		names = append(names, p.Name+":"+string(p.State))
	}
	// Motorbike drawn but still a draft; Dentist's drawing came back renamed to
	// Motorbike (the scripted model keeps answering that), so it failed.
	if strings.Join(names, ",") != "Motorbike:drawn,Dentist:failed" {
		t.Fatalf("pending = %v", names)
	}
	var moto string
	for _, r := range rows.rows["p1"] {
		if r.Name == "Motorbike" {
			moto = r.IllustrationID
		}
		if r.Name == "Dentist" {
			uc.SetSkipped(ctx, "p1", r.ID, true)
		}
	}
	calls := len(llm.calls)
	if pending, _ = uc.Ensure(ctx, "p1", ""); len(pending) != 1 || len(llm.calls) != calls {
		t.Fatalf("second Ensure must not re-plan or redraw: %d pending, %d new calls", len(pending), len(llm.calls)-calls)
	}
	ill := lib.rows[moto]
	ill.Status = domain.IllustrationApproved
	lib.rows[moto] = ill
	if pending, _ = uc.Ensure(ctx, "p1", ""); len(pending) != 0 {
		t.Fatalf("approved + skipped must open the gate, still pending: %+v", pending)
	}
	drawings, _ := uc.ForCode(ctx, "p1")
	if len(drawings) == 0 || drawings[0].Name != "Motorbike" {
		t.Fatalf("the video's own drawing must come first: %+v", drawings)
	}
	for _, d := range drawings {
		if d.Name == "Tooth" || d.Code == "" {
			t.Errorf("kit built-ins have no code to hand over: %+v", d)
		}
	}
}

func TestForCodeNeverHandsOverADraft(t *testing.T) {
	uc, _, lib := stage(&scriptedLLM{replies: []string{"{}"}})
	lib.rows["d"] = domain.Illustration{ID: "d", Name: "Draft", Code: "export function Draft() {}", Status: domain.IllustrationDraft}
	drawings, _ := uc.ForCode(context.Background(), "p1")
	for _, d := range drawings {
		if d.Name == "Draft" {
			t.Fatal("a draft reached the code step")
		}
	}
}
