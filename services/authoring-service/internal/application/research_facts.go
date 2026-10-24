package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"authoring/internal/domain"
)

// The fields of a fact the Creator can edit by hand, and the names under which
// a hand-written field is remembered so that a pasted answer does not
// overwrite it.
const (
	factFieldQuestion    = "question"
	factFieldClaim       = "claim_vi"
	factFieldType        = "type"
	factFieldSourceTitle = "source_title"
	factFieldPublisher   = "publisher"
	factFieldYear        = "year"
	factFieldURL         = "url"
	factFieldQuote       = "quote"
	factFieldConfidence  = "confidence"
	factFieldNote        = "note"
)

// factTextFields are the fields a hand edit and a pasted answer both set,
// in the order a fact is written out.
var factTextFields = []string{
	factFieldQuestion, factFieldClaim, factFieldType, factFieldSourceTitle, factFieldPublisher,
	factFieldYear, factFieldURL, factFieldQuote, factFieldConfidence, factFieldNote,
}

// factField reads one of the fields above out of a fact.
func factField(f domain.Fact, name string) string {
	switch name {
	case factFieldQuestion:
		return f.Question
	case factFieldClaim:
		return f.ClaimVI
	case factFieldType:
		return string(f.Type)
	case factFieldSourceTitle:
		return f.SourceTitle
	case factFieldPublisher:
		return f.Publisher
	case factFieldYear:
		return f.Year
	case factFieldURL:
		return f.URL
	case factFieldQuote:
		return f.Quote
	case factFieldConfidence:
		return f.Confidence
	case factFieldNote:
		return f.Note
	}
	return ""
}

// withFactField returns f with one field set.
func withFactField(f domain.Fact, name, value string) domain.Fact {
	switch name {
	case factFieldQuestion:
		f.Question = value
	case factFieldClaim:
		f.ClaimVI = value
	case factFieldType:
		f.Type = domain.FactType(value)
	case factFieldSourceTitle:
		f.SourceTitle = value
	case factFieldPublisher:
		f.Publisher = value
	case factFieldYear:
		f.Year = value
	case factFieldURL:
		f.URL = value
	case factFieldQuote:
		f.Quote = value
	case factFieldConfidence:
		f.Confidence = value
	case factFieldNote:
		f.Note = value
	}
	return f
}

// FactInput is a fact the Creator adds by hand.
type FactInput struct {
	Question    string
	ClaimVI     string
	Type        domain.FactType
	SourceTitle string
	Publisher   string
	Year        string
	URL         string
	Quote       string
}

// FactFlags are the settings of a fact changed straight in the table; a nil
// field is left as it is.
type FactFlags struct {
	Type *domain.FactType
	Must *bool
	Keep *bool
}

// PasteResult is what reading a pasted answer did.
type PasteResult struct {
	Added    []string
	Updated  []string
	Rejected []domain.PasteRejected
	Log      PasteLog
}

// loadFact reads one stored fact.
func (uc *ResearchDeskUseCase) loadFact(ctx context.Context, projectID, id string) (domain.StoredFact, error) {
	return uc.units.GetResearchFact(ctx, projectID, id)
}

// saveFact stores a fact.
func (uc *ResearchDeskUseCase) saveFact(ctx context.Context, projectID string, fact domain.StoredFact) error {
	return uc.units.SaveResearchFact(ctx, projectID, fact)
}

// saveAndCheck stores a fact whose link or quote is new, marking it as being
// checked (or as having no source), and queues the check. Without a verifier a
// fact with a link says honestly that it could not be checked.
func (uc *ResearchDeskUseCase) saveAndCheck(ctx context.Context, projectID string, fact domain.StoredFact) error {
	switch {
	case fact.Fact.URL == "":
		fact.Fact.Link = domain.FactLink{State: domain.LinkNone}
	case uc.verifier == nil:
		fact.Fact.Link = domain.FactLink{State: domain.LinkUnreachable, Why: "kiểm link chưa được bật trên máy chủ"}
	default:
		fact.Fact.Link = domain.FactLink{State: domain.LinkRunning}
	}
	if err := uc.saveFact(ctx, projectID, fact); err != nil {
		return err
	}
	if fact.Fact.Link.State == domain.LinkRunning {
		uc.verifier.Enqueue(projectID, fact.ID, fact.Fact.URL, fact.Fact.Quote)
	}
	return nil
}

// reopenGate sends an approved fact table back to review: the Creator approved
// the table as it was.
func (uc *ResearchDeskUseCase) reopenGate(ctx context.Context, projectID string) error {
	unit, gate, err := uc.gate(ctx, projectID)
	if err != nil {
		return err
	}
	if unit.Status != domain.UnitApproved {
		return nil
	}
	unit.Status = domain.UnitPending
	_, err = uc.saveGate(ctx, unit, gate)
	return err
}

// mergePasted applies a pasted fact over a stored one: every text field the
// Creator did not write takes the pasted value. The settings the Creator made
// in the table (must, keep) and the link check are not touched here.
func mergePasted(stored domain.Fact, creatorFields []string, pasted domain.Fact) domain.Fact {
	mine := map[string]bool{}
	for _, f := range creatorFields {
		mine[f] = true
	}
	for _, name := range factTextFields {
		if !mine[name] {
			stored = withFactField(stored, name, factField(pasted, name))
		}
	}
	return stored
}

// IngestPaste reads a research tool's answer into the fact table: new codes are
// added, codes already there are updated except for what the Creator wrote or
// set, and rows that cannot be read are left out with their reasons. The paste
// is logged. A fact whose link or quote is new is queued for checking. An
// answer with no research JSON block reads nothing and is not logged.
func (uc *ResearchDeskUseCase) IngestPaste(ctx context.Context, projectID, raw string) (PasteResult, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return PasteResult{}, err
	}
	questions, err := uc.questions(ctx, projectID)
	if err != nil {
		return PasteResult{}, err
	}
	known := map[string]bool{}
	for _, q := range questions {
		known[q.ID] = true
	}
	parsed, err := domain.ParseResearchPaste(raw, func(id string) bool { return known[id] })
	if err != nil {
		return PasteResult{}, err
	}
	result := PasteResult{Added: []string{}, Updated: []string{}, Rejected: parsed.Rejected}
	if result.Rejected == nil {
		result.Rejected = []domain.PasteRejected{}
	}
	for _, claim := range parsed.Claims {
		stored, err := uc.loadFact(ctx, projectID, claim.ID)
		switch {
		case errors.Is(err, domain.ErrUnitNotFound):
			fresh := domain.StoredFact{ID: claim.ID, Fact: claim.Fact, CreatorFields: []string{}}
			if err := uc.saveAndCheck(ctx, projectID, fresh); err != nil {
				return PasteResult{}, err
			}
			result.Added = append(result.Added, claim.ID)
		case err != nil:
			return PasteResult{}, err
		default:
			merged := stored
			merged.Fact = mergePasted(stored.Fact, stored.CreatorFields, claim.Fact)
			sourceChanged := merged.Fact.URL != stored.Fact.URL || merged.Fact.Quote != stored.Fact.Quote
			merged.Fact.Link = stored.Fact.Link
			if sourceChanged {
				err = uc.saveAndCheck(ctx, projectID, merged)
			} else {
				err = uc.saveFact(ctx, projectID, merged)
			}
			if err != nil {
				return PasteResult{}, err
			}
			result.Updated = append(result.Updated, claim.ID)
		}
	}
	if len(result.Added) > 0 || len(result.Updated) > 0 {
		if err := uc.reopenGate(ctx, projectID); err != nil {
			return PasteResult{}, err
		}
	}
	if result.Log, err = uc.units.AddPaste(ctx, projectID, PasteLog{
		Added: result.Added, Updated: result.Updated, Rejected: result.Rejected,
	}); err != nil {
		return PasteResult{}, err
	}
	return result, nil
}

// AddFact adds a fact the Creator wrote. Every field they filled is theirs, so
// a later pasted answer does not overwrite it. A fact with a link is queued for
// checking.
func (uc *ResearchDeskUseCase) AddFact(ctx context.Context, projectID string, in FactInput) (FactView, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return FactView{}, err
	}
	in.ClaimVI = strings.TrimSpace(in.ClaimVI)
	in.URL = strings.TrimSpace(in.URL)
	if in.ClaimVI == "" {
		return FactView{}, fmt.Errorf("%w: dữ kiện không được trống", domain.ErrInvalidUnitEdit)
	}
	if !domain.ValidFactType(in.Type) {
		return FactView{}, fmt.Errorf("%w: loại dữ kiện %q không có", domain.ErrInvalidUnitEdit, in.Type)
	}
	if in.URL != "" && !domain.ValidSourceURL(in.URL) {
		return FactView{}, fmt.Errorf("%w: link không dùng được (cần http hoặc https, không kèm tài khoản)", domain.ErrInvalidUnitEdit)
	}
	if err := uc.requireQuestion(ctx, projectID, in.Question); err != nil {
		return FactView{}, err
	}
	existing, err := uc.units.ListResearchFacts(ctx, projectID)
	if err != nil {
		return FactView{}, err
	}
	ids := make([]string, 0, len(existing))
	for _, f := range existing {
		ids = append(ids, f.ID)
	}
	id := domain.NextFactID(ids)
	fact := domain.Fact{
		Question: in.Question, ClaimVI: in.ClaimVI, Type: in.Type,
		SourceTitle: strings.TrimSpace(in.SourceTitle), Publisher: strings.TrimSpace(in.Publisher), Year: strings.TrimSpace(in.Year),
		URL: in.URL, Quote: strings.TrimSpace(in.Quote), Keep: true,
	}
	mine := []string{factFieldQuestion, factFieldClaim, factFieldType}
	for _, name := range []string{factFieldSourceTitle, factFieldPublisher, factFieldYear, factFieldURL, factFieldQuote} {
		if factField(fact, name) != "" {
			mine = append(mine, name)
		}
	}
	if err := uc.saveAndCheck(ctx, projectID, domain.StoredFact{ID: id, Fact: fact, CreatorFields: mine}); err != nil {
		return FactView{}, err
	}
	if err := uc.reopenGate(ctx, projectID); err != nil {
		return FactView{}, err
	}
	return uc.factView(ctx, projectID, id)
}

// requireQuestion fails unless question names a research question that exists.
func (uc *ResearchDeskUseCase) requireQuestion(ctx context.Context, projectID, question string) error {
	if _, err := uc.units.GetUnit(ctx, projectID, domain.UnitKindResearchQuestion, question); err != nil {
		if errors.Is(err, domain.ErrUnitNotFound) {
			return fmt.Errorf("%w: câu hỏi %q không có", domain.ErrInvalidUnitEdit, question)
		}
		return err
	}
	return nil
}

// factView reads one fact as the screen shows it.
func (uc *ResearchDeskUseCase) factView(ctx context.Context, projectID, id string) (FactView, error) {
	stored, err := uc.loadFact(ctx, projectID, id)
	if err != nil {
		return FactView{}, err
	}
	mine := stored.CreatorFields
	if mine == nil {
		mine = []string{}
	}
	return FactView{ID: id, Fact: stored.Fact, Mine: mine}, nil
}

// SetFactFlags changes a fact's type, must-say mark or kept/dropped state in
// the table. A type the Creator set is theirs. The approved table goes back to
// review.
func (uc *ResearchDeskUseCase) SetFactFlags(ctx context.Context, projectID, id string, flags FactFlags) (FactView, error) {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return FactView{}, err
	}
	if flags.Type != nil && !domain.ValidFactType(*flags.Type) {
		return FactView{}, fmt.Errorf("%w: loại dữ kiện %q không có", domain.ErrInvalidUnitEdit, *flags.Type)
	}
	stored, err := uc.loadFact(ctx, projectID, id)
	if err != nil {
		return FactView{}, err
	}
	fact := &stored.Fact
	changed := false
	if flags.Type != nil && *flags.Type != fact.Type {
		fact.Type = *flags.Type
		if !stored.HasCreatorField(factFieldType) {
			stored.CreatorFields = append(stored.CreatorFields, factFieldType)
		}
		changed = true
	}
	if flags.Must != nil && *flags.Must != fact.Must {
		fact.Must = *flags.Must
		changed = true
	}
	if flags.Keep != nil && *flags.Keep != fact.Keep {
		fact.Keep = *flags.Keep
		changed = true
	}
	if !changed {
		return uc.factView(ctx, projectID, id)
	}
	if err := uc.saveFact(ctx, projectID, stored); err != nil {
		return FactView{}, err
	}
	if err := uc.reopenGate(ctx, projectID); err != nil {
		return FactView{}, err
	}
	return uc.factView(ctx, projectID, id)
}

// Verify checks the links of the given facts again (every kept fact with a link
// when ids is empty). A fact with no link is marked as having no source.
func (uc *ResearchDeskUseCase) Verify(ctx context.Context, projectID string, ids []string) error {
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	if len(ids) == 0 {
		facts, err := uc.facts(ctx, projectID)
		if err != nil {
			return err
		}
		for _, f := range facts {
			if f.Keep && f.URL != "" {
				ids = append(ids, f.ID)
			}
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		stored, err := uc.loadFact(ctx, projectID, id)
		if err != nil {
			return err
		}
		if err := uc.saveAndCheck(ctx, projectID, stored); err != nil {
			return err
		}
	}
	return nil
}

// factKind makes a fact editable through the generic unit API (UnitKind for
// domain.UnitKindFact).
type factKind struct{ desk *ResearchDeskUseCase }

// FactUnitKind returns the fact's UnitKind.
func (uc *ResearchDeskUseCase) FactUnitKind() UnitKind { return factKind{desk: uc} }

// Edit stores the Creator's changes to the text fields of a fact. A field they
// change becomes theirs. A new link or quote queues the check again; the
// approved table goes back to review.
func (k factKind) Edit(ctx context.Context, projectID, key string, fields map[string]string) (domain.ReviewUnit, error) {
	uc := k.desk
	if len(fields) == 0 {
		return domain.ReviewUnit{}, fmt.Errorf("%w: không có ô nào để lưu", domain.ErrInvalidUnitEdit)
	}
	for name := range fields {
		if name == factFieldType || !isFactTextField(name) {
			return domain.ReviewUnit{}, fmt.Errorf("%w: dữ kiện không sửa được ô %q ở đây", domain.ErrInvalidUnitEdit, name)
		}
	}
	stored, err := uc.loadFact(ctx, projectID, key)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	fact := stored.Fact
	before := fact
	for _, name := range factTextFields {
		value, ok := fields[name]
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch {
		case name == factFieldClaim && value == "":
			return domain.ReviewUnit{}, fmt.Errorf("%w: dữ kiện không được trống", domain.ErrInvalidUnitEdit)
		case name == factFieldURL && value != "" && !domain.ValidSourceURL(value):
			return domain.ReviewUnit{}, fmt.Errorf("%w: link không dùng được (cần http hoặc https, không kèm tài khoản)", domain.ErrInvalidUnitEdit)
		case name == factFieldQuestion:
			if err := uc.requireQuestion(ctx, projectID, value); err != nil {
				return domain.ReviewUnit{}, err
			}
		}
		if value == factField(fact, name) {
			continue
		}
		fact = withFactField(fact, name, value)
		if !stored.HasCreatorField(name) {
			stored.CreatorFields = append(stored.CreatorFields, name)
		}
	}
	if fact == before {
		return factUnit(projectID, stored)
	}
	stored.Fact = fact
	if fact.URL != before.URL || fact.Quote != before.Quote {
		err = uc.saveAndCheck(ctx, projectID, stored)
	} else {
		err = uc.saveFact(ctx, projectID, stored)
	}
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	if err := uc.reopenGate(ctx, projectID); err != nil {
		return domain.ReviewUnit{}, err
	}
	saved, err := uc.loadFact(ctx, projectID, key)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	return factUnit(projectID, saved)
}

// factUnit gives a stored fact the shape the generic unit API answers with.
func factUnit(projectID string, stored domain.StoredFact) (domain.ReviewUnit, error) {
	raw, err := json.Marshal(stored.Fact)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	return domain.ReviewUnit{
		ProjectID: projectID, Kind: domain.UnitKindFact, Key: stored.ID, Status: domain.UnitPending,
		Content: raw, CreatorFields: stored.CreatorFields,
	}, nil
}

func isFactTextField(name string) bool {
	for _, f := range factTextFields {
		if f == name {
			return true
		}
	}
	return false
}

// Redo is not offered for a fact: it comes from a research answer or from the
// Creator.
func (factKind) Redo(context.Context, string, string, string) (domain.ReviewUnit, error) {
	return domain.ReviewUnit{}, fmt.Errorf("%w: dữ kiện không làm lại bằng AI, hãy sửa tay hoặc dán kết quả mới", domain.ErrInvalidUnitEdit)
}

// Impact lists the work in later steps that uses the fact, as the registered
// finders report it; empty while no later step uses facts.
func (k factKind) Impact(ctx context.Context, projectID, key string) ([]string, error) {
	items := []string{}
	for _, find := range k.desk.dependents {
		found, err := find(ctx, projectID, key)
		if err != nil {
			return nil, err
		}
		items = append(items, found...)
	}
	return items, nil
}

// CheckApprovable accepts any fact; the table is what gets approved.
func (factKind) CheckApprovable(context.Context, domain.ReviewUnit) error { return nil }
