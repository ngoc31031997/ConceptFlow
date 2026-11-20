package domain

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// IDs of the machine checks of step 11. The m-shot checks block approving a shot
// and locking; the others are warnings the Creator can accept with a reason.
const (
	CheckShotUnit   = "m-shot-unit"
	CheckShotScript = "m-shot-script"
	CheckShotAssets = "m-shot-assets"
	CheckShotEnum   = "m-shot-enum"
	CheckShotText   = "m-shot-text"
	CheckShotFrame  = "m-shot-frame"

	CheckAssetUnused = "m-asset-unused"
	CheckUnitCover   = "m-unit-cover"
	CheckAssetCount  = "m-asset-count"
)

// IsBlockingCheck reports whether a failing check of step 11 cannot be accepted.
func IsBlockingCheck(id string) bool { return strings.HasPrefix(id, "m-shot-") }

// minAssetShots is how many shots an asset has to appear in to be worth drawing
// once; an asset in fewer is flagged.
const minAssetShots = 2

// ShotProblem is one thing that fails in a shot, under the check it belongs to.
type ShotProblem struct {
	Check  string `json:"check"`
	Detail string `json:"detail"`
}

// ShotContext is what checking one shot reads besides the shot.
type ShotContext struct {
	// UnitExists says the shot's unit is a unit of the locked plan.
	UnitExists bool
	// UnitRefs are the script line ids of the shot's unit.
	UnitRefs []string
	// Assets are the project's assets by id.
	Assets map[string]StoryboardAsset
	// Typography is the locked visual concept's rules for on-screen text.
	Typography Typography
	// Frame is the shot's last drawing; SpecHash the hash of the spec it has now,
	// empty when the spec cannot be built (an asset it names is missing).
	Frame    DrawnFrame
	SpecHash string
}

// CheckShot lists what fails in a shot; a shot with problems cannot be approved.
func CheckShot(sh Shot, ctx ShotContext) []ShotProblem {
	var out []ShotProblem
	add := func(check, format string, a ...any) {
		out = append(out, ShotProblem{Check: check, Detail: fmt.Sprintf(format, a...)})
	}
	if !ctx.UnitExists {
		add(CheckShotUnit, "%s thuộc unit không có trong kế hoạch hình đã chốt", sh.ShotID)
	}
	if len(sh.ScriptRefs) == 0 {
		add(CheckShotScript, "%s chưa phủ câu nào", sh.ShotID)
	}
	for _, r := range sh.ScriptRefs {
		if !contains(ctx.UnitRefs, r) {
			add(CheckShotScript, "%s phủ câu %s không thuộc unit", sh.ShotID, r)
		}
	}
	for _, id := range sh.AssetRefs {
		a, ok := ctx.Assets[id]
		switch {
		case !ok:
			add(CheckShotAssets, "%s dùng tài sản %s không có", sh.ShotID, id)
		case a.Status != AssetApproved:
			add(CheckShotAssets, "%s dùng tài sản %s chưa duyệt", sh.ShotID, id)
		}
	}
	for _, msg := range compositionProblems(sh.Composition) {
		add(CheckShotEnum, "%s: %s", sh.ShotID, msg)
	}
	for i, t := range sh.OnScreenText {
		if strings.TrimSpace(t) == "" {
			add(CheckShotText, "%s: dòng chữ %d trống", sh.ShotID, i+1)
			continue
		}
		if max := ctx.Typography.Labels.MaxWords; max > 0 && len(strings.Fields(t)) > max {
			add(CheckShotText, "%s: dòng chữ «%s» dài quá %d từ của luật chữ", sh.ShotID, t, max)
		}
	}
	switch {
	case !ctx.Frame.Drawn():
		add(CheckShotFrame, "%s chưa có ảnh khung", sh.ShotID)
	case ctx.SpecHash != "" && ctx.Frame.Stale(ctx.SpecHash):
		add(CheckShotFrame, "%s có ảnh khung cũ, cần dựng lại", sh.ShotID)
	}
	return out
}

// compositionProblems lists what is outside the closed vocabularies or missing in
// a composition.
func compositionProblems(c ShotComposition) []string {
	var p problems
	p.need(c.Subject, "thiếu đối tượng chính (subject)")
	p.enum(c.SubjectPosition, ShotPositions, "vị trí của đối tượng chính")
	p.enum(c.CameraView, ShotCameraViews, "góc máy")
	if strings.TrimSpace(c.MainObject) == "" {
		if c.MainObjectPosition != "" {
			p.add("có vị trí của vật chính nhưng không có vật chính")
		}
	} else {
		p.enum(c.MainObjectPosition, ShotPositions, "vị trí của vật chính")
	}
	return p.list
}

// StoryboardState is everything step 11 has stored, read for checking and
// locking.
type StoryboardState struct {
	ProjectID string
	// Assets are the project's assets; Units the stored shots by plan unit id and
	// Frames the last drawing of every shot by shot id.
	Assets []StoryboardAsset
	Units  map[string]UnitShots
	Frames map[string]DrawnFrame
	// Kit is the project's reference kit, as the frame spec carries it.
	Kit json.RawMessage
	// Review is the review of the whole storyboard.
	Review ReviewState
}

// AssetMap returns the assets by id.
func (st StoryboardState) AssetMap() map[string]StoryboardAsset {
	out := make(map[string]StoryboardAsset, len(st.Assets))
	for _, a := range st.Assets {
		out[a.AssetID] = a
	}
	return out
}

// ShotContextFor builds the context of shot sh in unit unitID.
func (st StoryboardState) ShotContextFor(in StoryboardInput, unitID string, sh Shot) ShotContext {
	ui, found := in.Unit(unitID)
	ctx := ShotContext{UnitExists: found, Assets: st.AssetMap(), Frame: st.Frames[sh.ShotID]}
	if !found {
		return ctx
	}
	ctx.UnitRefs, ctx.Typography = ui.Unit.ScriptRefs, ui.Typography
	if spec, err := BuildFrameSpec(st.ProjectID, sh, ui, ctx.Assets, st.Kit); err == nil {
		ctx.SpecHash = SpecHash(spec)
	}
	return ctx
}

// SpecHashOf is the hash of the spec shot sh has now, which a frame drawn from it
// carries as its frame hash.
func (st StoryboardState) SpecHashOf(in StoryboardInput, unitID string, sh Shot) (string, error) {
	ui, ok := in.Unit(unitID)
	if !ok {
		return "", fmt.Errorf("unit %s không có trong kế hoạch hình đã chốt", unitID)
	}
	spec, err := BuildFrameSpec(st.ProjectID, sh, ui, st.AssetMap(), st.Kit)
	if err != nil {
		return "", err
	}
	return SpecHash(spec), nil
}

// unitOrder lists the unit ids of the state: the plan's units in plan order, then
// any stored unit the plan does not have in id order.
func (st StoryboardState) unitOrder(in StoryboardInput) []string {
	order := in.UnitIDs()
	known := map[string]bool{}
	for _, id := range order {
		known[id] = true
	}
	var extra []string
	for id := range st.Units {
		if !known[id] {
			extra = append(extra, id)
		}
	}
	sort.Strings(extra)
	return append(order, extra...)
}

// StoryboardChecks are the checks of the whole storyboard, worked out on every
// read and never stored: the six blocking m-shot checks, then the three warnings.
// Detail names what fails; a Creator's acceptance of a warning holds while Detail
// stays the same.
func StoryboardChecks(in StoryboardInput, st StoryboardState) []MachineCheck {
	byCheck := map[string][]string{}
	uses := map[string]int{}
	var uncovered []string
	for _, unitID := range st.unitOrder(in) {
		u := st.Units[unitID]
		if len(u.Shots) == 0 {
			if _, planned := in.planUnit(unitID); planned {
				uncovered = append(uncovered, unitID)
			}
			continue
		}
		covered := map[string]bool{}
		for _, sh := range u.Shots {
			for _, p := range CheckShot(sh, st.ShotContextFor(in, unitID, sh)) {
				byCheck[p.Check] = append(byCheck[p.Check], p.Detail)
			}
			for _, r := range sh.ScriptRefs {
				covered[r] = true
			}
			for _, id := range sh.AssetRefs {
				uses[id]++
			}
		}
		if pu, ok := in.planUnit(unitID); ok {
			var missing []string
			for _, r := range pu.ScriptRefs {
				if !covered[r] {
					missing = append(missing, r)
				}
			}
			if len(missing) > 0 {
				byCheck[CheckShotScript] = append(byCheck[CheckShotScript], fmt.Sprintf("%s: câu %s chưa có shot nào phủ", unitID, strings.Join(missing, ", ")))
			}
		}
	}
	var unused, single []string
	for _, a := range st.Assets {
		if a.Status != AssetApproved {
			continue
		}
		switch n := uses[a.AssetID]; {
		case n == 0:
			unused = append(unused, a.AssetID)
		case n < minAssetShots:
			single = append(single, a.AssetID)
		}
	}
	mk := func(id, label string) MachineCheck {
		d := byCheck[id]
		return MachineCheck{ID: id, Label: label, OK: len(d) == 0, Detail: strings.Join(d, "; "), Items: d}
	}
	warn := func(id, label string, list []string, detail func([]string) string) MachineCheck {
		c := MachineCheck{ID: id, Label: label, OK: len(list) == 0}
		if !c.OK {
			c.Detail = detail(list)
			c.Items = list
		}
		return c
	}
	return []MachineCheck{
		mk(CheckShotUnit, "Mỗi shot thuộc một unit của kế hoạch hình"),
		mk(CheckShotScript, "Shot chỉ phủ câu của unit và các shot phủ hết câu của unit"),
		mk(CheckShotAssets, "Shot chỉ dùng tài sản đã duyệt"),
		mk(CheckShotEnum, "Vị trí và góc máy thuộc danh sách cho phép"),
		mk(CheckShotText, "Chữ trên màn hình theo luật chữ đã chốt"),
		mk(CheckShotFrame, "Mỗi shot có ảnh khung và ảnh không cũ"),
		warn(CheckAssetUnused, "Mỗi tài sản đã duyệt được ít nhất một shot dùng", unused, func(l []string) string {
			return "chưa shot nào dùng: " + strings.Join(l, ", ")
		}),
		warn(CheckUnitCover, "Mỗi unit có ít nhất một shot", uncovered, func(l []string) string {
			return "chưa có shot: " + strings.Join(l, ", ")
		}),
		warn(CheckAssetCount, fmt.Sprintf("Mỗi tài sản dùng ở ít nhất %d shot", minAssetShots), single, func(l []string) string {
			return "chỉ một shot dùng: " + strings.Join(l, ", ")
		}),
	}
}

// ShotChecksBlocking returns the blocking problems of one shot as plain text, the
// reasons its approval is refused.
func ShotChecksBlocking(sh Shot, ctx ShotContext) []string {
	var out []string
	for _, p := range CheckShot(sh, ctx) {
		out = append(out, p.Detail)
	}
	return out
}
