package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"authoring/internal/domain"
)

// CR-052 — deleting a drawing a video still needs, and the Hình mẫu folder.

type fakeProjectStatus struct {
	status map[string]domain.ProjectStatus
	fail   bool
}

func (f fakeProjectStatus) GetStatus(_ context.Context, id string) (domain.ProjectStatus, error) {
	if f.fail {
		return "", errors.New("orchestrator down")
	}
	s, ok := f.status[id]
	if !ok {
		return "", domain.ErrProjectNotFound
	}
	return s, nil
}

func approvedBus(repo *fakeIllustrationRepo) domain.Illustration {
	bus := domain.Illustration{ID: "bus", Name: "Bus", Title: "Xe buýt", FolderID: "phuong-tien", Code: busCode,
		Usage: "<Bus color /> — 320×210", Status: domain.IllustrationApproved, Version: 3}
	repo.rows[bus.ID] = bus
	return bus
}

func TestDeleteIsRefusedWhileAProjectBeforeTheResultUsesTheDrawing(t *testing.T) {
	repo := newFakeIllustrationRepo()
	approvedBus(repo)
	repo.users["bus"] = []IllustrationUser{{ProjectID: "p-render", Topic: "Sâu răng"}, {ProjectID: "p-done", Topic: "Xe cộ"}}
	repo.linked["bus"] = []string{"p-render", "p-done"}
	projects := fakeProjectStatus{status: map[string]domain.ProjectStatus{
		"p-render": domain.StatusRendering, "p-done": domain.StatusPublished}}
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{}).WithProjectStatus(projects)

	err := uc.Delete(context.Background(), "bus")
	var inUse *IllustrationInUseError
	if !errors.As(err, &inUse) || !errors.Is(err, ErrIllustrationInUse) {
		t.Fatalf("want in-use error, got %v", err)
	}
	if len(inUse.Projects) != 1 || inUse.Projects[0].ProjectID != "p-render" || !strings.Contains(err.Error(), `"Sâu răng"`) {
		t.Fatalf("only the project still working is named: %+v %q", inUse.Projects, err)
	}
	if _, ok := repo.rows["bus"]; !ok {
		t.Fatal("drawing deleted anyway")
	}
}

func TestDeleteGoesThroughWhenEveryUserIsDoneOrGone(t *testing.T) {
	repo := newFakeIllustrationRepo()
	approvedBus(repo)
	repo.users["bus"] = []IllustrationUser{{ProjectID: "p-done"}, {ProjectID: "p-gone"}}
	repo.linked["bus"] = []string{"p-done", "p-gone"}
	projects := fakeProjectStatus{status: map[string]domain.ProjectStatus{"p-done": domain.StatusReadyToPublish}}
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{}).WithProjectStatus(projects)
	if err := uc.Delete(context.Background(), "bus"); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.rows["bus"]; ok {
		t.Fatal("drawing not deleted")
	}
}

func TestDeleteRefusesWhenTheUsageCannotBeChecked(t *testing.T) {
	repo := newFakeIllustrationRepo()
	approvedBus(repo)
	repo.users["bus"] = []IllustrationUser{{ProjectID: "p1"}}
	for name, uc := range map[string]*IllustrationsUseCase{
		"orchestrator down": NewIllustrationsUseCase(repo, &fakeRenderer{}).WithProjectStatus(fakeProjectStatus{fail: true}),
		"not wired":         NewIllustrationsUseCase(repo, &fakeRenderer{}),
	} {
		if err := uc.Delete(context.Background(), "bus"); !errors.Is(err, ErrIllustrationUsageUnknown) {
			t.Errorf("%s: want usage-unknown, got %v", name, err)
		}
	}
	if _, ok := repo.rows["bus"]; !ok {
		t.Fatal("drawing deleted without the check")
	}
}

func TestDeleteRefusesAProjectThatLinkedTheDrawingAfterTheCheck(t *testing.T) {
	repo := newFakeIllustrationRepo()
	approvedBus(repo)
	repo.linked["bus"] = []string{"p-new"} // linked, but not found by the usage query the first time round
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{}).WithProjectStatus(fakeProjectStatus{})
	if err := uc.Delete(context.Background(), "bus"); !errors.Is(err, ErrIllustrationUsageUnknown) {
		t.Fatalf("want a refusal, got %v", err)
	}
	if _, ok := repo.rows["bus"]; !ok {
		t.Fatal("deleted under a project that was never checked")
	}
}

func TestKitAndExemplarsCannotBeDeletedEditedOrReReviewed(t *testing.T) {
	repo := newFakeIllustrationRepo()
	repo.rows["ex"] = domain.Illustration{ID: "ex", Name: "Cat", FolderID: domain.ExemplarFolderID, Exemplar: true,
		HomeFolderID: "dong-vat", Code: "export function Cat() {}", Status: domain.IllustrationApproved}
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{}).WithProjectStatus(fakeProjectStatus{})
	ctx := context.Background()
	for _, id := range []string{"builtin-Tooth", "ex"} {
		if err := uc.Delete(ctx, id); !errors.Is(err, ErrIllustrationReadOnly) {
			t.Errorf("%s delete: %v", id, err)
		}
		if _, err := uc.Update(ctx, id, domain.Illustration{Code: busCode}); !errors.Is(err, ErrIllustrationReadOnly) {
			t.Errorf("%s update: %v", id, err)
		}
		if _, err := uc.SetStatus(ctx, id, domain.IllustrationDraft); !errors.Is(err, ErrIllustrationReadOnly) {
			t.Errorf("%s status: %v", id, err)
		}
	}
}

func TestNothingIsSavedIntoTheExemplarFolderByHand(t *testing.T) {
	repo := newFakeIllustrationRepo()
	bus := approvedBus(repo)
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{})
	ctx := context.Background()
	if _, err := uc.Create(ctx, domain.Illustration{Name: "Van", FolderID: domain.ExemplarFolderID, Code: busCode}); !errors.Is(err, ErrExemplarFolder) {
		t.Fatalf("create: %v", err)
	}
	if _, err := uc.Update(ctx, bus.ID, domain.Illustration{FolderID: domain.ExemplarFolderID}); !errors.Is(err, ErrExemplarFolder) {
		t.Fatalf("move: %v", err)
	}
}

func TestMakeExemplarCopiesUnderANewNameAndRendersIt(t *testing.T) {
	repo, rend := newFakeIllustrationRepo(), &fakeRenderer{}
	bus := approvedBus(repo)
	uc := NewIllustrationsUseCase(repo, rend)
	cp, err := uc.MakeExemplar(context.Background(), bus.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cp.Name != "BusMau" || cp.FolderID != domain.ExemplarFolderID || !cp.Exemplar || cp.SourceID != bus.ID ||
		cp.Status != domain.IllustrationApproved || cp.Title != bus.Title || !cp.HasPreview {
		t.Fatalf("copy: %+v", cp)
	}
	if !strings.Contains(cp.Code, "export function BusMau()") || cp.Usage != "<BusMau color /> — 320×210" {
		t.Fatalf("copy not renamed: %q %q", cp.Code, cp.Usage)
	}
	if rend.calls[len(rend.calls)-1] != "BusMau|"+cp.Code {
		t.Fatal("the copy must pass the renderer under its own name")
	}
	if repo.rows[bus.ID].Code != busCode {
		t.Fatal("the source changed")
	}
	if _, err := uc.MakeExemplar(context.Background(), bus.ID); !errors.Is(err, ErrAlreadyExemplar) {
		t.Fatalf("second copy: %v", err)
	}
	if _, err := uc.MakeExemplar(context.Background(), cp.ID); !errors.Is(err, ErrAlreadyExemplar) {
		t.Fatalf("copy of a copy: %v", err)
	}
	ex, _ := uc.Exemplars(context.Background())
	if len(ex) != 1 || ex[0].ID != cp.ID {
		t.Fatalf("exemplars: %+v", ex)
	}
}

func TestMakeExemplarRefusesDraftsKitAndAFullFolder(t *testing.T) {
	repo := newFakeIllustrationRepo()
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{})
	ctx := context.Background()
	repo.rows["draft"] = domain.Illustration{ID: "draft", Name: "Van", FolderID: "phuong-tien", Code: busCode, Status: domain.IllustrationDraft}
	for _, id := range []string{"draft", "builtin-Tooth"} {
		if _, err := uc.MakeExemplar(ctx, id); !errors.Is(err, ErrNotExemplarSource) {
			t.Errorf("%s: %v", id, err)
		}
	}
	for n := 0; n < domain.MaxExemplars; n++ {
		id := fmt.Sprintf("ex%d", n)
		repo.rows[id] = domain.Illustration{ID: id, Name: fmt.Sprintf("Ex%d", n), FolderID: domain.ExemplarFolderID, Exemplar: true}
	}
	bus := approvedBus(repo)
	if _, err := uc.MakeExemplar(ctx, bus.ID); !errors.Is(err, ErrExemplarLimit) {
		t.Fatalf("sixth exemplar: %v", err)
	}
}

func TestUnmakeDeletesACopyAndFilesAnOriginalBack(t *testing.T) {
	repo := newFakeIllustrationRepo()
	bus := approvedBus(repo)
	repo.rows["cat"] = domain.Illustration{ID: "cat", Name: "Cat", FolderID: domain.ExemplarFolderID, Exemplar: true,
		HomeFolderID: "dong-vat", Code: "export function Cat() {}", Status: domain.IllustrationApproved}
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{}).WithProjectStatus(fakeProjectStatus{})
	ctx := context.Background()
	cp, err := uc.MakeExemplar(ctx, bus.ID)
	if err != nil {
		t.Fatal(err)
	}
	if out, err := uc.UnmakeExemplar(ctx, cp.ID); err != nil || out != nil {
		t.Fatalf("unmake copy: %+v %v", out, err)
	}
	if _, ok := repo.rows[cp.ID]; ok {
		t.Fatal("copy still there")
	}
	if _, ok := repo.rows[bus.ID]; !ok {
		t.Fatal("source went with the copy")
	}
	out, err := uc.UnmakeExemplar(ctx, "cat")
	if err != nil || out == nil || out.Exemplar || out.FolderID != "dong-vat" || out.ReadOnly() {
		t.Fatalf("unmake original: %+v %v", out, err)
	}
	if _, err := uc.UnmakeExemplar(ctx, bus.ID); !errors.Is(err, ErrNotExemplar) {
		t.Fatalf("unmake a plain drawing: %v", err)
	}
}

func TestUnmakeACopyAVideoStillUsesIsRefused(t *testing.T) {
	repo := newFakeIllustrationRepo()
	repo.rows["cp"] = domain.Illustration{ID: "cp", Name: "BusMau", FolderID: domain.ExemplarFolderID, Exemplar: true,
		SourceID: "bus", Code: busCode, Status: domain.IllustrationApproved}
	repo.users["cp"] = []IllustrationUser{{ProjectID: "p1", Topic: "Xe"}}
	uc := NewIllustrationsUseCase(repo, &fakeRenderer{}).
		WithProjectStatus(fakeProjectStatus{status: map[string]domain.ProjectStatus{"p1": domain.StatusDraft}})
	if _, err := uc.UnmakeExemplar(context.Background(), "cp"); !errors.Is(err, ErrIllustrationInUse) {
		t.Fatalf("want in-use, got %v", err)
	}
}
