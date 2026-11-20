package domain

import (
	"strings"
)

// --- assets ------------------------------------------------------------------------

// assetReply is one asset as the AI writes it in the first phase: the frame of
// the list, no description. The id and the status are the machine's: the AI names
// an existing asset through same_as, and a reply with an asset_id key is refused.
type assetReply struct {
	SameAs             string   `json:"same_as"`
	ConceptID          string   `json:"concept_id"`
	Name               string   `json:"name"`
	Kind               string   `json:"kind"`
	Tier               string   `json:"tier"`
	BaseIllustrationID string   `json:"base_illustration_id"`
	Units              []string `json:"units"`
}

type assetsReply struct {
	Assets []assetReply `json:"assets"`
}

// AssetDraft is an asset the AI proposed in the first phase, checked, not yet
// stored. SameAs is the id of the asset it keeps or revises, empty for a new one.
type AssetDraft struct {
	SameAs             string   `json:"same_as"`
	ConceptID          string   `json:"concept_id"`
	Name               string   `json:"name"`
	Kind               string   `json:"kind"`
	Tier               string   `json:"tier"`
	BaseIllustrationID string   `json:"base_illustration_id"`
	Units              []string `json:"units"`
}

// AssetsReplyContext is what the asset list reply is checked against.
type AssetsReplyContext struct {
	// ConceptIDs are the concept ids of the locked visual concept's
	// representation rules.
	ConceptIDs map[string]bool
	// PlanUnits are the unit ids of the locked visual plan.
	PlanUnits map[string]bool
	// Library maps the id of every approved library drawing to its kind.
	Library map[string]string
	// Current are the ids of the assets the project has; InUse those among them
	// that shots use.
	Current map[string]bool
	InUse   map[string]bool
}

// ConceptIDs returns the concept ids of the representation rules: characters,
// objects, invisible things and scientific concepts.
func ConceptIDs(rr RepresentationRules) map[string]bool {
	out := map[string]bool{}
	for _, t := range rr.Characters.Items {
		out[t.ConceptID] = true
	}
	for _, t := range rr.Objects {
		out[t.ConceptID] = true
	}
	for _, t := range rr.Invisible {
		out[t.ConceptID] = true
	}
	for _, c := range rr.ScientificConcepts {
		out[c.ConceptID] = true
	}
	return out
}

// PaletteTokenSet returns the tokens of the palette.
func PaletteTokenSet(cs ColorSystem) map[string]bool {
	out := map[string]bool{}
	for _, s := range cs.Palette {
		out[s.Token] = true
	}
	return out
}

func nonNil(l []string) []string {
	if l == nil {
		return []string{}
	}
	return l
}

// ParseAssetsReply reads the AI's asset list. The reply is unusable, and nothing
// is stored, when it has an unknown key, no asset, a concept_id outside the
// representation rules (a backdrop may have none), a unit outside the plan or
// listed twice, a base illustration that is not an approved library drawing of
// the asset's kind, a kind or tier outside its list, a name that is not a
// PascalCase component name, a same_as naming no current asset or one twice, or
// when it drops an asset that shots use.
func ParseAssetsReply(raw []byte, ctx AssetsReplyContext) ([]AssetDraft, error) {
	var r assetsReply
	if err := decodeReply(raw, &r); err != nil {
		return nil, err
	}
	if len(r.Assets) == 0 {
		return nil, unreadable("không có tài sản nào")
	}
	out := make([]AssetDraft, 0, len(r.Assets))
	kept := map[string]bool{}
	for i, ra := range r.Assets {
		n := i + 1
		if !contains(AssetKinds, ra.Kind) {
			return nil, unreadable("tài sản %d có kind lạ «%s»", n, ra.Kind)
		}
		if !contains(AssetTiers, ra.Tier) {
			return nil, unreadable("tài sản %d có tier lạ «%s»", n, ra.Tier)
		}
		if !assetNamePattern.MatchString(ra.Name) {
			return nil, unreadable("tài sản %d có tên «%s» không phải tên thành phần PascalCase", n, ra.Name)
		}
		switch {
		case ra.ConceptID == "" && ra.Kind == AssetKindBackdrop:
		case !ctx.ConceptIDs[ra.ConceptID]:
			return nil, unreadable("tài sản %d (%s) có concept_id lạ «%s», không có trong quy tắc thể hiện", n, ra.Name, ra.ConceptID)
		}
		seenUnit := map[string]bool{}
		for _, u := range ra.Units {
			if !ctx.PlanUnits[u] {
				return nil, unreadable("tài sản %d (%s) có unit lạ «%s», không có trong kế hoạch hình", n, ra.Name, u)
			}
			if seenUnit[u] {
				return nil, unreadable("tài sản %d (%s) ghi unit «%s» hai lần", n, ra.Name, u)
			}
			seenUnit[u] = true
		}
		if ra.BaseIllustrationID != "" {
			kind, ok := ctx.Library[ra.BaseIllustrationID]
			if !ok {
				return nil, unreadable("tài sản %d (%s) lấy hình gốc «%s» không có trong thư viện đã duyệt", n, ra.Name, ra.BaseIllustrationID)
			}
			if kind != ra.Kind {
				return nil, unreadable("tài sản %d (%s) là %s nhưng hình gốc «%s» là %s", n, ra.Name, ra.Kind, ra.BaseIllustrationID, kind)
			}
		}
		if ra.SameAs != "" {
			if !ctx.Current[ra.SameAs] {
				return nil, unreadable("tài sản %d ghi same_as «%s», không có tài sản này", n, ra.SameAs)
			}
			if kept[ra.SameAs] {
				return nil, unreadable("tài sản %s được ghi hai lần", ra.SameAs)
			}
			kept[ra.SameAs] = true
		}
		out = append(out, AssetDraft{
			SameAs: ra.SameAs, ConceptID: ra.ConceptID, Name: ra.Name, Kind: ra.Kind, Tier: ra.Tier,
			BaseIllustrationID: ra.BaseIllustrationID, Units: nonNil(ra.Units),
		})
	}
	for id := range ctx.InUse {
		if ctx.Current[id] && !kept[id] {
			return nil, unreadable("bỏ mất %s, tài sản đang được shot dùng; giữ lại với same_as «%s»", id, id)
		}
	}
	return out, nil
}

// --- the detail of one asset -----------------------------------------------------

// AssetDetail is what the second phase writes for one asset: how it looks and
// feels, what it must not imply, its colours and the expressions it needs.
type AssetDetail struct {
	Portrayal     string   `json:"portrayal"`
	MustNotImply  []string `json:"must_not_imply"`
	PaletteTokens []string `json:"palette_tokens"`
	Expressions   []string `json:"expressions"`
}

type assetDetailReply struct {
	Detail *AssetDetail `json:"detail"`
}

// ParseAssetDetailReply reads the AI's detail of one asset. The reply is
// unusable, and nothing is stored, when it has an unknown key (at any depth),
// no detail object, an empty portrayal, or a palette token outside the palette.
func ParseAssetDetailReply(raw []byte, palette map[string]bool) (AssetDetail, error) {
	var r assetDetailReply
	if err := decodeReply(raw, &r); err != nil {
		return AssetDetail{}, err
	}
	if r.Detail == nil {
		return AssetDetail{}, unreadable("không có khoá detail")
	}
	d := *r.Detail
	if strings.TrimSpace(d.Portrayal) == "" {
		return AssetDetail{}, unreadable("detail thiếu portrayal")
	}
	for _, t := range d.PaletteTokens {
		if !palette[t] {
			return AssetDetail{}, unreadable("detail dùng token màu lạ «%s», không có trong bảng màu", t)
		}
	}
	d.MustNotImply, d.PaletteTokens, d.Expressions = nonNil(d.MustNotImply), nonNil(d.PaletteTokens), nonNil(d.Expressions)
	return d, nil
}

// --- the shots of one unit -------------------------------------------------------------

// shotReply is one shot as the AI writes it. The id, the unit, the beat and the
// status are the machine's.
type shotReply struct {
	ScriptRefs    []string        `json:"script_refs"`
	ShotPurpose   string          `json:"shot_purpose"`
	Composition   ShotComposition `json:"composition"`
	ActionSummary string          `json:"action_summary"`
	OnScreenText  []string        `json:"on_screen_text"`
	AssetRefs     []string        `json:"asset_refs"`
	Notes         []string        `json:"notes"`
}

type shotsReply struct {
	Shots []shotReply `json:"shots"`
}

// ShotsReplyContext is what the reply for one unit is checked against.
type ShotsReplyContext struct {
	// UnitRefs are the script line ids of the unit.
	UnitRefs []string
	// ApprovedAssets are the ids of the approved assets.
	ApprovedAssets map[string]bool
	// Typography holds the word limit of on-screen text.
	Typography Typography
}

// ParseShotsReply reads the AI's shots of one unit and returns them without ids,
// pending, with split_screen turned into the two halves of a split frame. The
// reply is unusable, and nothing is stored, when it has an unknown key or no
// shot; a shot covers no line, a line outside the unit, or leaves a line of the
// unit with no shot; names an asset that is not approved; has a position or
// camera view outside the vocabularies, a split_screen without exactly two
// objects, or an asset id as subject that asset_refs lacks; or has a line of text
// longer than the typography allows.
func ParseShotsReply(raw []byte, ctx ShotsReplyContext) ([]Shot, error) {
	var r shotsReply
	if err := decodeReply(raw, &r); err != nil {
		return nil, err
	}
	if len(r.Shots) == 0 {
		return nil, unreadable("không có shot nào")
	}
	covered := map[string]bool{}
	out := make([]Shot, 0, len(r.Shots))
	for i, rs := range r.Shots {
		n := i + 1
		if len(rs.ScriptRefs) == 0 {
			return nil, unreadable("shot %d không phủ câu nào", n)
		}
		for _, ref := range rs.ScriptRefs {
			if !contains(ctx.UnitRefs, ref) {
				return nil, unreadable("shot %d phủ câu «%s» không thuộc unit", n, ref)
			}
			covered[ref] = true
		}
		if strings.TrimSpace(rs.ShotPurpose) == "" || strings.TrimSpace(rs.ActionSummary) == "" {
			return nil, unreadable("shot %d thiếu shot_purpose hoặc action_summary", n)
		}
		for _, id := range rs.AssetRefs {
			if !ctx.ApprovedAssets[id] {
				return nil, unreadable("shot %d dùng tài sản «%s» không có trong danh sách đã duyệt", n, id)
			}
		}
		comp, err := NormalizeComposition(rs.Composition)
		if err != nil {
			return nil, unreadable("shot %d: %v", n, err)
		}
		if bad := compositionProblems(comp); len(bad) > 0 {
			return nil, unreadable("shot %d: %s", n, bad[0])
		}
		for _, name := range []string{comp.Subject, comp.MainObject} {
			if IsAssetID(name) && !contains(rs.AssetRefs, name) {
				return nil, unreadable("shot %d đặt %s vào bố cục nhưng asset_refs không có nó", n, name)
			}
		}
		for _, t := range rs.OnScreenText {
			if strings.TrimSpace(t) == "" {
				return nil, unreadable("shot %d có dòng chữ trống", n)
			}
			if max := ctx.Typography.Labels.MaxWords; max > 0 && len(strings.Fields(t)) > max {
				return nil, unreadable("shot %d có dòng chữ «%s» dài quá %d từ của luật chữ", n, t, max)
			}
		}
		sh := Shot{ScriptRefs: rs.ScriptRefs, ShotPurpose: rs.ShotPurpose, Composition: comp, ActionSummary: rs.ActionSummary,
			OnScreenText: rs.OnScreenText, AssetRefs: rs.AssetRefs, Notes: rs.Notes, Status: UnitPending}
		sh.normalize()
		out = append(out, sh)
	}
	var missing []string
	for _, ref := range ctx.UnitRefs {
		if !covered[ref] {
			missing = append(missing, ref)
		}
	}
	if len(missing) > 0 {
		return nil, unreadable("các shot không phủ câu %s của unit", strings.Join(missing, ", "))
	}
	return out, nil
}

// MergeShots puts the AI's shots in place of the unit's stored ones. The AI names
// no shot ids, so shots match by place: the n-th new shot takes the id and the
// fields the Creator wrote in the n-th stored one, and shots past the stored ones
// take the next numbers from nextSeq, the project's counter. A stored shot the
// new list has no place for goes, and its number is not given again; it is an
// error to drop one the Creator edited. Every shot goes back to review. It returns
// the merged unit with its counter and the Creator's fields that still stand.
func MergeShots(fresh []Shot, old UnitShots, creatorFields []string, nextSeq int) (UnitShots, []string, error) {
	if nextSeq < 1 {
		nextSeq = 1
	}
	out := UnitShots{Shots: make([]Shot, 0, len(fresh)), NextSeq: nextSeq}
	kept := []string{}
	for i, sh := range fresh {
		if i < len(old.Shots) {
			o := old.Shots[i]
			sh.ShotID = o.ShotID
			for _, f := range fieldsOfUnit(creatorFields, o.ShotID) {
				switch f {
				case "script_refs":
					sh.ScriptRefs = o.ScriptRefs
				case "shot_purpose":
					sh.ShotPurpose = o.ShotPurpose
				case "composition":
					sh.Composition = o.Composition
				case "action_summary":
					sh.ActionSummary = o.ActionSummary
				case "on_screen_text":
					sh.OnScreenText = o.OnScreenText
				case "asset_refs":
					sh.AssetRefs = o.AssetRefs
				case "notes":
					sh.Notes = o.Notes
				default:
					continue
				}
				kept = append(kept, ShotCreatorField(o.ShotID, f))
			}
		} else {
			sh.ShotID = ShotID(out.NextSeq)
			out.NextSeq++
		}
		sh.Status = UnitPending
		sh.normalize()
		out.Shots = append(out.Shots, sh)
	}
	for _, o := range old.Shots[min(len(fresh), len(old.Shots)):] {
		if len(fieldsOfUnit(creatorFields, o.ShotID)) > 0 {
			return UnitShots{}, nil, unreadable("bỏ mất %s, shot Creator đã sửa tay; viết đủ số shot để giữ lại nó", o.ShotID)
		}
	}
	for _, sh := range out.Shots {
		if n := ShotSeq(sh.ShotID); n >= out.NextSeq {
			out.NextSeq = n + 1
		}
	}
	if old.NextSeq > out.NextSeq {
		out.NextSeq = old.NextSeq
	}
	return out, kept, nil
}

// --- review --------------------------------------------------------------------------

// ParseStoryboardReviewReply reads the AI's review of the storyboard. A finding
// names a shot, a unit of the plan or an asset, and a field the Creator can write
// in a shot or an asset; a unit has no field of its own.
func ParseStoryboardReviewReply(raw []byte, prev ReviewState, shotIDs, unitIDs, assetIDs map[string]bool) (ReviewReply, error) {
	return parseReviewReply(raw, prev, func(f ReviewReplyItem) string {
		var fields []string
		switch {
		case shotIDs[f.UnitKey]:
			fields = ShotEditableFields
		case assetIDs[f.UnitKey]:
			fields = AssetEditableFields
		case unitIDs[f.UnitKey]:
		default:
			return "nói về shot, unit hay tài sản lạ «" + f.UnitKey + "»"
		}
		if f.Field != "" && !contains(fields, f.Field) {
			return "nói về trường lạ «" + f.Field + "»"
		}
		return ""
	})
}
