package domain

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// DocumentAsset is one asset of a locked storyboard.
type DocumentAsset struct {
	AssetID     string `json:"asset_id"`
	ConceptID   string `json:"concept_id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Tier        string `json:"tier"`
	Portrayal   string `json:"portrayal"`
	PreviewPath string `json:"preview_path"`
}

// DocumentShot is one shot of a locked storyboard. FramePath is relative to the
// project's shared folder, e.g. storyboard/v1/SH-001.png.
type DocumentShot struct {
	ShotID        string          `json:"shot_id"`
	UnitID        string          `json:"unit_id"`
	BeatID        string          `json:"beat_id"`
	ScriptRefs    []string        `json:"script_refs"`
	FramePath     string          `json:"frame_path"`
	ShotPurpose   string          `json:"shot_purpose"`
	Composition   ShotComposition `json:"composition"`
	ActionSummary string          `json:"action_summary"`
	OnScreenText  []string        `json:"on_screen_text"`
	AssetRefs     []string        `json:"asset_refs"`
	Notes         []string        `json:"notes"`
}

// StoryboardDocument is a locked storyboard: the frozen document the later steps
// read.
type StoryboardDocument struct {
	SchemaVersion  string                   `json:"schema_version"`
	ProjectID      string                   `json:"project_id"`
	DocumentID     string                   `json:"document_id"`
	Status         string                   `json:"status"`
	Version        int                      `json:"version"`
	LockedAt       time.Time                `json:"locked_at"`
	SourceVersions StoryboardSourceVersions `json:"source_versions"`
	Assets         []DocumentAsset          `json:"assets"`
	Shots          []DocumentShot           `json:"shots"`
	Notes          []string                 `json:"notes"`
	Review         DocumentReview           `json:"review"`
}

// StoryboardFramePath is where a locked version keeps a shot's frame image,
// relative to the project's shared folder.
func StoryboardFramePath(version int, shotID string) string {
	return fmt.Sprintf("storyboard/v%d/%s.png", version, shotID)
}

// StoryboardLockBlockers lists what stops the storyboard from locking, empty when
// it can. The conditions: the sequence is locked and none of the locked steps
// under it moved on; every asset is approved; every unit has a shot; every shot is
// approved with a frame that is drawn and not stale; the blocking checks pass and
// each warning passes or was accepted; the storyboard was reviewed at least once
// and no finding is open.
func StoryboardLockBlockers(in StoryboardInput, st StoryboardState) []string {
	var out []string
	for _, why := range in.Outdated {
		out = append(out, "Bản chốt bước trước đã cũ: "+why)
	}
	var unapprovedAssets, noShot, unapprovedShots []string
	for _, a := range st.Assets {
		if a.Status != AssetApproved {
			unapprovedAssets = append(unapprovedAssets, a.AssetID)
		}
	}
	for _, unitID := range in.UnitIDs() {
		if len(st.Units[unitID].Shots) == 0 {
			noShot = append(noShot, unitID)
		}
	}
	for _, unitID := range st.unitOrder(in) {
		for _, sh := range st.Units[unitID].Shots {
			if sh.Status != UnitApproved {
				unapprovedShots = append(unapprovedShots, sh.ShotID)
			}
		}
	}
	out = appendGrouped(out, "tài sản chưa duyệt", "tài sản", unapprovedAssets)
	out = appendGrouped(out, "unit chưa có shot", "unit", noShot)
	out = appendGrouped(out, "shot chưa duyệt", "shot", unapprovedShots)
	for _, c := range StoryboardChecks(in, st) {
		if c.OK || c.ID == CheckUnitCover {
			// CheckUnitCover repeats the "unit chưa có shot" blocker above.
			continue
		}
		if _, accepted := st.Review.CheckAccepted(c); accepted && !IsBlockingCheck(c.ID) {
			continue
		}
		out = appendGrouped(out, c.Label, "mục", c.Items)
	}
	switch {
	case st.Review.Rounds < 1:
		out = append(out, "Chưa rà soát storyboard lần nào")
	case st.Review.OpenCount() > 0:
		out = append(out, fmt.Sprintf("Còn %d vấn đề rà soát chưa xử lý", st.Review.OpenCount()))
	}
	return out
}

// BuildStoryboardDocument assembles the locked document of the given version. It
// fails with ErrLockNotReady, naming every reason, when StoryboardLockBlockers is
// not empty. Shots are in the order of the sequence, then by shot id.
func BuildStoryboardDocument(projectID string, version int, lockedAt time.Time, in StoryboardInput, st StoryboardState) (StoryboardDocument, error) {
	if blockers := StoryboardLockBlockers(in, st); len(blockers) > 0 {
		return StoryboardDocument{}, fmt.Errorf("%w: %s", ErrLockNotReady, strings.Join(blockers, "; "))
	}
	assets := make([]DocumentAsset, 0, len(st.Assets))
	for _, a := range st.Assets {
		assets = append(assets, DocumentAsset{AssetID: a.AssetID, ConceptID: a.ConceptID, Name: a.Name, Kind: a.Kind,
			Tier: a.Tier, Portrayal: a.Portrayal, PreviewPath: a.PreviewPath})
	}
	shots := []DocumentShot{}
	for _, unitID := range in.UnitIDs() {
		pu, _ := in.planUnit(unitID)
		unitShots := append([]Shot{}, st.Units[unitID].Shots...)
		sort.SliceStable(unitShots, func(i, j int) bool { return ShotSeq(unitShots[i].ShotID) < ShotSeq(unitShots[j].ShotID) })
		for _, sh := range unitShots {
			shots = append(shots, DocumentShot{
				ShotID: sh.ShotID, UnitID: unitID, BeatID: pu.BeatID, ScriptRefs: nonNil(sh.ScriptRefs),
				FramePath: StoryboardFramePath(version, sh.ShotID), ShotPurpose: sh.ShotPurpose, Composition: sh.Composition,
				ActionSummary: sh.ActionSummary, OnScreenText: nonNil(sh.OnScreenText), AssetRefs: nonNil(sh.AssetRefs), Notes: nonNil(sh.Notes),
			})
		}
	}
	comments := []ReviewComment{}
	for _, f := range st.Review.Findings {
		comments = append(comments, ReviewComment{ID: f.ID, Group: f.Group, Text: f.Text, Decision: f.State, Why: f.Why})
	}
	for _, c := range StoryboardChecks(in, st) {
		if d, ok := st.Review.CheckAccepted(c); ok && !c.OK {
			comments = append(comments, ReviewComment{ID: c.ID, Group: "machine", Text: c.Label + ": " + c.Detail, Decision: FindingAccepted, Why: d.Why})
		}
	}
	return StoryboardDocument{
		SchemaVersion: SchemaVersion, ProjectID: projectID, DocumentID: fmt.Sprintf("storyboard-v%d", version), Status: "locked",
		Version: version, LockedAt: lockedAt, SourceVersions: in.SourceVersions(), Assets: assets, Shots: shots,
		Notes:  nonNil(st.Review.Notes),
		Review: DocumentReview{Reviewer: "creator", ApprovedAt: lockedAt, Comments: comments},
	}, nil
}

// SameStoryboardContent reports whether two documents hold the same storyboard:
// the same source versions, assets and shots. The version, the lock time, the
// frame paths (which carry the version) and the review do not count. A lock whose
// document is the same as the latest locked one is ErrNothingToLock.
func SameStoryboardContent(a, b StoryboardDocument) bool {
	return contentKey(a) == contentKey(b)
}

func contentKey(d StoryboardDocument) string {
	shots := make([]DocumentShot, len(d.Shots))
	copy(shots, d.Shots)
	for i := range shots {
		shots[i].FramePath = ""
	}
	raw, _ := json.Marshal(struct {
		Sources StoryboardSourceVersions `json:"sources"`
		Assets  []DocumentAsset          `json:"assets"`
		Shots   []DocumentShot           `json:"shots"`
	}{d.SourceVersions, d.Assets, shots})
	return string(raw)
}

// blockerSample is how many ids a grouped blocker names before "và k ... khác".
const blockerSample = 3

// appendGrouped adds one blocker for a whole kind of problem: the count, what is
// wrong and the first few ids, never one entry per item. noun names the items in
// the "và k <noun> khác" tail. Nothing is added for an empty list.
func appendGrouped(out []string, what, noun string, items []string) []string {
	if len(items) == 0 {
		return out
	}
	shown := items
	if len(shown) > blockerSample {
		shown = shown[:blockerSample]
	}
	list := strings.Join(shown, ", ")
	if more := len(items) - len(shown); more > 0 {
		list = fmt.Sprintf("%s và %d %s khác", list, more, noun)
	}
	if noun == "mục" {
		return append(out, fmt.Sprintf("%s: %d mục chưa đạt (%s)", what, len(items), list))
	}
	return append(out, fmt.Sprintf("%d %s (%s)", len(items), what, list))
}
