package application

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"visual/internal/domain"
)

func TestFontsFromRendering_ListIsPassedToThePromptAndChecked(t *testing.T) {
	r := newRig(t)
	r.queueAll()
	mustState(t)(r.desk.GenerateUnit(ctx, project, domain.UnitTypography, ""))
	if !strings.Contains(r.llm.calls[0].System, "Be Vietnam Pro") {
		t.Error("the prompt of the typography part carries the fonts rendering listed")
	}
	st := mustState(t)(r.desk.State(ctx, project))
	if st.FontsError != "" || len(st.Fonts) != 1 {
		t.Errorf("fonts %v error %q", st.Fonts, st.FontsError)
	}
	r.llm.replies[domain.PromptRoleOf(domain.UnitTypography)] = []string{strings.Replace(goodReplies[domain.UnitTypography], "Be Vietnam Pro", "Comic Sans", 1)}
	st = mustState(t)(r.desk.GenerateUnit(ctx, project, domain.UnitTypography, ""))
	u := unitView(st, domain.UnitTypography)
	if len(u.Problems) != 1 || !strings.Contains(u.Problems[0], "Comic Sans") {
		t.Errorf("a font outside the list fails the part: %v", u.Problems)
	}
	if _, err := r.desk.Approve(ctx, project, domain.UnitTypography); !errors.Is(err, domain.ErrUnitInvalid) {
		t.Errorf("approve: %v", err)
	}
}

func TestRenderingDown_BlocksTheTypographyPartButNotTheScreen(t *testing.T) {
	r := newRig(t)
	r.queueAll()
	mustState(t)(r.desk.Generate(ctx, project, "", false))
	mustState(t)(r.desk.ApproveAll(ctx, project))

	r.fonts.err = fmt.Errorf("%w: connection refused", ErrRenderingFailed)
	st := mustState(t)(r.desk.State(ctx, project))
	if !strings.Contains(st.FontsError, "rendering") || st.CanLock {
		t.Fatalf("the screen still loads, names rendering and cannot be locked: %q %v", st.FontsError, st.CanLock)
	}
	if !hasLine(st.Blockers, "Không kiểm được font") {
		t.Errorf("blockers %v", st.Blockers)
	}
	u := unitView(st, domain.UnitTypography)
	if len(u.Problems) != 1 || !strings.Contains(u.Problems[0], "rendering") {
		t.Errorf("problems %v", u.Problems)
	}
	if _, err := r.desk.Lock(ctx, project, ""); !errors.Is(err, domain.ErrLockNotReady) {
		t.Errorf("lock: %v", err)
	}

	mustState(t)(r.desk.Unapprove(ctx, project, domain.UnitTypography))
	if _, err := r.desk.Approve(ctx, project, domain.UnitTypography); !errors.Is(err, domain.ErrUnitInvalid) {
		t.Errorf("approve while rendering is down: %v", err)
	}
	before := len(r.llm.calls)
	if _, err := r.desk.GenerateUnit(ctx, project, domain.UnitTypography, ""); !errors.Is(err, ErrRenderingFailed) {
		t.Errorf("generate typography: %v", err)
	}
	if len(r.llm.calls) != before {
		t.Error("no model is asked for the typography part without the font list")
	}
	r.llm.replies[domain.PromptRoleOf(domain.UnitVisualLanguage)] = []string{goodReplies[domain.UnitVisualLanguage]}
	if _, err := r.desk.GenerateUnit(ctx, project, domain.UnitVisualLanguage, ""); err != nil {
		t.Errorf("the other parts do not need rendering: %v", err)
	}

	r.fonts.err = nil
	st = mustState(t)(r.desk.ApproveAll(ctx, project))
	if st.FontsError != "" || unitView(st, domain.UnitTypography).Status != domain.UnitApproved {
		t.Errorf("once rendering answers the part can be approved again: %q", st.FontsError)
	}
}
