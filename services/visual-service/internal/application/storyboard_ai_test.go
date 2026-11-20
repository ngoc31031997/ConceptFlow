package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"visual/internal/domain"
)

func assetJSON(sameAs, concept, name, kind, tier string, units ...string) string {
	us := `[]`
	if len(units) > 0 {
		us = `["` + strings.Join(units, `","`) + `"]`
	}
	return fmt.Sprintf(`{"same_as":%q,"concept_id":%q,"name":%q,"kind":%q,"tier":%q,"base_illustration_id":"","units":%s}`,
		sameAs, concept, name, kind, tier, us)
}

func assetsReply(a ...string) string { return `{"assets":[` + strings.Join(a, ",") + `]}` }

func detailReply(portrayal string, tokens ...string) string {
	toks := `[]`
	if len(tokens) > 0 {
		toks = `["` + strings.Join(tokens, `","`) + `"]`
	}
	return fmt.Sprintf(`{"detail":{"portrayal":%q,"must_not_imply":["siêu nhân"],"palette_tokens":%s,"expressions":["ngước nhìn"]}}`, portrayal, toks)
}

func TestStoryboardAI_TheAssetListIsCreatedFromAReplyWithoutDetailAndKeepsWhatTheCreatorWrote(t *testing.T) {
	f := newSbFixture(t)
	f.rendering.library = []LibraryDrawing{{ID: "lib1", Kind: "figure", Name: "Nguoi", Tags: []string{"người"}, DetailLevel: "rich"}}
	f.queue(domain.PromptRoleStoryboardAssets, assetsReply(
		assetJSON("", "astro", "Astro", "figure", "hero", "VP-001", "VP-002"),
		assetJSON("", "", "Space", "backdrop", "background", "VP-001")))
	st := f.must(f.desk.GenerateAssets(f.ctx, project, "ít thôi"))
	if len(st.Assets.Items) != 2 || st.Assets.Items[0].AssetID != "AST-ASTRO-001" || st.Assets.Approved {
		t.Fatalf("assets %+v", st.Assets.Items)
	}
	first := st.Assets.Items[0]
	if first.HasDetail || first.Portrayal != "" || len(first.Units) != 2 || first.Units[1] != "VP-002" || st.Assets.WithoutDetail != 2 || st.Assets.WithDetail != 0 {
		t.Errorf("a listed asset has its frame and no detail: %+v counts %d/%d", first, st.Assets.WithDetail, st.Assets.WithoutDetail)
	}
	prompt := f.llm.calls[0].System
	for _, want := range []string{"Phi hành gia", "lib1", "ít thôi", "VP-001", "(chưa có)"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the prompt lacks %q", want)
		}
	}
	for _, not := range []string{"navy", "siêu nhân"} {
		if strings.Contains(prompt, not) {
			t.Errorf("the list writer gets slim inputs, but the prompt has %q", not)
		}
	}
	if f.llm.calls[0].Step != StoryboardStep || f.llm.calls[0].Role != domain.PromptRoleStoryboardAssets || f.llm.calls[0].Model != "ollama" {
		t.Errorf("call %+v", f.llm.calls[0])
	}
	// The Creator rewrites the units; a second AI pass keeps them and an asset
	// whose fields do not change is not sent to rendering again.
	f.must(f.desk.EditAsset(f.ctx, project, "AST-ASTRO-001", jfields(t, map[string]any{"units": []string{"VP-003"}})))
	f.rendering.updated = nil
	f.queue(domain.PromptRoleStoryboardAssets, assetsReply(
		assetJSON("AST-ASTRO-001", "astro", "Astro", "figure", "hero", "VP-001"),
		assetJSON("AST-SPACE-001", "", "Space", "backdrop", "background", "VP-001")))
	st = f.must(f.desk.GenerateAssets(f.ctx, project, ""))
	if got := st.Assets.Items[0]; len(got.Units) != 1 || got.Units[0] != "VP-003" {
		t.Errorf("a field the Creator wrote must stay: %v", got.Units)
	}
	if len(f.rendering.updated) != 0 {
		t.Errorf("nothing changed, so rendering must not be asked: %v", f.rendering.updated)
	}
	if !strings.Contains(f.llm.calls[1].System, "AST-ASTRO-001") || !strings.Contains(f.llm.calls[1].System, "AST-ASTRO-001.units") {
		t.Error("the prompt must show the current assets and the Creator's fields")
	}
}

func TestStoryboardAI_AnAssetReplyThatDoesNotFitChangesNothing(t *testing.T) {
	cases := map[string]string{
		"concept outside the rules":     assetsReply(assetJSON("", "ghost", "Astro", "figure", "hero")),
		"unit outside the plan":         assetsReply(assetJSON("", "astro", "Astro", "figure", "hero", "VP-099")),
		"name not PascalCase":           assetsReply(assetJSON("", "astro", "astro x", "figure", "hero")),
		"kind outside the vocabulary":   assetsReply(assetJSON("", "astro", "Astro", "monster", "hero")),
		"same_as of an unknown asset":   assetsReply(assetJSON("AST-GHOST-001", "astro", "Astro", "figure", "hero")),
		"base illustration not in lib":  strings.Replace(assetsReply(assetJSON("", "astro", "Astro", "figure", "hero")), `"base_illustration_id":""`, `"base_illustration_id":"lib9"`, 1),
		"a detail key":                  strings.Replace(assetsReply(assetJSON("", "astro", "Astro", "figure", "hero")), `"units"`, `"portrayal":"p","units"`, 1),
		"not JSON the writer may write": `{"assets":[],"x":1}`,
	}
	for name, reply := range cases {
		t.Run(name, func(t *testing.T) {
			f := newSbFixture(t)
			f.queue(domain.PromptRoleStoryboardAssets, reply)
			if _, err := f.desk.GenerateAssets(f.ctx, project, ""); !errors.Is(err, domain.ErrUnreadableReply) {
				t.Fatalf("want ErrUnreadableReply, got %v", err)
			}
			if len(f.rendering.assets) != 0 {
				t.Error("nothing may be created")
			}
		})
	}
}

func TestStoryboardAI_AnAssetShotsUseCannotBeLeftOutOfTheList(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	f.writeUnit1()
	f.queue(domain.PromptRoleStoryboardAssets, assetsReply(assetJSON("AST-SPACE-001", "", "Space", "backdrop", "background")))
	if _, err := f.desk.GenerateAssets(f.ctx, project, ""); !errors.Is(err, domain.ErrUnreadableReply) {
		t.Fatalf("want ErrUnreadableReply, got %v", err)
	}
	if len(f.rendering.deleted) != 0 || len(f.rendering.assets) != 2 {
		t.Error("nothing may change")
	}
}

func TestStoryboardAI_ALeftOutAssetNothingUsesIsDeleted(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	f.queue(domain.PromptRoleStoryboardAssets, assetsReply(assetJSON("AST-ASTRO-001", "astro", "Astro", "figure", "hero")))
	st := f.must(f.desk.GenerateAssets(f.ctx, project, ""))
	if len(st.Assets.Items) != 1 || len(f.rendering.deleted) != 1 || f.rendering.deleted[0] != "AST-SPACE-001" {
		t.Errorf("assets %d deleted %v", len(st.Assets.Items), f.rendering.deleted)
	}
}

func TestStoryboardAI_ShotsAreWrittenOnlyOverApprovedAssets(t *testing.T) {
	f := newSbFixture(t)
	f.queue(domain.PromptRoleStoryboardUnit, shotsReply(shotJSON([]string{"b1.1"}, nil, "mở")))
	if _, err := f.desk.GenerateUnit(f.ctx, project, "VP-001", ""); !errors.Is(err, domain.ErrAssetsMissing) {
		t.Fatalf("no asset: want ErrAssetsMissing, got %v", err)
	}
	f.must(f.desk.AddAsset(f.ctx, project, jfields(t, map[string]any{"concept_id": "astro", "name": "Astro", "portrayal": "p"})))
	_, err := f.desk.GenerateUnit(f.ctx, project, "VP-001", "")
	if !errors.Is(err, domain.ErrAssetsMissing) || !strings.Contains(err.Error(), "AST-ASTRO-001") {
		t.Fatalf("a planned asset: want ErrAssetsMissing naming it, got %v", err)
	}
	if len(f.llm.calls) != 0 {
		t.Error("no model turn may run")
	}
	if _, err := f.desk.Generate(f.ctx, project, ""); !errors.Is(err, domain.ErrAssetsMissing) {
		t.Errorf("generate all: want ErrAssetsMissing, got %v", err)
	}
}

func TestStoryboardAI_AUnitIsWrittenWithItsFramesAndNumbersTheNextUnitContinues(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	f.queue(domain.PromptRoleStoryboardUnit,
		shotsReply(shotJSON([]string{"b1.1"}, []string{"Góc nhìn A"}, "mở")),
		shotsReply(shotJSON([]string{"b1.2"}, nil, "một"), shotJSON([]string{"b1.3"}, nil, "hai")))
	st := f.must(f.desk.GenerateUnit(f.ctx, project, "VP-001", ""))
	sh := unitOf(st, "VP-001").Shots[0]
	if sh.UnitID != "VP-001" || sh.BeatID != "b1" || !sh.Frame.Drawn || sh.Frame.Path != "storyboard/SH-001.png" {
		t.Fatalf("shot %+v", sh)
	}
	req := f.rendering.frames[0]
	if req.UnitID != "VP-001" || req.BeatID != "b1" || req.SpecHash == "" || req.ShotID != "SH-001" || req.Background.AssetID != "AST-SPACE-001" ||
		len(req.Placements) != 1 || req.Placements[0].Position != "lower_left" || req.OnScreenText[0].MinPx != 28 || req.Background.Color != "#0F1B3D" || req.TextColor != "#FFFFFF" {
		t.Errorf("frame request %+v", req)
	}
	if string(req.Kit) == "" || !strings.Contains(string(req.Kit), `"ink":"#101010"`) {
		t.Errorf("the kit of the style the video picked must be sent: %s", req.Kit)
	}
	if !strings.Contains(f.llm.calls[0].System, "(không có: unit đang viết là unit đầu video)") {
		t.Error("the first unit has no unit before it")
	}
	st = f.must(f.desk.GenerateUnit(f.ctx, project, "VP-002", ""))
	if got := unitOf(st, "VP-002").Shots; len(got) != 2 || got[0].ShotID != "SH-002" || got[1].ShotID != "SH-003" {
		t.Fatalf("shots %+v", got)
	}
	if !strings.Contains(f.llm.calls[1].System, `"shot_id":"SH-001"`) {
		t.Error("a unit reads the shots of the unit before it")
	}
	if !strings.Contains(f.llm.calls[1].System, "AST-ASTRO-001") || !strings.Contains(f.llm.calls[1].System, "lower_left") {
		t.Error("the prompt lists the approved assets and the allowed positions")
	}
}

func TestStoryboardAI_AShotReplyThatDoesNotFitStoresNothing(t *testing.T) {
	good := shotJSON([]string{"b1.1"}, nil, "mở")
	cases := map[string]string{
		"asset not approved":          strings.Replace(good, `"asset_refs":["AST-ASTRO-001","AST-SPACE-001"]`, `"asset_refs":["AST-GHOST-001"]`, 1),
		"line outside the unit":       shotsReply(shotJSON([]string{"b2.1"}, nil, "x")),
		"line of the unit uncovered":  shotsReply(strings.Replace(good, `["b1.1"]`, `[]`, 1)),
		"position outside":            strings.Replace(shotsReply(good), "lower_left", "somewhere", 1),
		"text longer than the rules":  shotsReply(shotJSON([]string{"b1.1"}, []string{"một hai ba bốn năm"}, "x")),
		"no shot":                     `{"shots":[]}`,
		"an id the machine writes":    strings.Replace(shotsReply(good), `{"script_refs"`, `{"shot_id":"SH-005","script_refs"`, 1),
		"split_screen with one thing": strings.Replace(shotsReply(good), `"subject_position":"lower_left"`, `"subject_position":"split_screen"`, 1),
	}
	if _, ok := cases["asset not approved"]; ok {
		cases["asset not approved"] = shotsReply(cases["asset not approved"])
	}
	for name, reply := range cases {
		t.Run(name, func(t *testing.T) {
			f := newSbFixture(t)
			f.approvedAssets()
			f.queue(domain.PromptRoleStoryboardUnit, reply)
			if _, err := f.desk.GenerateUnit(f.ctx, project, "VP-001", ""); !errors.Is(err, domain.ErrUnreadableReply) {
				t.Fatalf("want ErrUnreadableReply, got %v", err)
			}
			if len(f.store.units) != 0 || len(f.rendering.frames) != 0 {
				t.Error("nothing may be stored or drawn")
			}
		})
	}
}

func TestStoryboardAI_RewritingAUnitKeepsTheShotNumberAndTheCreatorsFields(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	f.writeUnit1()
	f.must(f.desk.EditShot(f.ctx, project, "VP-001", "SH-001", jfields(t, map[string]any{"shot_purpose": "của Creator"})))
	f.queue(domain.PromptRoleStoryboardUnit, shotsReply(shotJSON([]string{"b1.1"}, nil, "của AI")))
	st := f.must(f.desk.GenerateUnit(f.ctx, project, "VP-001", "viết lại"))
	sh := unitOf(st, "VP-001").Shots[0]
	if sh.ShotID != "SH-001" || sh.ShotPurpose != "của Creator" || sh.Status != domain.UnitPending {
		t.Errorf("shot %+v", sh)
	}
	if !strings.Contains(f.llm.calls[1].System, "SH-001.shot_purpose") {
		t.Error("the prompt must name the Creator's fields")
	}
	// An AI that drops a shot the Creator edited is refused.
	f.must(f.desk.AddShot(f.ctx, project, "VP-001", jfields(t, map[string]any{"script_refs": []string{"b1.1"}, "shot_purpose": "thêm của Creator"})))
	f.queue(domain.PromptRoleStoryboardUnit, shotsReply(shotJSON([]string{"b1.1"}, nil, "một")))
	if _, err := f.desk.GenerateUnit(f.ctx, project, "VP-001", ""); !errors.Is(err, domain.ErrUnreadableReply) {
		t.Errorf("dropping an edited shot: want ErrUnreadableReply, got %v", err)
	}
}

func TestStoryboardAI_GenerateWritesEachEmptyUnitInItsOwnTurnInSequenceOrder(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	f.writeUnit1()
	f.queue(domain.PromptRoleStoryboardUnit,
		shotsReply(shotJSON([]string{"b1.2", "b1.3"}, nil, "giữa")),
		shotsReply(shotJSON([]string{"b2.1"}, nil, "đóng")))
	st := f.must(f.desk.Generate(f.ctx, project, ""))
	if len(st.EmptyUnits) != 0 || st.ShotsTotal != 3 {
		t.Fatalf("empty %v total %d", st.EmptyUnits, st.ShotsTotal)
	}
	if len(f.llm.calls) != 3 || !strings.Contains(f.llm.calls[2].System, `"shot_purpose":"giữa"`) {
		t.Errorf("the last unit must read the one before it (%d turns)", len(f.llm.calls))
	}
	// A failing unit stops the run and keeps the ones before it.
	f2 := newSbFixture(t)
	f2.approvedAssets()
	f2.queue(domain.PromptRoleStoryboardUnit, shotsReply(shotJSON([]string{"b1.1"}, nil, "mở")), `{"shots":[]}`)
	if _, err := f2.desk.Generate(f2.ctx, project, ""); !errors.Is(err, domain.ErrUnreadableReply) || !strings.Contains(err.Error(), "VP-002") {
		t.Fatalf("want ErrUnreadableReply naming VP-002, got %v", err)
	}
	if st := f2.must(f2.desk.State(f2.ctx, project)); st.ShotsTotal != 1 {
		t.Errorf("VP-001 stays: %d shots", st.ShotsTotal)
	}
}

func TestStoryboardAI_AIOutsideTheSystemBuildsThePromptThenReadsThePastedReply(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	promptRun := NewExternalAIRun(ExternalAIPrompt, "")
	_, err := f.desk.GenerateUnit(WithExternalAI(f.ctx, promptRun), project, "VP-001", "ghi chú")
	if !errors.Is(err, ErrExternalAIPromptBuilt) {
		t.Fatalf("want ErrExternalAIPromptBuilt, got %v", err)
	}
	if p, ok := promptRun.Prompt(); !ok || !strings.Contains(p, "ghi chú") || !strings.Contains(p, "VP-001") {
		t.Errorf("prompt %q", p)
	}
	if len(f.llm.calls) != 0 || len(f.store.units) != 0 {
		t.Error("no model may be asked and nothing stored")
	}
	reply := shotsReply(shotJSON([]string{"b1.1"}, nil, "mở"))
	st := f.must(f.desk.GenerateUnit(WithExternalAI(f.ctx, NewExternalAIRun(ExternalAIPaste, reply)), project, "VP-001", ""))
	if len(unitOf(st, "VP-001").Shots) != 1 {
		t.Error("the pasted reply must be written")
	}
	// A pasted reply that does not fit is refused like the system's own.
	bad := NewExternalAIRun(ExternalAIPaste, `{"shots":[]}`)
	if _, err := f.desk.GenerateUnit(WithExternalAI(f.ctx, bad), project, "VP-002", ""); !errors.Is(err, domain.ErrUnreadableReply) {
		t.Errorf("want ErrUnreadableReply, got %v", err)
	}
	// The asset list and the review take the same modes.
	assetsRun := NewExternalAIRun(ExternalAIPrompt, "")
	if _, err := f.desk.GenerateAssets(WithExternalAI(f.ctx, assetsRun), project, ""); !errors.Is(err, ErrExternalAIPromptBuilt) {
		t.Errorf("assets: %v", err)
	}
	reviewRun := NewExternalAIRun(ExternalAIPrompt, "")
	if _, err := f.desk.Review(WithExternalAI(f.ctx, reviewRun), project, ""); !errors.Is(err, ErrExternalAIPromptBuilt) {
		t.Errorf("review: %v", err)
	}
}

func sbFinding(group, key, field, text, fix string) string {
	return fmt.Sprintf(`{"group":%q,"unit_key":%q,"field":%q,"text":%q,"detail":"d","fix":%q,"same_as":""}`, group, key, field, text, fix)
}

func TestStoryboardAI_ReviewFindingsNameAShotAnAssetOrAUnit(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	f.writeUnit1()
	if _, err := newSbFixture(t).desk.Review(f.ctx, project, ""); !errors.Is(err, domain.ErrUnitNotWritten) {
		t.Errorf("nothing to review: %v", err)
	}
	f.queue(domain.PromptRoleStoryboardReview, reviewReply(
		sbFinding("clarity", "SH-001", "shot_purpose", "mơ hồ", "Cho thấy phi hành gia"),
		sbFinding("consistency", "AST-ASTRO-001", "portrayal", "lệch", "Phi hành gia thật"),
		sbFinding("accuracy", "VP-001", "", "sai", "")))
	st := f.must(f.desk.Review(f.ctx, project, ""))
	if len(st.Review.Findings) != 3 || st.Review.OpenCount != 3 || !st.Review.Reviewed {
		t.Fatalf("review %+v", st.Review)
	}
	for _, want := range []string{"SH-001", "AST-ASTRO-001", "có ảnh khung", "[chưa đạt]"} {
		if !strings.Contains(f.llm.calls[len(f.llm.calls)-1].System, want) {
			t.Errorf("the review prompt lacks %q", want)
		}
	}
	// Applying writes the fix into the shot (a Creator field, back to review) and
	// into the asset at rendering.
	f.must(f.desk.ApproveShot(f.ctx, project, "VP-001", "SH-001"))
	st = f.must(f.desk.ApplyFinding(f.ctx, project, "f1"))
	sh := unitOf(st, "VP-001").Shots[0]
	if sh.ShotPurpose != "Cho thấy phi hành gia" || sh.Status != domain.UnitPending || sh.CreatorFields[0] != "shot_purpose" {
		t.Errorf("shot %+v", sh)
	}
	st = f.must(f.desk.ApplyFinding(f.ctx, project, "f2"))
	if got := st.Assets.Items[0].Portrayal; got != "Phi hành gia thật" {
		t.Errorf("portrayal %q", got)
	}
	if _, err := f.desk.ApplyFinding(f.ctx, project, "f3"); !errors.Is(err, domain.ErrFindingNotApplicable) {
		t.Errorf("a unit has no field to write: %v", err)
	}
	if _, err := f.desk.ApplyFinding(f.ctx, project, "f9"); !errors.Is(err, domain.ErrUnknownFinding) {
		t.Errorf("%v", err)
	}
	st = f.must(f.desk.AcceptFinding(f.ctx, project, "f3", "chấp nhận"))
	if st.Review.OpenCount != 0 {
		t.Errorf("open %d", st.Review.OpenCount)
	}
	if _, err := f.desk.AcceptFinding(f.ctx, project, domain.CheckShotFrame, "x"); !errors.Is(err, domain.ErrFindingNotApplicable) {
		t.Errorf("a blocking check cannot be accepted: %v", err)
	}
	f.queue(domain.PromptRoleStoryboardReview, reviewReply(sbFinding("clarity", "SH-777", "", "lạ", "")))
	if _, err := f.desk.Review(f.ctx, project, ""); !errors.Is(err, domain.ErrUnreadableReply) {
		t.Errorf("a finding about an unknown shot: want ErrUnreadableReply, got %v", err)
	}
}

// readyToLock writes, draws, approves and reviews all three units.
func (f *sbFixture) readyToLock() {
	f.t.Helper()
	f.approvedAssets()
	f.queue(domain.PromptRoleStoryboardUnit,
		shotsReply(shotJSON([]string{"b1.1"}, nil, "mở")),
		shotsReply(shotJSON([]string{"b1.2", "b1.3"}, nil, "giữa")),
		shotsReply(shotJSON([]string{"b2.1"}, nil, "đóng")))
	f.must(f.desk.Generate(f.ctx, project, ""))
	f.queue(domain.PromptRoleStoryboardReview, reviewReply())
	f.must(f.desk.Review(f.ctx, project, ""))
	f.must(f.desk.ApproveAll(f.ctx, project))
}

func TestStoryboardAI_LockNeedsEverythingApprovedDrawnAndReviewedThenSendsTheDocumentToRendering(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	if _, err := f.desk.Lock(f.ctx, project, ""); !errors.Is(err, domain.ErrLockNotReady) {
		t.Fatalf("an empty storyboard: want ErrLockNotReady, got %v", err)
	}
	f.queue(domain.PromptRoleStoryboardUnit,
		shotsReply(shotJSON([]string{"b1.1"}, nil, "mở")),
		shotsReply(shotJSON([]string{"b1.2", "b1.3"}, nil, "giữa")),
		shotsReply(shotJSON([]string{"b2.1"}, nil, "đóng")))
	f.must(f.desk.Generate(f.ctx, project, ""))
	_, err := f.desk.Lock(f.ctx, project, "")
	if !errors.Is(err, domain.ErrLockNotReady) || !strings.Contains(err.Error(), "chưa duyệt") || !strings.Contains(err.Error(), "rà soát") {
		t.Fatalf("shots unapproved and not reviewed: got %v", err)
	}
	f.queue(domain.PromptRoleStoryboardReview, reviewReply())
	f.must(f.desk.Review(f.ctx, project, ""))
	st := f.must(f.desk.ApproveAll(f.ctx, project))
	if st.ShotsApproved != 3 {
		t.Fatalf("approved %d", st.ShotsApproved)
	}
	st = f.must(f.desk.State(f.ctx, project))
	if !st.CanLock || st.NextVersion != 1 {
		t.Fatalf("blockers %v", st.Blockers)
	}
	st = f.must(f.desk.Lock(f.ctx, project, ""))
	if st.LockedVersion != 1 || st.NextVersion != 2 || st.LockedNote != "Chốt lần đầu" || len(f.rendering.locked) != 1 {
		t.Fatalf("locked %d next %d note %q", st.LockedVersion, st.NextVersion, st.LockedNote)
	}
	rec := f.rendering.locked[0]
	if rec.SequenceVersion != 1 {
		t.Errorf("sequence version %d", rec.SequenceVersion)
	}
	var doc domain.StoryboardDocument
	if err := json.Unmarshal(rec.Document, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.DocumentID != "storyboard-v1" || doc.Status != "locked" || len(doc.Shots) != 3 || doc.Shots[0].FramePath != "storyboard/v1/SH-001.png" ||
		doc.Shots[1].UnitID != "VP-002" || len(doc.Assets) != 2 || doc.SourceVersions.VisualSequence != 1 {
		t.Errorf("document %+v", doc)
	}
	if st.CanLock || len(st.Blockers) == 0 || !strings.Contains(strings.Join(st.Blockers, " "), "Không có thay đổi") {
		t.Errorf("nothing changed: %v", st.Blockers)
	}
	if _, err := f.desk.Lock(f.ctx, project, ""); !errors.Is(err, domain.ErrNothingToLock) {
		t.Fatalf("a second lock: want ErrNothingToLock, got %v", err)
	}
	v, err := f.desk.Version(f.ctx, project, 1)
	if err != nil || v.Version != 1 {
		t.Errorf("version %+v %v", v, err)
	}
	if _, err := f.desk.Version(f.ctx, project, 2); !errors.Is(err, domain.ErrNoVersion) {
		t.Errorf("want ErrNoVersion, got %v", err)
	}
	if latest, err := f.desk.LatestDocument(f.ctx, project); err != nil || len(latest) == 0 {
		t.Errorf("latest %s %v", latest, err)
	}
}

func TestStoryboardAI_ASecondLockTakesTheNextVersionAfterAChange(t *testing.T) {
	f := newSbFixture(t)
	f.readyToLock()
	for _, c := range f.must(f.desk.State(f.ctx, project)).Checks {
		if !c.OK && !domain.IsBlockingCheck(c.ID) {
			f.must(f.desk.AcceptFinding(f.ctx, project, c.ID, "đủ dùng"))
		}
	}
	f.must(f.desk.Lock(f.ctx, project, ""))
	f.must(f.desk.EditShot(f.ctx, project, "VP-003", "SH-003", jfields(t, map[string]any{"shot_purpose": "đóng khác"})))
	f.must(f.desk.ApproveShot(f.ctx, project, "VP-003", "SH-003"))
	st := f.must(f.desk.State(f.ctx, project))
	if !st.CanLock || len(st.Draft) != 1 || st.Draft[0] != "SH-003 · đổi" {
		t.Fatalf("blockers %v draft %v", st.Blockers, st.Draft)
	}
	st = f.must(f.desk.Lock(f.ctx, project, "đổi mục đích"))
	if st.LockedVersion != 2 || st.LockedNote != "đổi mục đích" || f.rendering.locked[1].Version != 2 {
		t.Errorf("locked %d", st.LockedVersion)
	}
}

func TestStoryboardAI_ASequenceWrittenFromOlderStepsCannotLock(t *testing.T) {
	f := newSbFixture(t)
	f.readyToLock()
	f.store.concepts = append(f.store.concepts, VersionRecord{Version: 2, ScriptVersion: 2, Document: sbConceptDoc(t)})
	_, err := f.desk.Lock(f.ctx, project, "")
	if !errors.Is(err, domain.ErrSequenceOutdated) {
		t.Fatalf("want ErrSequenceOutdated, got %v", err)
	}
	if len(f.rendering.locked) != 0 {
		t.Error("nothing may be locked")
	}
}

func TestStoryboardAI_ARefusalOfRenderingWhenLockingSurfaces(t *testing.T) {
	f := newSbFixture(t)
	f.readyToLock()
	for _, c := range f.must(f.desk.State(f.ctx, project)).Checks {
		if !c.OK && !domain.IsBlockingCheck(c.ID) {
			f.must(f.desk.AcceptFinding(f.ctx, project, c.ID, "đủ dùng"))
		}
	}
	f.rendering.lockErr = &RenderingError{Status: 409, Message: "thiếu khung"}
	var re *RenderingError
	if _, err := f.desk.Lock(f.ctx, project, ""); !errors.As(err, &re) || re.Status != 409 {
		t.Fatalf("want rendering's refusal, got %v", err)
	}
}

// listedAssets puts the two assets of a listed video at rendering with no detail.
func (f *sbFixture) listedAssets() {
	f.t.Helper()
	f.queue(domain.PromptRoleStoryboardAssets, assetsReply(
		assetJSON("", "astro", "Astro", "figure", "hero", "VP-001", "VP-002"),
		assetJSON("", "", "Space", "backdrop", "background", "VP-002")))
	f.must(f.desk.GenerateAssets(f.ctx, project, ""))
	f.llm.calls = nil
}

func TestStoryboardAI_AnAssetWithoutDetailCannotBeDrawn(t *testing.T) {
	f := newSbFixture(t)
	f.listedAssets()
	if _, err := f.desk.DrawAsset(f.ctx, project, "AST-ASTRO-001"); !errors.Is(err, domain.ErrAssetNoDetail) {
		t.Fatalf("want ErrAssetNoDetail, got %v", err)
	}
	if len(f.rendering.draws) != 0 {
		t.Error("rendering must not be asked to draw")
	}
	f.queue(domain.PromptRoleStoryboardAssetDetail, detailReply("nhỏ bé", "ink"))
	f.must(f.desk.GenerateAssetDetail(f.ctx, project, "AST-ASTRO-001", ""))
	st := f.must(f.desk.DrawAsset(f.ctx, project, "AST-ASTRO-001"))
	if got := st.Assets.Items[0]; got.Status != domain.AssetDrawn || !got.HasDetail {
		t.Errorf("an asset with detail is drawn: %+v", got)
	}
}

func TestStoryboardAI_TheDetailOfOneAssetIsWrittenFromSlimInputs(t *testing.T) {
	f := newSbFixture(t)
	f.listedAssets()
	f.queue(domain.PromptRoleStoryboardAssetDetail, detailReply("nhỏ bé giữa vũ trụ", "ink"))
	st := f.must(f.desk.GenerateAssetDetail(f.ctx, project, "AST-ASTRO-001", "dịu hơn"))
	got := st.Assets.Items[0]
	if !got.HasDetail || got.Portrayal != "nhỏ bé giữa vũ trụ" || len(got.MustNotImply) != 1 || got.PaletteTokens[0] != "ink" || got.Expressions[0] != "ngước nhìn" {
		t.Fatalf("detail %+v", got)
	}
	if st.Assets.WithDetail != 1 || st.Assets.WithoutDetail != 1 || st.Assets.Items[1].HasDetail {
		t.Errorf("counts %d/%d", st.Assets.WithDetail, st.Assets.WithoutDetail)
	}
	if got.Name != "Astro" || got.Tier != "hero" || len(got.Units) != 2 {
		t.Errorf("the frame fields must not change: %+v", got)
	}
	c := f.llm.calls[0]
	if c.Role != domain.PromptRoleStoryboardAssetDetail || c.Step != StoryboardStep {
		t.Errorf("call %+v", c)
	}
	for _, want := range []string{"AST-ASTRO-001", "Phi hành gia", "bộ đồ trắng", "siêu nhân", "navy", "VP-001", "VP-002", "dịu hơn", "Space · background"} {
		if !strings.Contains(c.System, want) {
			t.Errorf("the prompt lacks %q", want)
		}
	}
	// VP-003 is not a unit of the asset: its plan line stays out.
	if strings.Contains(c.System, "VP-003") {
		t.Error("only the units the asset appears in belong in the prompt")
	}
}

func TestStoryboardAI_ADetailReplyThatDoesNotFitChangesNothing(t *testing.T) {
	cases := map[string]string{
		"token outside the palette": detailReply("p", "pink"),
		"no portrayal":              detailReply(" "),
		"a frame key":               `{"detail":{"portrayal":"p","must_not_imply":[],"palette_tokens":[],"expressions":[],"tier":"hero"}}`,
		"no detail key":             `{"portrayal":"p"}`,
	}
	for name, reply := range cases {
		t.Run(name, func(t *testing.T) {
			f := newSbFixture(t)
			f.listedAssets()
			f.rendering.updated = nil
			f.queue(domain.PromptRoleStoryboardAssetDetail, reply)
			if _, err := f.desk.GenerateAssetDetail(f.ctx, project, "AST-ASTRO-001", ""); !errors.Is(err, domain.ErrUnreadableReply) {
				t.Fatalf("want ErrUnreadableReply, got %v", err)
			}
			if len(f.rendering.updated) != 0 || f.rendering.assets[0].Portrayal != "" {
				t.Error("nothing may be stored")
			}
		})
	}
	f := newSbFixture(t)
	f.listedAssets()
	if _, err := f.desk.GenerateAssetDetail(f.ctx, project, "AST-GHOST-001", ""); !errors.Is(err, domain.ErrUnknownAsset) {
		t.Errorf("an unknown asset: %v", err)
	}
}

func TestStoryboardAI_RewritingADetailKeepsTheFieldsTheCreatorWroteAndReopensADrawnAsset(t *testing.T) {
	f := newSbFixture(t)
	f.listedAssets()
	f.queue(domain.PromptRoleStoryboardAssetDetail, detailReply("của AI", "ink"))
	f.must(f.desk.GenerateAssetDetail(f.ctx, project, "AST-ASTRO-001", ""))
	f.must(f.desk.DrawAsset(f.ctx, project, "AST-ASTRO-001"))
	f.must(f.desk.EditAsset(f.ctx, project, "AST-ASTRO-001", jfields(t, map[string]any{"expressions": []string{"của Creator"}})))
	f.queue(domain.PromptRoleStoryboardAssetDetail, detailReply("AI viết lại", "ink"))
	st := f.must(f.desk.GenerateAssetDetail(f.ctx, project, "AST-ASTRO-001", "viết lại"))
	got := st.Assets.Items[0]
	if got.Portrayal != "AI viết lại" || len(got.Expressions) != 1 || got.Expressions[0] != "của Creator" {
		t.Errorf("the Creator's field stays, the AI's is rewritten: %+v", got)
	}
	if got.Status != domain.AssetPlanned {
		t.Errorf("a changed detail sends a drawn asset back to planned: %q", got.Status)
	}
	if !strings.Contains(f.llm.calls[len(f.llm.calls)-1].System, "AST-ASTRO-001.expressions") && !strings.Contains(f.llm.calls[len(f.llm.calls)-1].System, "expressions") {
		t.Error("the prompt names the Creator's fields")
	}
}

func TestStoryboardAI_EveryAssetWithoutDetailIsWrittenInOrderEachReadingTheOnesBefore(t *testing.T) {
	f := newSbFixture(t)
	f.listedAssets()
	f.queue(domain.PromptRoleStoryboardAssetDetail, detailReply("nhân vật", "ink"), detailReply("phông", "navy"))
	st := f.must(f.desk.GenerateAllAssetDetails(f.ctx, project, ""))
	if st.Assets.WithDetail != 2 || st.Assets.WithoutDetail != 0 {
		t.Fatalf("counts %d/%d", st.Assets.WithDetail, st.Assets.WithoutDetail)
	}
	if len(f.llm.calls) != 2 || !strings.Contains(f.llm.calls[0].System, "AST-ASTRO-001") || !strings.Contains(f.llm.calls[1].System, "AST-SPACE-001") {
		t.Fatalf("one turn per asset in list order: %d calls", len(f.llm.calls))
	}
	if !strings.Contains(f.llm.calls[1].System, "Astro · hero · nhân vật") {
		t.Error("the second asset reads the detail of the first")
	}
	// Nothing is left to write: no further turn runs.
	f.llm.calls = nil
	f.must(f.desk.GenerateAllAssetDetails(f.ctx, project, ""))
	if len(f.llm.calls) != 0 {
		t.Error("an asset with detail is not written again")
	}
}

func TestStoryboardAI_AFailedAssetStopsTheRunAndKeepsTheDetailsWritten(t *testing.T) {
	f := newSbFixture(t)
	f.listedAssets()
	f.queue(domain.PromptRoleStoryboardAssetDetail, detailReply("nhân vật", "ink"), detailReply("phông", "pink"))
	_, err := f.desk.GenerateAllAssetDetails(f.ctx, project, "")
	if !errors.Is(err, domain.ErrUnreadableReply) || !strings.Contains(err.Error(), "AST-SPACE-001") {
		t.Fatalf("want ErrUnreadableReply naming the asset, got %v", err)
	}
	st := f.must(f.desk.State(f.ctx, project))
	if !st.Assets.Items[0].HasDetail || st.Assets.Items[1].HasDetail {
		t.Error("the first detail stays, the failed one is not written")
	}
}

func TestStoryboardAI_AIOutsideTheSystemWritesOneAssetDetail(t *testing.T) {
	f := newSbFixture(t)
	f.listedAssets()
	promptRun := NewExternalAIRun(ExternalAIPrompt, "")
	_, err := f.desk.GenerateAssetDetail(WithExternalAI(f.ctx, promptRun), project, "AST-ASTRO-001", "ghi chú")
	if !errors.Is(err, ErrExternalAIPromptBuilt) {
		t.Fatalf("want ErrExternalAIPromptBuilt, got %v", err)
	}
	if p, ok := promptRun.Prompt(); !ok || !strings.Contains(p, "ghi chú") || !strings.Contains(p, "AST-ASTRO-001") {
		t.Errorf("prompt %q", p)
	}
	if len(f.llm.calls) != 0 || f.rendering.assets[0].Portrayal != "" {
		t.Error("no model may be asked and nothing stored")
	}
	st := f.must(f.desk.GenerateAssetDetail(WithExternalAI(f.ctx, NewExternalAIRun(ExternalAIPaste, detailReply("dán", "ink"))), project, "AST-ASTRO-001", ""))
	if !st.Assets.Items[0].HasDetail || st.Assets.Items[0].Portrayal != "dán" {
		t.Error("the pasted reply must be written")
	}
	bad := NewExternalAIRun(ExternalAIPaste, `{"detail":{"portrayal":"x","palette_tokens":["pink"]}}`)
	if _, err := f.desk.GenerateAssetDetail(WithExternalAI(f.ctx, bad), project, "AST-SPACE-001", ""); !errors.Is(err, domain.ErrUnreadableReply) {
		t.Errorf("a pasted reply that does not fit: %v", err)
	}
}

func TestStoryboardAI_EditingTheUnitsOfAnAssetReopensTheListAndIsTheCreators(t *testing.T) {
	f := newSbFixture(t)
	f.listedAssets()
	f.queue(domain.PromptRoleStoryboardAssetDetail, detailReply("p", "ink"), detailReply("q", "navy"))
	f.must(f.desk.GenerateAllAssetDetails(f.ctx, project, ""))
	f.must(f.desk.ApproveAssets(f.ctx, project))
	st := f.must(f.desk.EditAsset(f.ctx, project, "AST-ASTRO-001", jfields(t, map[string]any{"units": []string{"VP-003"}})))
	if st.Assets.Approved || !containsStr(st.Assets.Items[0].CreatorFields, "units") || st.Assets.Items[0].Units[0] != "VP-003" {
		t.Errorf("assets %+v approved %v", st.Assets.Items[0], st.Assets.Approved)
	}
}
