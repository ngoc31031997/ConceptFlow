package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Unit kinds of step 11.
const (
	// UnitKindStoryboardAssets holds the list of asset ids and their approval
	// under ReviewUnitKey; what an asset is lives in rendering.
	UnitKindStoryboardAssets = "storyboard_assets"
	// UnitKindStoryboardUnit holds the shots of one plan unit; its unit_key is the
	// plan unit id.
	UnitKindStoryboardUnit = "storyboard_unit"
	// UnitKindStoryboardReview holds the review of the whole storyboard under
	// ReviewUnitKey.
	UnitKindStoryboardReview = "storyboard_review"
)

// Kinds of an asset: a figure is a character or object placed in a frame, a
// backdrop fills the frame behind them.
const (
	AssetKindFigure   = "figure"
	AssetKindBackdrop = "backdrop"
)

// AssetKinds lists every asset kind.
var AssetKinds = []string{AssetKindFigure, AssetKindBackdrop}

// Tiers of an asset: how much detail it is drawn with and how large it is placed.
const (
	AssetTierHero       = "hero"
	AssetTierSupport    = "support"
	AssetTierBackground = "background"
)

// AssetTiers lists every asset tier.
var AssetTiers = []string{AssetTierHero, AssetTierSupport, AssetTierBackground}

// Statuses of an asset as the Creator moves it through.
const (
	AssetPlanned  = "planned"
	AssetDrawn    = "drawn"
	AssetApproved = "approved"
)

// Positions of a subject in a frame. split_left and split_right are the halves of
// a split screen and are written only through NormalizeComposition.
const (
	PositionSplitLeft  = "split_left"
	PositionSplitRight = "split_right"
	// PositionSplitScreen is how the AI names a split frame; it is never stored.
	PositionSplitScreen = "split_screen"
)

// ShotPositions is the closed vocabulary of positions.
var ShotPositions = []string{
	"upper_left", "upper_center", "upper_right",
	"center_left", "center", "center_right",
	"lower_left", "lower_center", "lower_right",
	PositionSplitLeft, PositionSplitRight, "full",
}

// Camera views of a shot.
const (
	CameraWide       = "wide"
	CameraMedium     = "medium"
	CameraClose      = "close"
	CameraComparison = "comparison"
)

// ShotCameraViews is the closed vocabulary of camera views.
var ShotCameraViews = []string{CameraWide, CameraMedium, CameraClose, CameraComparison}

// ShotComposition says what is where in the frame. Subject and MainObject are an
// asset id or a free name (a character that is not an asset); the positions and
// the camera view come from the closed vocabularies. MainObject and its
// position are empty together.
type ShotComposition struct {
	Subject            string `json:"subject"`
	SubjectPosition    string `json:"subject_position"`
	MainObject         string `json:"main_object"`
	MainObjectPosition string `json:"main_object_position"`
	CameraView         string `json:"camera_view"`
}

// Shot is one frame of a unit. The unit and the beat are those of the unit that
// holds the shot, never written by the AI.
type Shot struct {
	ShotID        string          `json:"shot_id"`
	ScriptRefs    []string        `json:"script_refs"`
	ShotPurpose   string          `json:"shot_purpose"`
	Composition   ShotComposition `json:"composition"`
	ActionSummary string          `json:"action_summary"`
	OnScreenText  []string        `json:"on_screen_text"`
	AssetRefs     []string        `json:"asset_refs"`
	Notes         []string        `json:"notes"`
	Status        string          `json:"status"`
}

// UnitShots is the content of a storyboard_unit review unit: the shots of one
// plan unit in viewing order and the last drawing of each shot's frame. NextSeq
// is the number the next shot of the project takes; shot numbers are never reused.
type UnitShots struct {
	Shots   []Shot `json:"shots"`
	NextSeq int    `json:"next_seq"`
	// Frames is the last drawing rendering recorded for each shot, by shot id.
	Frames map[string]DrawnFrame `json:"frames,omitempty"`
}

// AssetEntry is one asset as the review unit lists it: its id and approval.
type AssetEntry struct {
	AssetID string `json:"asset_id"`
	Status  string `json:"status"`
}

// AssetList is the content of the storyboard_assets review unit.
type AssetList struct {
	Assets  []AssetEntry `json:"assets"`
	NextSeq int          `json:"next_seq"`
}

// StoryboardAsset is a project asset as rendering holds it, the reference
// visual-service reads to check shots, build frame specs and the locked
// document. Version grows with every redraw, so a shot's frame goes stale when an
// asset it uses is redrawn.
type StoryboardAsset struct {
	AssetID            string   `json:"asset_id"`
	ConceptID          string   `json:"concept_id"`
	Name               string   `json:"name"`
	Kind               string   `json:"kind"`
	Tier               string   `json:"tier"`
	Portrayal          string   `json:"portrayal"`
	MustNotImply       []string `json:"must_not_imply"`
	PaletteTokens      []string `json:"palette_tokens"`
	Expressions        []string `json:"expressions"`
	BaseIllustrationID string   `json:"base_illustration_id"`
	// Units are the plan units the asset appears in (VP ids), written by the
	// first phase and editable by the Creator.
	Units       []string `json:"units"`
	Version     int      `json:"version"`
	PreviewPath string   `json:"preview_path"`
	Status      string   `json:"status"`
}

// HasDetail reports whether the asset has its detail: a portrayal. Only then can
// it be drawn.
func (a StoryboardAsset) HasDetail() bool { return strings.TrimSpace(a.Portrayal) != "" }

// AssetEditableFields are the fields of an asset a review finding can point at.
var AssetEditableFields = []string{"portrayal", "must_not_imply", "palette_tokens", "expressions"}

// ShotEditableFields are the fields of a shot the Creator can write. The id comes
// from the machine, the unit from where the shot sits and the status from
// approving.
var ShotEditableFields = []string{"script_refs", "shot_purpose", "composition", "action_summary", "on_screen_text", "asset_refs", "notes"}

var (
	shotIDPattern  = regexp.MustCompile(`^SH-[0-9]{3,}$`)
	assetIDPattern = regexp.MustCompile(`^AST-[A-Z0-9_-]+-[0-9]{3,}$`)
	// assetNamePattern is a PascalCase component name.
	assetNamePattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
)

// ShotID is the id of the n-th shot of a project; numbers are never reused.
func ShotID(n int) string { return fmt.Sprintf("SH-%03d", n) }

// ShotSeq is the number inside a shot id (SH-007 is 7), or 0.
func ShotSeq(id string) int {
	if !shotIDPattern.MatchString(id) {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimPrefix(id, "SH-"))
	return n
}

// IsAssetID reports whether s has the shape of an asset id.
func IsAssetID(s string) bool { return assetIDPattern.MatchString(s) }

// NextShotSeq is the number the next shot of the project takes: past every
// counter the units stored and every shot id they hold. A counter never goes
// down, so a deleted shot's number is not given again.
func NextShotSeq(units map[string]UnitShots) int {
	next := 1
	for _, u := range units {
		if u.NextSeq > next {
			next = u.NextSeq
		}
		for _, sh := range u.Shots {
			if n := ShotSeq(sh.ShotID); n >= next {
				next = n + 1
			}
		}
	}
	return next
}

// NormalizeComposition turns split_screen into the two halves of a split frame:
// it needs exactly two objects, the subject and the main object, which take
// split_left and split_right. split_screen on a frame without two objects is an
// error; every other value is left for the vocabulary check.
func NormalizeComposition(c ShotComposition) (ShotComposition, error) {
	if c.SubjectPosition != PositionSplitScreen && c.MainObjectPosition != PositionSplitScreen {
		return c, nil
	}
	if strings.TrimSpace(c.Subject) == "" || strings.TrimSpace(c.MainObject) == "" {
		return c, fmt.Errorf("%s cần đúng hai đối tượng trong khung (subject và main_object)", PositionSplitScreen)
	}
	c.SubjectPosition = PositionSplitLeft
	c.MainObjectPosition = PositionSplitRight
	return c, nil
}

// DecodeUnitShots reads the stored shots of a unit. Empty content is a unit with
// no shot yet; an unknown field is an error.
func DecodeUnitShots(raw []byte) (UnitShots, error) {
	var s UnitShots
	if !isEmptyContent(raw) {
		if err := strictDecode(raw, &s); err != nil {
			return UnitShots{}, err
		}
	}
	for i := range s.Shots {
		s.Shots[i].normalize()
		if n := ShotSeq(s.Shots[i].ShotID); n >= s.NextSeq {
			s.NextSeq = n + 1
		}
	}
	if s.Shots == nil {
		s.Shots = []Shot{}
	}
	if s.NextSeq < 1 {
		s.NextSeq = 1
	}
	return s, nil
}

// DecodeShot reads one shot's JSON strictly. An unknown field or a value of the
// wrong type is ErrInvalidEdit.
func DecodeShot(raw []byte) (Shot, error) {
	var sh Shot
	if err := strictDecode(raw, &sh); err != nil {
		return Shot{}, err
	}
	sh.normalize()
	return sh, nil
}

// Encode returns the stored content of the unit.
func (s UnitShots) Encode() ([]byte, error) {
	shots := make([]Shot, len(s.Shots))
	copy(shots, s.Shots)
	for i := range shots {
		shots[i].normalize()
	}
	s.Shots = shots
	return json.Marshal(s)
}

// Index returns the position of shot id, or -1.
func (s UnitShots) Index(shotID string) int {
	for i, sh := range s.Shots {
		if sh.ShotID == shotID {
			return i
		}
	}
	return -1
}

// normalize turns lists a JSON null left nil into empty ones and gives a shot
// without status the pending one.
func (sh *Shot) normalize() {
	if sh.ScriptRefs == nil {
		sh.ScriptRefs = []string{}
	}
	if sh.OnScreenText == nil {
		sh.OnScreenText = []string{}
	}
	if sh.AssetRefs == nil {
		sh.AssetRefs = []string{}
	}
	if sh.Notes == nil {
		sh.Notes = []string{}
	}
	if sh.Status == "" {
		sh.Status = UnitPending
	}
}

// UsesAsset reports whether the shot names asset id in its asset references.
func (sh Shot) UsesAsset(id string) bool { return contains(sh.AssetRefs, id) }

// DecodeAssetList reads the stored asset list. Empty content is a video with no
// asset yet.
func DecodeAssetList(raw []byte) (AssetList, error) {
	var l AssetList
	if !isEmptyContent(raw) {
		if err := strictDecode(raw, &l); err != nil {
			return AssetList{}, err
		}
	}
	if l.Assets == nil {
		l.Assets = []AssetEntry{}
	}
	if l.NextSeq < 1 {
		l.NextSeq = 1
	}
	return l, nil
}

// Encode returns the stored content of the asset list.
func (l AssetList) Encode() ([]byte, error) {
	if l.Assets == nil {
		l.Assets = []AssetEntry{}
	}
	return json.Marshal(l)
}

// Index returns the position of asset id, or -1.
func (l AssetList) Index(assetID string) int {
	for i, a := range l.Assets {
		if a.AssetID == assetID {
			return i
		}
	}
	return -1
}

// AllApproved reports whether the list has assets and every one is approved.
func (l AssetList) AllApproved() bool {
	if len(l.Assets) == 0 {
		return false
	}
	for _, a := range l.Assets {
		if a.Status != UnitApproved {
			return false
		}
	}
	return true
}

// UnitApproval is the status of a unit worked out from its shots: approved only
// when it has shots and every one is approved.
func UnitApproval(u UnitShots) string {
	if len(u.Shots) == 0 {
		return UnitPending
	}
	for _, sh := range u.Shots {
		if sh.Status != UnitApproved {
			return UnitPending
		}
	}
	return UnitApproved
}

// BeatApproval is the status of a beat, or of the whole step, from its units:
// approved only when it has units and every one is approved.
func BeatApproval(units []UnitShots) string {
	if len(units) == 0 {
		return UnitPending
	}
	for _, u := range units {
		if UnitApproval(u) != UnitApproved {
			return UnitPending
		}
	}
	return UnitApproved
}

// ShotCreatorField is the path "SH-001.shot_purpose" a Creator-written field is
// stored under.
func ShotCreatorField(shotID, field string) string { return shotID + "." + field }

// SortedShotIDs returns the ids of every shot in units in number order.
func SortedShotIDs(units map[string]UnitShots) []string {
	var ids []string
	for _, u := range units {
		for _, sh := range u.Shots {
			ids = append(ids, sh.ShotID)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ShotSeq(ids[i]) < ShotSeq(ids[j]) })
	return ids
}
