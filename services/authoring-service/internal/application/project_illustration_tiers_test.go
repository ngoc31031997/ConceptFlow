package application

import (
	"context"
	"strings"
	"testing"

	"authoring/internal/domain"
)

const tieredPlanJSON = `{"reuse": [
   {"name": "Airplane", "tier": "hero", "reason": "phim nói về chiếc máy bay"},
   {"name": "Cloud", "tier": "hero", "reason": "mây là chủ đề"},
   {"name": "Apple", "tier": "background", "reason": "quả táo chỉ để trang trí"},
   "Tooth"
 ],
 "draw": [
   {"name": "Bee", "description": "con ong mật nhìn ngang", "folder_id": "dong-vat", "shots": ["1.1"], "tier": "hero", "reason": "ong thụ phấn là chủ đề"},
   {"name": "Hive", "description": "tổ ong", "folder_id": "do-vat", "tier": "weird"}
 ]}`

func rowNamed(t *testing.T, rows []domain.ProjectIllustration, name string) domain.ProjectIllustration {
	t.Helper()
	for _, r := range rows {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no row named %s in %+v", name, rows)
	return domain.ProjectIllustration{}
}

func TestPlanReadsTheTierAndReasonOfEveryRow(t *testing.T) {
	llm := &scriptedLLM{replies: []string{tieredPlanJSON}}
	uc, _, _ := stage(llm)
	rows, err := uc.Plan(context.Background(), "p1", "")
	if err != nil {
		t.Fatal(err)
	}
	bee := rowNamed(t, rows, "Bee")
	if bee.Tier != domain.TierHero || bee.TierReason != "ong thụ phấn là chủ đề" || bee.State != domain.PIPlanned {
		t.Fatalf("Bee: %+v", bee)
	}
	if hive := rowNamed(t, rows, "Hive"); hive.Tier != domain.TierSupport {
		t.Fatalf("an unknown tier must become the supporting one, got %q", hive.Tier)
	}
	if apple := rowNamed(t, rows, "Apple"); apple.Tier != domain.TierBackground || apple.State != domain.PIReused {
		t.Fatalf("Apple: %+v", apple)
	}
	// A bare name is a supporting drawing.
	if tooth := rowNamed(t, rows, "Tooth"); tooth.Tier != domain.TierSupport || tooth.State != domain.PIReused {
		t.Fatalf("Tooth: %+v", tooth)
	}
}

func TestAKitDrawingWhoseDetailIsNotEnoughForTheTierIsDrawnAgainUnderANewName(t *testing.T) {
	llm := &scriptedLLM{replies: []string{tieredPlanJSON}}
	uc, _, _ := stage(llm)
	rows, err := uc.Plan(context.Background(), "p1", "")
	if err != nil {
		t.Fatal(err)
	}
	// Airplane is a rich drawing, enough for a hero: reused as it is.
	if plane := rowNamed(t, rows, "Airplane"); plane.State != domain.PIReused || plane.IllustrationID != "builtin-Airplane" {
		t.Fatalf("Airplane: %+v", plane)
	}
	// Cloud is a silhouette: not enough for a hero, so a detailed Cloud is planned.
	cloud := rowNamed(t, rows, "CloudHero")
	if cloud.State != domain.PIPlanned || cloud.IllustrationID != "" || cloud.Tier != domain.TierHero {
		t.Fatalf("CloudHero: %+v", cloud)
	}
	if !strings.Contains(cloud.Description, "vẽ kỹ hơn") {
		t.Fatalf("the new drawing is not described as a more detailed one: %q", cloud.Description)
	}
}

func TestThePlannerOnlySeesDrawingsOfTheProjectsKitAndTheirDetail(t *testing.T) {
	llm := &scriptedLLM{replies: []string{`{"reuse": [], "draw": []}`}}
	uc, _, lib := stage(llm)
	lib.rows["old"] = domain.Illustration{ID: "old", Name: "OldKitThing", FolderID: "do-vat", KitKey: "oldkit", DetailLevel: domain.DetailRich,
		Code: "export function OldKitThing() {}", Status: domain.IllustrationApproved}
	lib.rows["mine"] = domain.Illustration{ID: "mine", Name: "NewKitThing", FolderID: "do-vat", KitKey: domain.PhatsangKitKey, DetailLevel: domain.DetailSilhouette,
		Code: "export function NewKitThing() {}", Status: domain.IllustrationApproved}
	if _, err := uc.Plan(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	system := llm.calls[0].System
	if strings.Contains(system, "OldKitThing") {
		t.Fatal("a drawing of another kit was offered to the planner")
	}
	if !strings.Contains(system, "NewKitThing") || !strings.Contains(system, "chi tiết: silhouette") {
		t.Fatalf("the planner did not see the kit's drawing with its detail level:\n%s", system)
	}
}

func TestChangingTheTierOfADrawnRowSendsItBackToDrawingOnlyWhenTheDetailIsNotEnough(t *testing.T) {
	ctx := context.Background()
	uc, rows, lib := stage(&scriptedLLM{})
	lib.rows["ill-bee"] = domain.Illustration{ID: "ill-bee", Name: "Bee", FolderID: "dong-vat", KitKey: domain.PhatsangKitKey,
		DetailLevel: domain.DetailMedium, Code: "export function Bee() {}", Status: domain.IllustrationApproved}
	rows.rows["p1"] = []domain.ProjectIllustration{
		{ID: "r1", ProjectID: "p1", Name: "Bee", FolderID: "dong-vat", Kind: domain.IllustrationFigure, Tier: domain.TierSupport,
			State: domain.PIDrawn, IllustrationID: "ill-bee"},
	}
	// Downward: a medium drawing is enough for a background.
	got, err := uc.SetTier(ctx, "p1", "r1", domain.TierBackground)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tier != domain.TierBackground || got.State != domain.PIDrawn || got.IllustrationID != "ill-bee" || got.Name != "Bee" {
		t.Fatalf("lowering the tier changed the drawing: %+v", got)
	}
	// Upward: not enough detail for a hero, so the row is drawn again under its own name.
	got, err = uc.SetTier(ctx, "p1", "r1", domain.TierHero)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tier != domain.TierHero || got.State != domain.PIPlanned || got.IllustrationID != "" || got.Name != "BeeHero" {
		t.Fatalf("raising the tier: %+v", got)
	}
	// The drawing stays in the library for the videos that still use it.
	if _, ok := lib.rows["ill-bee"]; !ok {
		t.Fatal("the library drawing was removed")
	}
}

func TestSetTierRefusesWhatCannotHaveOne(t *testing.T) {
	ctx := context.Background()
	uc, rows, _ := stage(&scriptedLLM{})
	rows.rows["p1"] = []domain.ProjectIllustration{
		{ID: "r1", ProjectID: "p1", Name: "Meadow", Kind: domain.IllustrationBackdrop, State: domain.PIReused},
		{ID: "r2", ProjectID: "p1", Name: "Bee", Kind: domain.IllustrationFigure, Tier: domain.TierSupport, State: domain.PIDrawing},
	}
	if _, err := uc.SetTier(ctx, "p1", "r1", domain.TierHero); err == nil || !strings.Contains(err.Error(), "nền") {
		t.Fatalf("a backdrop has no tier: %v", err)
	}
	if _, err := uc.SetTier(ctx, "p1", "r2", domain.TierHero); err == nil || !strings.Contains(err.Error(), "đang được vẽ") {
		t.Fatalf("a drawing in flight cannot change tier: %v", err)
	}
	if _, err := uc.SetTier(ctx, "p1", "r2", "star"); err == nil || !strings.Contains(err.Error(), "không hợp lệ") {
		t.Fatalf("an unknown tier: %v", err)
	}
}

func TestTheNameOfADrawingForATierReplacesTheTierItAlreadyHas(t *testing.T) {
	for _, c := range []struct {
		name string
		tier domain.Tier
		want string
	}{
		{"Bee", domain.TierHero, "BeeHero"},
		{"BeeHero", domain.TierSupport, "BeeSupport"},
		{"BeeSupport", domain.TierBackground, "BeeBackground"},
		{"Hero", domain.TierHero, "HeroHero"}, // a name that is only a tier word keeps it
	} {
		if got := nameForTier(c.name, c.tier); got != c.want {
			t.Errorf("nameForTier(%q, %q) = %q, want %q", c.name, c.tier, got, c.want)
		}
	}
}

func TestOnlyDrawingsOfTheProjectsKitAreHandedToTheCode(t *testing.T) {
	ctx := context.Background()
	uc, _, lib := stage(&scriptedLLM{})
	lib.rows["a"] = domain.Illustration{ID: "a", Name: "Mine", FolderID: "do-vat", KitKey: domain.PhatsangKitKey,
		Usage: "<Mine />", Code: "export function Mine() {}", Status: domain.IllustrationApproved}
	lib.rows["b"] = domain.Illustration{ID: "b", Name: "Theirs", FolderID: "do-vat", KitKey: "oldkit",
		Usage: "<Theirs />", Code: "export function Theirs() {}", Status: domain.IllustrationApproved}
	drawings, err := uc.ForCode(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, d := range drawings {
		names = append(names, d.Name)
	}
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "Mine") || strings.Contains(joined, "Theirs") {
		t.Fatalf("drawings handed to the code: %s", joined)
	}
}
