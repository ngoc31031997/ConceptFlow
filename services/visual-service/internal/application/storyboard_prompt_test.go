package application

import (
	"strings"
	"testing"

	"visual/internal/domain"
)

func TestConceptIndexTextLeavesOutHowEachIsDrawn(t *testing.T) {
	rr := domain.RepresentationRules{
		Objects:            []domain.RepresentedThing{{ConceptID: "seed", Term: "hạt giống", Draw: "chiếc chìa khoá steel_gear"}},
		ScientificConcepts: []domain.ScientificConcept{{ConceptID: "entropy", Term: "entropy", Representation: "đồng hồ đo"}},
	}
	got := conceptIndexText(rr)
	want := "seed · vật thể · hạt giống\nentropy · khái niệm · entropy"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPlanIndexTextKeepsTheMethodNameOnly(t *testing.T) {
	var in domain.StoryboardInput
	in.Seq.Plan.Units = []domain.DocumentUnit{{
		UnitID: "VP-001", BeatID: "B1",
		EssentialVisualInfo: []domain.EssentialInfo{{Item: "bàn tay"}, {Item: "con xúc xắc"}},
		VisualApproach:      domain.VisualApproach{Method: "diagram", Description: "mô tả dài về cách dựng hình"},
	}}
	got := planIndexText(in)
	want := "VP-001 · B1 · thông tin hình bắt buộc: bàn tay; con xúc xắc · kiểu thể hiện: diagram"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if strings.Contains(got, "mô tả dài") {
		t.Fatal("the method description must not reach the asset list prompt")
	}
}
