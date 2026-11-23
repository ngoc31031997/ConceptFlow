package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"visual/internal/domain"
)

// --- saving ----------------------------------------------------------------------

// saveUnit stores the shots of plan unit unitID. The row reads approved when
// every shot is. A unit stored for the first time takes the hashes its sources
// have now; rehash gives an existing unit those hashes too (the AI wrote it
// whole), otherwise only the hash of the assets it uses follows the shots.
func (d *StoryboardDesk) saveUnit(ctx context.Context, data *storyboardData, projectID, unitID string, u domain.UnitShots, creatorFields []string, rehash bool) error {
	raw, err := u.Encode()
	if err != nil {
		return err
	}
	stored, err := domain.DecodeUnitShots(raw)
	if err != nil {
		return err
	}
	hashes := data.in.UnitHashes(unitID, stored.Shots, data.assetMap())
	row, ok := data.rows[unitID]
	switch {
	case !ok:
		row = domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindStoryboardUnit, Key: unitID, InputHashes: hashes}
	case rehash:
		row.InputHashes = hashes
	default:
		merged := map[string]string{}
		for k, v := range row.InputHashes {
			merged[k] = v
		}
		merged[domain.StoryboardSourceAssets] = hashes[domain.StoryboardSourceAssets]
		row.InputHashes = merged
	}
	row.Content, row.Status = raw, domain.UnitApproval(stored)
	row.CreatorFields = append([]string{}, creatorFields...)
	if err := d.store.SaveUnit(ctx, row); err != nil {
		return err
	}
	data.rows[unitID], data.units[unitID] = row, stored
	return nil
}

// saveReview stores the review of the whole storyboard.
func (d *StoryboardDesk) saveReview(ctx context.Context, data *storyboardData) error {
	raw, err := json.Marshal(data.review)
	if err != nil {
		return err
	}
	row := data.reviewRow
	row.Content = raw
	return d.store.SaveUnit(ctx, row)
}

// saveAssetList stores the asset list's approval: the entries and the Creator's
// fields. The row reads approved when every asset of the video is approved in it.
func (d *StoryboardDesk) saveAssetList(ctx context.Context, data *storyboardData, entries []domain.AssetEntry, creatorFields []string) error {
	list := domain.AssetList{Assets: entries, NextSeq: data.list.NextSeq}
	raw, err := list.Encode()
	if err != nil {
		return err
	}
	row := data.listRow
	row.Content, row.CreatorFields, row.Status = raw, append([]string{}, creatorFields...), domain.UnitPending
	if list.AllApproved() {
		row.Status = domain.UnitApproved
	}
	if err := d.store.SaveUnit(ctx, row); err != nil {
		return err
	}
	data.listRow, data.list = row, list
	return nil
}

// listEntries are the entries of the asset list with asset id set to status and
// the others as they are; an entry of an asset rendering no longer has goes.
func (data *storyboardData) listEntries(id, status string) []domain.AssetEntry {
	out := []domain.AssetEntry{}
	have := map[string]bool{}
	for _, a := range data.assets {
		have[a.AssetID] = true
	}
	seen := false
	for _, e := range data.list.Assets {
		if !have[e.AssetID] {
			continue
		}
		if e.AssetID == id {
			e.Status, seen = status, true
		}
		out = append(out, e)
	}
	if !seen && have[id] {
		out = append(out, domain.AssetEntry{AssetID: id, Status: status})
	}
	return out
}

// --- frames ----------------------------------------------------------------------

// frameOf draws the frame of one shot from the spec it has now and returns the
// record to keep. It does not fail: a refusal of rendering, a video with no kit
// or a shot whose spec cannot be built is the frame's error, shown on the shot
// card, and never a drawing that did not happen. prev is the last record; a
// frame already drawn from the same spec is not drawn again. A drawing rendering
// reports as failed (503) leaves it no image, so the record drops its path; any
// other refusal keeps the last image.
func (d *StoryboardDesk) frameOf(ctx context.Context, data *storyboardData, projectID, unitID string, sh domain.Shot, prev domain.DrawnFrame) domain.DrawnFrame {
	fail := func(msg string, dropImage bool) domain.DrawnFrame {
		f := prev
		f.Error = msg
		if dropImage {
			f.FramePath, f.SpecHash, f.FrameHash = "", "", ""
		}
		return f
	}
	if err := data.needKit(); err != nil {
		return fail(err.Error(), false)
	}
	ui, ok := data.in.Unit(unitID)
	if !ok {
		return fail(domain.ErrUnknownPlanUnit.Error(), false)
	}
	spec, err := domain.BuildFrameSpec(projectID, sh, ui, data.assetMap(), data.kit)
	if err != nil {
		return fail(err.Error(), false)
	}
	hash := domain.SpecHash(spec)
	if prev.Drawn() && prev.SpecHash == hash && prev.Error == "" {
		return prev
	}
	res, err := d.rendering.DrawFrame(ctx, projectID, FrameRequest{FrameSpec: spec, UnitID: unitID, BeatID: ui.Unit.BeatID, SpecHash: hash})
	if err != nil {
		var re *RenderingError
		return fail(err.Error(), errors.As(err, &re) && re.Status == 503)
	}
	if res.SpecHash == "" {
		res.SpecHash = hash
	}
	return domain.DrawnFrame{FramePath: res.FramePath, SpecHash: res.SpecHash, FrameHash: res.FrameHash, Error: res.Error}
}

// drawFrames draws the frames of the given shots of a unit and stores each
// record as it comes. Only a failure to store stops it.
func (d *StoryboardDesk) drawFrames(ctx context.Context, data *storyboardData, projectID, unitID string, shotIDs ...string) error {
	for _, id := range shotIDs {
		u := data.units[unitID]
		i := u.Index(id)
		if i < 0 {
			continue
		}
		frames := map[string]domain.DrawnFrame{}
		for k, v := range u.Frames {
			frames[k] = v
		}
		frames[id] = d.frameOf(ctx, data, projectID, unitID, u.Shots[i], frames[id])
		u.Frames = frames
		if err := d.saveUnit(ctx, data, projectID, unitID, u, data.rows[unitID].CreatorFields, false); err != nil {
			return err
		}
	}
	return nil
}

// DrawFrame draws the frame of one shot again, from the shot as it is now.
func (d *StoryboardDesk) DrawFrame(ctx context.Context, projectID, unitID, shotID string) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if i := data.units[unitID].Index(shotID); i < 0 {
		return nil, domain.ErrUnknownShot
	}
	if err := data.needKit(); err != nil {
		return nil, err
	}
	if err := d.drawFrames(ctx, data, projectID, unitID, shotID); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// --- shots -----------------------------------------------------------------------

func (data *storyboardData) planUnit(unitID string) error {
	if _, ok := data.in.Unit(unitID); !ok {
		return domain.ErrUnknownPlanUnit
	}
	return nil
}

// readShot reads a shot from the Creator's fields over base. The fields become
// the Creator's and the shot goes back to review.
func readShot(base domain.Shot, fields map[string]json.RawMessage) (domain.Shot, []string, error) {
	names, err := editableNames(fields, domain.ShotEditableFields, "shot")
	if err != nil {
		return domain.Shot{}, nil, err
	}
	merged, err := overlay(base, fields, names)
	if err != nil {
		return domain.Shot{}, nil, err
	}
	sh, err := domain.DecodeShot(merged)
	if err != nil {
		return domain.Shot{}, nil, err
	}
	if sh.Composition, err = domain.NormalizeComposition(sh.Composition); err != nil {
		return domain.Shot{}, nil, fmt.Errorf("%w: %v", domain.ErrInvalidEdit, err)
	}
	sh.ShotID, sh.Status = base.ShotID, domain.UnitPending
	return sh, names, nil
}

func shotFieldPaths(shotID string, names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, domain.ShotCreatorField(shotID, n))
	}
	return out
}

// EditShot writes the Creator's fields into a shot, which goes back to review,
// and draws its frame again when the look changed. A drawing that fails is the
// frame's error; the shot is saved anyway.
func (d *StoryboardDesk) EditShot(ctx context.Context, projectID, unitID, shotID string, fields map[string]json.RawMessage) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := data.planUnit(unitID); err != nil {
		return nil, err
	}
	u := data.units[unitID]
	at := u.Index(shotID)
	if at < 0 {
		return nil, domain.ErrUnknownShot
	}
	sh, names, err := readShot(u.Shots[at], fields)
	if err != nil {
		return nil, err
	}
	shots := append([]domain.Shot{}, u.Shots...)
	shots[at] = sh
	u.Shots = shots
	row := data.rows[unitID]
	if err := d.saveUnit(ctx, data, projectID, unitID, u, row.AddCreatorFields(shotFieldPaths(shotID, names)...), false); err != nil {
		return nil, err
	}
	if err := d.drawFrames(ctx, data, projectID, unitID, shotID); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// AddShot adds a shot the Creator wrote at the end of a unit; it takes the next
// shot number of the video and every field it was given becomes the Creator's.
func (d *StoryboardDesk) AddShot(ctx context.Context, projectID, unitID string, fields map[string]json.RawMessage) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := data.planUnit(unitID); err != nil {
		return nil, err
	}
	n := domain.NextShotSeq(data.units)
	base := domain.Shot{ShotID: domain.ShotID(n), ScriptRefs: []string{}, OnScreenText: []string{}, AssetRefs: []string{}, Notes: []string{}, Status: domain.UnitPending}
	sh, names, err := readShot(base, fields)
	if err != nil {
		return nil, err
	}
	u := data.units[unitID]
	u.Shots = append(append([]domain.Shot{}, u.Shots...), sh)
	u.NextSeq = n + 1
	row := data.rows[unitID]
	if err := d.saveUnit(ctx, data, projectID, unitID, u, row.AddCreatorFields(shotFieldPaths(sh.ShotID, names)...), false); err != nil {
		return nil, err
	}
	if err := d.drawFrames(ctx, data, projectID, unitID, sh.ShotID); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// DeleteShot removes a shot from its unit; the last shot of a unit stays, and
// the number of a deleted shot is not used again.
func (d *StoryboardDesk) DeleteShot(ctx context.Context, projectID, unitID, shotID string) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	u := data.units[unitID]
	at := u.Index(shotID)
	if at < 0 {
		return nil, domain.ErrUnknownShot
	}
	if len(u.Shots) == 1 {
		return nil, domain.ErrLastShot
	}
	u.Shots = append(append([]domain.Shot{}, u.Shots[:at]...), u.Shots[at+1:]...)
	frames := map[string]domain.DrawnFrame{}
	for k, v := range u.Frames {
		if k != shotID {
			frames[k] = v
		}
	}
	u.Frames = frames
	kept := []string{}
	for _, f := range data.rows[unitID].CreatorFields {
		if !strings.HasPrefix(f, shotID+".") {
			kept = append(kept, f)
		}
	}
	if err := d.saveUnit(ctx, data, projectID, unitID, u, kept, false); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// setShotStatus approves or reopens a shot. Approving needs a shot that passes
// every blocking check, its frame drawn and not stale included.
func (d *StoryboardDesk) setShotStatus(ctx context.Context, projectID, unitID, shotID, status string) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	u := data.units[unitID]
	at := u.Index(shotID)
	if at < 0 {
		return nil, domain.ErrUnknownShot
	}
	sh := u.Shots[at]
	if status == domain.UnitApproved {
		if p := domain.ShotChecksBlocking(sh, data.checkState(projectID).ShotContextFor(data.in, unitID, sh)); len(p) > 0 {
			return nil, fmt.Errorf("%w: %s", domain.ErrUnitInvalid, strings.Join(p, "; "))
		}
	}
	shots := append([]domain.Shot{}, u.Shots...)
	shots[at].Status = status
	u.Shots = shots
	if err := d.saveUnit(ctx, data, projectID, unitID, u, data.rows[unitID].CreatorFields, false); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// ApproveShot approves one shot.
func (d *StoryboardDesk) ApproveShot(ctx context.Context, projectID, unitID, shotID string) (*StoryboardScreen, error) {
	return d.setShotStatus(ctx, projectID, unitID, shotID, domain.UnitApproved)
}

// UnapproveShot puts one shot back to review.
func (d *StoryboardDesk) UnapproveShot(ctx context.Context, projectID, unitID, shotID string) (*StoryboardScreen, error) {
	return d.setShotStatus(ctx, projectID, unitID, shotID, domain.UnitPending)
}

// ApproveAll approves every pending shot that passes its blocking checks; a
// shot with a problem or a stale frame stays.
func (d *StoryboardDesk) ApproveAll(ctx context.Context, projectID string) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, unitID := range data.in.UnitIDs() {
		u := data.units[unitID]
		shots := append([]domain.Shot{}, u.Shots...)
		changed := false
		for i, sh := range shots {
			if sh.Status == domain.UnitApproved {
				continue
			}
			if len(domain.ShotChecksBlocking(sh, data.checkState(projectID).ShotContextFor(data.in, unitID, sh))) == 0 {
				shots[i].Status, changed = domain.UnitApproved, true
			}
		}
		if !changed {
			continue
		}
		u.Shots = shots
		if err := d.saveUnit(ctx, data, projectID, unitID, u, data.rows[unitID].CreatorFields, false); err != nil {
			return nil, err
		}
	}
	return d.State(ctx, projectID)
}

// --- assets ----------------------------------------------------------------------

// assetInputFields are the fields of an asset the Creator can write.
var assetInputFields = []string{"concept_id", "name", "kind", "tier", "portrayal", "must_not_imply", "palette_tokens", "expressions",
	"base_illustration_id", "units", "to_library", "library_folder_id"}

// assetCreatorFields are those of them an AI rewrite of the list keeps.
func assetCreatorFields(names []string) []string {
	var out []string
	for _, n := range names {
		if n != "to_library" && n != "library_folder_id" {
			out = append(out, n)
		}
	}
	return out
}

// readAssetInput reads an asset from the Creator's fields over base.
func readAssetInput(base AssetInput, fields map[string]json.RawMessage) (AssetInput, []string, error) {
	names, err := editableNames(fields, assetInputFields, "tài sản")
	if err != nil {
		return AssetInput{}, nil, err
	}
	merged, err := overlay(base, fields, names)
	if err != nil {
		return AssetInput{}, nil, err
	}
	in, err := decodeAssetInput(merged)
	if err != nil {
		return AssetInput{}, nil, err
	}
	return in, names, nil
}

func decodeAssetInput(raw []byte) (AssetInput, error) {
	var in AssetInput
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return AssetInput{}, fmt.Errorf("%w: %v", domain.ErrInvalidEdit, err)
	}
	in.MustNotImply, in.PaletteTokens, in.Expressions, in.Units = nonNilList(in.MustNotImply), nonNilList(in.PaletteTokens), nonNilList(in.Expressions), nonNilList(in.Units)
	return in, nil
}

// AddAsset has rendering create an asset the Creator wrote; it starts planned.
func (d *StoryboardDesk) AddAsset(ctx context.Context, projectID string, fields map[string]json.RawMessage) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	in, names, err := readAssetInput(AssetInput{Kind: domain.AssetKindFigure, Tier: domain.AssetTierSupport, ToLibrary: true,
		MustNotImply: []string{}, PaletteTokens: []string{}, Expressions: []string{}, Units: []string{}}, fields)
	if err != nil {
		return nil, err
	}
	created, err := d.rendering.CreateAsset(ctx, projectID, in)
	if err != nil {
		return nil, err
	}
	data.assets = append(data.assets, created)
	if err := d.saveAssetList(ctx, data, data.listEntries(created.AssetID, domain.UnitPending), data.listRow.AddCreatorFields(shotFieldPaths(created.AssetID, assetCreatorFields(names))...)); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// EditAsset writes the Creator's fields into an asset at rendering. The asset
// leaves the approved list; a change to what it looks like sends it back to
// planned at rendering.
func (d *StoryboardDesk) EditAsset(ctx context.Context, projectID, assetID string, fields map[string]json.RawMessage) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	cur, ok := data.findAsset(assetID)
	if !ok {
		return nil, domain.ErrUnknownAsset
	}
	in, names, err := readAssetInput(AssetInputOf(cur), fields)
	if err != nil {
		return nil, err
	}
	if _, err := d.rendering.UpdateAsset(ctx, projectID, assetID, in); err != nil {
		return nil, err
	}
	if err := d.saveAssetList(ctx, data, data.listEntries(assetID, domain.UnitPending), data.listRow.AddCreatorFields(shotFieldPaths(assetID, assetCreatorFields(names))...)); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// DeleteAsset deletes an asset at rendering. An asset a shot uses is refused:
// rendering cannot know which shots use it.
func (d *StoryboardDesk) DeleteAsset(ctx context.Context, projectID, assetID string) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if _, ok := data.findAsset(assetID); !ok {
		return nil, domain.ErrUnknownAsset
	}
	if users := data.assetUsers()[assetID]; len(users) > 0 {
		return nil, fmt.Errorf("%w: %s dùng %s", domain.ErrAssetInUse, strings.Join(users, ", "), assetID)
	}
	if err := d.rendering.DeleteAsset(ctx, projectID, assetID); err != nil {
		return nil, err
	}
	kept := []string{}
	for _, f := range data.listRow.CreatorFields {
		if !strings.HasPrefix(f, assetID+".") {
			kept = append(kept, f)
		}
	}
	rest := []ProjectAsset{}
	for _, a := range data.assets {
		if a.AssetID != assetID {
			rest = append(rest, a)
		}
	}
	data.assets = rest
	if err := d.saveAssetList(ctx, data, data.listEntries("", domain.UnitPending), kept); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// drawRequest is what rendering draws an asset with. A video with no kit cannot
// draw, and nothing is drawn before the Creator approved the asset list, so the
// drawings follow a list the Creator agreed to.
func (data *storyboardData) drawRequest(note string) (AssetDrawRequest, error) {
	if err := data.needKit(); err != nil {
		return AssetDrawRequest{}, err
	}
	if !data.list.AllApproved() {
		return AssetDrawRequest{}, domain.ErrAssetListNotApproved
	}
	return AssetDrawRequest{Kit: data.kit, RefID: data.refID, Note: strings.TrimSpace(note)}, nil
}

// assetAction runs one call on an existing asset and returns the screen.
func (d *StoryboardDesk) assetAction(ctx context.Context, projectID, assetID string, call func(data *storyboardData) (ProjectAsset, error)) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if _, ok := data.findAsset(assetID); !ok {
		return nil, domain.ErrUnknownAsset
	}
	if _, err := call(data); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// DrawAsset has rendering draw an asset from its description. It runs on the
// system's drawing AI; there is no AI-outside mode. An asset without its detail,
// or of a list the Creator has not approved, is not drawn.
func (d *StoryboardDesk) DrawAsset(ctx context.Context, projectID, assetID string) (*StoryboardScreen, error) {
	return d.assetAction(ctx, projectID, assetID, func(data *storyboardData) (ProjectAsset, error) {
		if cur, _ := data.findAsset(assetID); !cur.HasDetail() {
			return ProjectAsset{}, fmt.Errorf("%w: %s", domain.ErrAssetNoDetail, assetID)
		}
		req, err := data.drawRequest("")
		if err != nil {
			return ProjectAsset{}, err
		}
		return d.rendering.DrawAsset(ctx, projectID, assetID, req)
	})
}

// RedrawAsset has rendering draw an asset again from the code it has, with the
// Creator's note, once the asset list is approved. The shots that use it have an old frame until they are drawn
// again.
func (d *StoryboardDesk) RedrawAsset(ctx context.Context, projectID, assetID, note string) (*StoryboardScreen, error) {
	return d.assetAction(ctx, projectID, assetID, func(data *storyboardData) (ProjectAsset, error) {
		req, err := data.drawRequest(note)
		if err != nil {
			return ProjectAsset{}, err
		}
		return d.rendering.RedrawAsset(ctx, projectID, assetID, req)
	})
}

// ApproveAsset approves a drawn asset; rendering adds it to the library when the
// asset asks to.
func (d *StoryboardDesk) ApproveAsset(ctx context.Context, projectID, assetID, projectName string) (*StoryboardScreen, error) {
	return d.assetAction(ctx, projectID, assetID, func(data *storyboardData) (ProjectAsset, error) {
		return d.rendering.ApproveAsset(ctx, projectID, assetID, AssetApproveRequest{RefID: data.refID, ProjectName: strings.TrimSpace(projectName)})
	})
}

// AssetPreview returns the PNG of an asset's current drawing.
func (d *StoryboardDesk) AssetPreview(ctx context.Context, projectID, assetID string) ([]byte, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, ErrNoProjectID
	}
	return d.rendering.AssetPreview(ctx, projectID, assetID)
}

// UnapproveAsset puts an asset back to drawn.
func (d *StoryboardDesk) UnapproveAsset(ctx context.Context, projectID, assetID string) (*StoryboardScreen, error) {
	return d.assetAction(ctx, projectID, assetID, func(*storyboardData) (ProjectAsset, error) {
		return d.rendering.UnapproveAsset(ctx, projectID, assetID)
	})
}

// ApproveAssets approves the asset list as it is. Every asset must have its
// detail, since approving the list opens the drawing of each one; it fails with
// ErrAssetNoDetail naming the assets that wait for it.
func (d *StoryboardDesk) ApproveAssets(ctx context.Context, projectID string) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(data.assets) == 0 {
		return nil, fmt.Errorf("%w: chưa có tài sản nào", domain.ErrUnitNotWritten)
	}
	var noDetail []string
	for _, a := range data.assets {
		if !a.HasDetail() {
			noDetail = append(noDetail, a.AssetID)
		}
	}
	if len(noDetail) > 0 {
		return nil, fmt.Errorf("%w: %s", domain.ErrAssetNoDetail, strings.Join(noDetail, ", "))
	}
	entries := make([]domain.AssetEntry, 0, len(data.assets))
	for _, a := range data.assets {
		entries = append(entries, domain.AssetEntry{AssetID: a.AssetID, Status: domain.UnitApproved})
	}
	if err := d.saveAssetList(ctx, data, entries, data.listRow.CreatorFields); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// UnapproveAssets puts the asset list back to review.
func (d *StoryboardDesk) UnapproveAssets(ctx context.Context, projectID string) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	entries := make([]domain.AssetEntry, 0, len(data.assets))
	for _, a := range data.assets {
		entries = append(entries, domain.AssetEntry{AssetID: a.AssetID, Status: domain.UnitPending})
	}
	if err := d.saveAssetList(ctx, data, entries, data.listRow.CreatorFields); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

// --- review findings ------------------------------------------------------------

// ApplyFinding writes a finding's fix into its field of the shot or asset it
// names; the field becomes the Creator's, the shot goes back to review and its
// frame is drawn again when the look changed.
func (d *StoryboardDesk) ApplyFinding(ctx context.Context, projectID, findingID string) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	i := data.review.Find(findingID)
	if i < 0 {
		return nil, domain.ErrUnknownFinding
	}
	f := data.review.Findings[i]
	if f.State != domain.FindingOpen || f.Field == "" || strings.TrimSpace(f.Fix) == "" {
		return nil, domain.ErrFindingNotApplicable
	}
	if unitID, at, ok := data.shotUnit(f.UnitKey); ok {
		err = d.applyToShot(ctx, data, projectID, unitID, at, f)
	} else if cur, ok := data.findAsset(f.UnitKey); ok {
		err = d.applyToAsset(ctx, data, projectID, cur, f)
	} else {
		err = domain.ErrFindingNotApplicable
	}
	if err != nil {
		return nil, err
	}
	data.review.Findings[i].State = domain.FindingApplied
	if err := d.saveReview(ctx, data); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}

func (d *StoryboardDesk) applyToShot(ctx context.Context, data *storyboardData, projectID, unitID string, at int, f domain.Finding) error {
	if !containsStr(domain.ShotEditableFields, f.Field) {
		return domain.ErrFindingNotApplicable
	}
	u := data.units[unitID]
	cur, err := json.Marshal(u.Shots[at])
	if err != nil {
		return err
	}
	fixed, err := domain.ApplyFix(cur, f.Field, f.Fix, f.Replaces)
	if err != nil {
		return err
	}
	sh, err := domain.DecodeShot(fixed)
	if err != nil {
		return domain.ErrFindingNotApplicable
	}
	sh.ShotID, sh.Status = u.Shots[at].ShotID, domain.UnitPending
	shots := append([]domain.Shot{}, u.Shots...)
	shots[at] = sh
	u.Shots = shots
	if err := d.saveUnit(ctx, data, projectID, unitID, u, data.rows[unitID].AddCreatorFields(domain.ShotCreatorField(sh.ShotID, f.Field)), false); err != nil {
		return err
	}
	return d.drawFrames(ctx, data, projectID, unitID, sh.ShotID)
}

func (d *StoryboardDesk) applyToAsset(ctx context.Context, data *storyboardData, projectID string, cur ProjectAsset, f domain.Finding) error {
	if !containsStr(domain.AssetEditableFields, f.Field) {
		return domain.ErrFindingNotApplicable
	}
	base, err := json.Marshal(AssetInputOf(cur))
	if err != nil {
		return err
	}
	fixed, err := domain.ApplyFix(base, f.Field, f.Fix, f.Replaces)
	if err != nil {
		return err
	}
	in, err := decodeAssetInput(fixed)
	if err != nil {
		return domain.ErrFindingNotApplicable
	}
	if _, err := d.rendering.UpdateAsset(ctx, projectID, cur.AssetID, in); err != nil {
		return err
	}
	return d.saveAssetList(ctx, data, data.listEntries(cur.AssetID, domain.UnitPending), data.listRow.AddCreatorFields(domain.ShotCreatorField(cur.AssetID, f.Field)))
}

// AcceptFinding accepts a finding, or a failing machine check, with the
// Creator's reason. A machine check accepted opens again when what fails changes.
func (d *StoryboardDesk) AcceptFinding(ctx context.Context, projectID, findingID, why string) (*StoryboardScreen, error) {
	data, err := d.load(ctx, projectID)
	if err != nil {
		return nil, err
	}
	why = strings.TrimSpace(why)
	if why == "" {
		why = "không ghi lý do"
	}
	if strings.HasPrefix(findingID, "m-") {
		if domain.IsBlockingCheck(findingID) {
			return nil, domain.ErrFindingNotApplicable
		}
		if err := acceptCheck(&data.review, domain.StoryboardChecks(data.in, data.checkState(projectID)), findingID, why); err != nil {
			return nil, err
		}
	} else {
		i := data.review.Find(findingID)
		if i < 0 {
			return nil, domain.ErrUnknownFinding
		}
		if data.review.Findings[i].State != domain.FindingOpen {
			return nil, domain.ErrFindingNotApplicable
		}
		data.review.Findings[i].State, data.review.Findings[i].Why = domain.FindingAccepted, why
	}
	if err := d.saveReview(ctx, data); err != nil {
		return nil, err
	}
	return d.State(ctx, projectID)
}
