package application

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"visual/internal/domain"
)

func TestStoryboardDesk_NeedsTheLockedStepsUnderIt(t *testing.T) {
	f := newSbFixture(t)
	f.store.sequences = nil
	if _, err := f.desk.State(f.ctx, project); !errors.Is(err, domain.ErrSequenceNotLocked) {
		t.Errorf("no locked sequence: want ErrSequenceNotLocked, got %v", err)
	}
	f.store.plans = nil
	if _, err := f.desk.State(f.ctx, project); !errors.Is(err, domain.ErrPlanNotLocked) {
		t.Errorf("no locked plan: want ErrPlanNotLocked, got %v", err)
	}
	f.store.concepts = nil
	if _, err := f.desk.State(f.ctx, project); !errors.Is(err, domain.ErrConceptNotLocked) {
		t.Errorf("no locked concept: want ErrConceptNotLocked, got %v", err)
	}
}

func TestStoryboardDesk_TheScreenListsEveryUnitInSequenceOrder(t *testing.T) {
	f := newSbFixture(t)
	st := f.must(f.desk.State(f.ctx, project))
	if len(st.Beats) != 2 || len(st.Beats[0].Units) != 2 || len(st.EmptyUnits) != 3 {
		t.Fatalf("beats %d, empty units %v", len(st.Beats), st.EmptyUnits)
	}
	if st.Beats[0].Units[1].UnitID != "VP-002" || st.Beats[0].Units[1].PurposeInFlow != "giữa" {
		t.Errorf("unit %+v", st.Beats[0].Units[1])
	}
	if st.CanLock || len(st.Blockers) == 0 {
		t.Error("an empty storyboard cannot lock")
	}
	if st.DefaultModel != "ollama" || st.Kit.RefID != "R12" || st.Kit.Missing {
		t.Errorf("model %q kit %+v", st.DefaultModel, st.Kit)
	}
}

func TestStoryboardDesk_AssetsFromTheCreatorAreChecksAtRendering(t *testing.T) {
	f := newSbFixture(t)
	st := f.must(f.desk.AddAsset(f.ctx, project, jfields(t, map[string]any{"concept_id": "astro", "name": "Astro", "portrayal": "phi hành gia", "tier": "hero"})))
	if len(st.Assets.Items) != 1 || st.Assets.Items[0].AssetID != "AST-ASTRO-001" || st.Assets.Items[0].Status != domain.AssetPlanned {
		t.Fatalf("assets %+v", st.Assets.Items)
	}
	st = f.must(f.desk.EditAsset(f.ctx, project, "AST-ASTRO-001", jfields(t, map[string]any{"portrayal": "người leo núi"})))
	if got := st.Assets.Items[0]; got.Portrayal != "người leo núi" || !containsStr(got.CreatorFields, "portrayal") {
		t.Errorf("asset %+v", got)
	}
	if _, err := f.desk.EditAsset(f.ctx, project, "AST-ASTRO-001", jfields(t, map[string]any{"scale": 3})); !errors.Is(err, domain.ErrInvalidEdit) {
		t.Errorf("an unknown field: want ErrInvalidEdit, got %v", err)
	}
	if _, err := f.desk.EditAsset(f.ctx, project, "AST-NOPE-001", jfields(t, map[string]any{"portrayal": "x"})); !errors.Is(err, domain.ErrUnknownAsset) {
		t.Errorf("an unknown asset: want ErrUnknownAsset, got %v", err)
	}
}

func TestStoryboardDesk_DrawingNeedsAKitAndApprovalNeedsADrawing(t *testing.T) {
	f := newSbFixtureWith(t, fakeKits{err: ErrNoReferencePicked})
	f.must(f.desk.AddAsset(f.ctx, project, jfields(t, map[string]any{"concept_id": "astro", "name": "Astro", "portrayal": "p"})))
	if _, err := f.desk.DrawAsset(f.ctx, project, "AST-ASTRO-001"); !errors.Is(err, domain.ErrNoKit) {
		t.Fatalf("a video with no kit: want ErrNoKit, got %v", err)
	}
	if len(f.rendering.draws) != 0 {
		t.Error("nothing may be drawn without a kit")
	}
	st := f.must(f.desk.State(f.ctx, project))
	if !st.Kit.Missing || st.Kit.Message == "" {
		t.Errorf("the screen must say there is no kit: %+v", st.Kit)
	}
	var re *RenderingError
	if _, err := f.desk.ApproveAsset(f.ctx, project, "AST-ASTRO-001", "Hố đen"); !errors.As(err, &re) || re.Status != 409 {
		t.Errorf("approving an undrawn asset: want rendering's 409, got %v", err)
	}
}

func TestStoryboardDesk_DrawRedrawApproveReachRenderingWithTheStyleOfStep8(t *testing.T) {
	f := newSbFixture(t)
	f.must(f.desk.AddAsset(f.ctx, project, jfields(t, map[string]any{"concept_id": "astro", "name": "Astro", "portrayal": "p"})))
	f.must(f.desk.DrawAsset(f.ctx, project, "AST-ASTRO-001"))
	st := f.must(f.desk.RedrawAsset(f.ctx, project, "AST-ASTRO-001", "to hơn"))
	st = f.must(f.desk.ApproveAsset(f.ctx, project, "AST-ASTRO-001", "Hố đen"))
	if a := st.Assets.Items[0]; a.Status != domain.AssetApproved || a.Version != 2 {
		t.Errorf("asset %+v", a.StoryboardAsset)
	}
	if len(f.rendering.approves) != 1 || f.rendering.approves[0].RefID != "R12" || f.rendering.approves[0].ProjectName != "Hố đen" {
		t.Errorf("approve request %+v", f.rendering.approves)
	}
	st = f.must(f.desk.UnapproveAsset(f.ctx, project, "AST-ASTRO-001"))
	if st.Assets.Items[0].Status != domain.AssetDrawn {
		t.Errorf("status %q", st.Assets.Items[0].Status)
	}
}

func TestStoryboardDesk_TheAssetListIsApprovedAsAWholeAndAnEditReopensIt(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	if _, err := f.desk.ApproveAssets(f.ctx, project); err != nil {
		t.Fatal(err)
	}
	st := f.must(f.desk.State(f.ctx, project))
	if !st.Assets.Approved {
		t.Fatal("the list must read approved")
	}
	st = f.must(f.desk.EditAsset(f.ctx, project, "AST-SPACE-001", jfields(t, map[string]any{"portrayal": "đêm"})))
	if st.Assets.Approved {
		t.Error("editing an asset reopens the list")
	}
	st = f.must(f.desk.UnapproveAssets(f.ctx, project))
	if st.Assets.Approved {
		t.Error("unapproving must reopen the list")
	}
	empty := newSbFixture(t)
	if _, err := empty.desk.ApproveAssets(empty.ctx, project); !errors.Is(err, domain.ErrUnitNotWritten) {
		t.Errorf("an empty list: want ErrUnitNotWritten, got %v", err)
	}
}

func TestStoryboardDesk_AnAssetAShotUsesCannotBeDeleted(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	f.writeUnit1()
	_, err := f.desk.DeleteAsset(f.ctx, project, "AST-ASTRO-001")
	if !errors.Is(err, domain.ErrAssetInUse) || !strings.Contains(err.Error(), "SH-001") {
		t.Fatalf("want ErrAssetInUse naming the shot, got %v", err)
	}
	if len(f.rendering.deleted) != 0 {
		t.Error("rendering must not be asked to delete it")
	}
	// Once its only shot goes, the asset can.
	f.must(f.desk.AddShot(f.ctx, project, "VP-001", jfields(t, map[string]any{"script_refs": []string{"b1.1"}, "composition": map[string]any{
		"subject": "Phi hành gia", "subject_position": "center", "main_object": "", "main_object_position": "", "camera_view": "wide"}})))
	f.must(f.desk.DeleteShot(f.ctx, project, "VP-001", "SH-001"))
	st := f.must(f.desk.DeleteAsset(f.ctx, project, "AST-ASTRO-001"))
	if len(st.Assets.Items) != 1 || len(f.rendering.deleted) != 1 {
		t.Errorf("assets %d, deleted %v", len(st.Assets.Items), f.rendering.deleted)
	}
}

func TestStoryboardDesk_ShotsAreEditedAddedAndDeletedByTheCreator(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	st := f.writeUnit1()
	if sh := unitOf(st, "VP-001").Shots[0]; sh.ShotID != "SH-001" || sh.Status != domain.UnitPending || !sh.Frame.Drawn {
		t.Fatalf("shot %+v", sh)
	}
	st = f.must(f.desk.EditShot(f.ctx, project, "VP-001", "SH-001", jfields(t, map[string]any{"on_screen_text": []string{"Góc nhìn B"}})))
	sh := unitOf(st, "VP-001").Shots[0]
	if len(sh.CreatorFields) != 1 || sh.CreatorFields[0] != "on_screen_text" || sh.OnScreenText[0] != "Góc nhìn B" {
		t.Errorf("shot %+v", sh)
	}
	if len(f.rendering.frames) != 2 || f.rendering.frames[0].SpecHash == f.rendering.frames[1].SpecHash {
		t.Errorf("changing the text must draw the frame again from another spec: %d frames", len(f.rendering.frames))
	}
	if _, err := f.desk.EditShot(f.ctx, project, "VP-001", "SH-001", jfields(t, map[string]any{"shot_id": "SH-009"})); !errors.Is(err, domain.ErrInvalidEdit) {
		t.Errorf("shot_id is the machine's: want ErrInvalidEdit, got %v", err)
	}
	if _, err := f.desk.EditShot(f.ctx, project, "VP-001", "SH-009", jfields(t, map[string]any{"notes": []string{"x"}})); !errors.Is(err, domain.ErrUnknownShot) {
		t.Errorf("want ErrUnknownShot, got %v", err)
	}
	if _, err := f.desk.EditShot(f.ctx, project, "VP-099", "SH-001", jfields(t, map[string]any{"notes": []string{"x"}})); !errors.Is(err, domain.ErrUnknownPlanUnit) {
		t.Errorf("want ErrUnknownPlanUnit, got %v", err)
	}
	if _, err := f.desk.DeleteShot(f.ctx, project, "VP-001", "SH-001"); !errors.Is(err, domain.ErrLastShot) {
		t.Fatalf("the last shot of a unit stays: got %v", err)
	}
	st = f.must(f.desk.AddShot(f.ctx, project, "VP-001", jfields(t, map[string]any{"script_refs": []string{"b1.1"}, "shot_purpose": "thêm",
		"composition": map[string]any{"subject": "AST-ASTRO-001", "subject_position": "center", "main_object": "", "main_object_position": "", "camera_view": "close"},
		"asset_refs":  []string{"AST-ASTRO-001"}})))
	if got := unitOf(st, "VP-001").Shots; len(got) != 2 || got[1].ShotID != "SH-002" {
		t.Fatalf("shots %+v", got)
	}
	st = f.must(f.desk.DeleteShot(f.ctx, project, "VP-001", "SH-002"))
	st = f.must(f.desk.AddShot(f.ctx, project, "VP-001", jfields(t, map[string]any{"script_refs": []string{"b1.1"}})))
	if got := unitOf(st, "VP-001").Shots; got[len(got)-1].ShotID != "SH-003" {
		t.Errorf("the number of a deleted shot is not given again: %+v", got)
	}
}

func TestStoryboardDesk_AFrameThatFailsToDrawIsTheShotsErrorAndDoesNotBlockSaving(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	f.rendering.frameErr = &RenderingError{Status: 503, Message: "bộ dựng không chạy được"}
	st := f.writeUnit1()
	sh := unitOf(st, "VP-001").Shots[0]
	if sh.ShotID != "SH-001" || sh.Frame.Drawn || !strings.Contains(sh.Frame.Error, "bộ dựng không chạy được") {
		t.Fatalf("the shot must be saved with the frame error: %+v", sh)
	}
	if _, err := f.desk.ApproveShot(f.ctx, project, "VP-001", "SH-001"); !errors.Is(err, domain.ErrUnitInvalid) {
		t.Errorf("a shot with no frame cannot be approved: %v", err)
	}
	// Drawing again once rendering is back clears the error.
	f.rendering.frameErr = nil
	st = f.must(f.desk.DrawFrame(f.ctx, project, "VP-001", "SH-001"))
	if sh := unitOf(st, "VP-001").Shots[0]; !sh.Frame.Drawn || sh.Frame.Error != "" {
		t.Errorf("frame %+v", sh.Frame)
	}
	if _, err := f.desk.ApproveShot(f.ctx, project, "VP-001", "SH-001"); err != nil {
		t.Errorf("approve: %v", err)
	}
}

func TestStoryboardDesk_AFrameWithoutAKitIsTheShotsErrorAndNeverFakeSuccess(t *testing.T) {
	f := newSbFixtureWith(t, fakeKits{err: ErrReferenceKitMissing})
	f.approvedAssets()
	st := f.writeUnit1()
	sh := unitOf(st, "VP-001").Shots[0]
	if sh.Frame.Drawn || sh.Frame.Error == "" {
		t.Fatalf("frame %+v", sh.Frame)
	}
	if len(f.rendering.frames) != 0 {
		t.Error("rendering must not be asked without a kit")
	}
}

func TestStoryboardDesk_ARefusalOfRenderingKeepsTheLastImageAndShowsTheReason(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	f.writeUnit1()
	f.rendering.frameErr = &RenderingError{Status: 409, Message: "tài sản chưa duyệt"}
	st := f.must(f.desk.EditShot(f.ctx, project, "VP-001", "SH-001", jfields(t, map[string]any{"on_screen_text": []string{"Đổi"}})))
	sh := unitOf(st, "VP-001").Shots[0]
	if !sh.Frame.Drawn || !sh.Frame.Stale || !strings.Contains(sh.Frame.Error, "tài sản chưa duyệt") {
		t.Errorf("an image rendering still has stays, stale, with the reason: %+v", sh.Frame)
	}
}

func TestStoryboardDesk_RedrawingAnAssetMakesTheFramesOfItsShotsStale(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	f.writeUnit1()
	f.must(f.desk.ApproveShot(f.ctx, project, "VP-001", "SH-001"))
	f.must(f.desk.RedrawAsset(f.ctx, project, "AST-ASTRO-001", ""))
	f.must(f.desk.ApproveAsset(f.ctx, project, "AST-ASTRO-001", "Hố đen"))
	st := f.must(f.desk.State(f.ctx, project))
	u := unitOf(st, "VP-001")
	if !u.Shots[0].Frame.Stale || !u.Stale {
		t.Errorf("the frame and the unit must read stale: %+v %v", u.Shots[0].Frame, u.StaleSources)
	}
	if _, err := f.desk.ApproveShot(f.ctx, project, "VP-001", "SH-001"); !errors.Is(err, domain.ErrUnitInvalid) {
		t.Errorf("a shot with a stale frame cannot be approved: %v", err)
	}
	st = f.must(f.desk.DrawFrame(f.ctx, project, "VP-001", "SH-001"))
	if st.Beats[0].Units[0].Shots[0].Frame.Stale {
		t.Error("drawing again clears the stale frame")
	}
}

func TestStoryboardDesk_ApproveAllLeavesShotsWithProblems(t *testing.T) {
	f := newSbFixture(t)
	f.approvedAssets()
	f.queue(domain.PromptRoleStoryboardUnit,
		shotsReply(shotJSON([]string{"b1.1"}, nil, "mở")),
		shotsReply(shotJSON([]string{"b1.2", "b1.3"}, nil, "giữa")))
	f.must(f.desk.GenerateUnit(f.ctx, project, "VP-001", ""))
	f.must(f.desk.GenerateUnit(f.ctx, project, "VP-002", ""))
	// A shot whose text is too long for the rules cannot pass its checks.
	f.must(f.desk.EditShot(f.ctx, project, "VP-002", "SH-002", jfields(t, map[string]any{"on_screen_text": []string{"một hai ba bốn năm sáu"}})))
	st := f.must(f.desk.ApproveAll(f.ctx, project))
	if st.ShotsApproved != 1 || unitOf(st, "VP-001").Shots[0].Status != domain.UnitApproved || unitOf(st, "VP-002").Shots[0].Status != domain.UnitPending {
		t.Errorf("approved %d: %+v", st.ShotsApproved, unitOf(st, "VP-002").Shots[0].Problems)
	}
	if !unitOf(st, "VP-001").Approved {
		t.Error("a unit whose shots are all approved reads approved")
	}
	st = f.must(f.desk.UnapproveShot(f.ctx, project, "VP-001", "SH-001"))
	if unitOf(st, "VP-001").Approved {
		t.Error("unapproving reopens the unit")
	}
}

func TestStoryboardDesk_ModelIsOneOfTheCatalog(t *testing.T) {
	f := newSbFixture(t)
	st := f.must(f.desk.SetModel(f.ctx, project, "deepseek-chat"))
	if st.Model != "deepseek-chat" {
		t.Errorf("model %q", st.Model)
	}
	if _, err := f.desk.SetModel(f.ctx, project, "nope"); !errors.Is(err, ErrUnknownModel) {
		t.Errorf("want ErrUnknownModel, got %v", err)
	}
	st = f.must(f.desk.SetModel(f.ctx, project, ""))
	if st.Model != "" || st.DefaultModel != "ollama" {
		t.Errorf("model %q default %q", st.Model, st.DefaultModel)
	}
}

func TestDeleteProject_AsksRenderingToRemoveTheStoryboardAndLogsItsFailure(t *testing.T) {
	store := &fakeProjectData{}
	rend := newFakeRendering()
	uc := NewDeleteProjectUseCase(store).WithStoryboardPurge(rend, slog.Default())
	if err := uc.Execute(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	if len(rend.purged) != 1 || rend.purged[0] != "p1" {
		t.Fatalf("purged %v", rend.purged)
	}
	rend.purgeErr = errors.New("rendering down")
	if err := uc.Execute(context.Background(), "p2"); err != nil {
		t.Errorf("a failed purge must not fail the delete: %v", err)
	}
	failing := &fakeProjectData{err: errors.New("db down")}
	rend2 := newFakeRendering()
	if err := NewDeleteProjectUseCase(failing).WithStoryboardPurge(rend2, slog.Default()).Execute(context.Background(), "p3"); err == nil || len(rend2.purged) != 0 {
		t.Errorf("when the rows are not deleted rendering is left alone: err %v purged %v", err, rend2.purged)
	}
}
