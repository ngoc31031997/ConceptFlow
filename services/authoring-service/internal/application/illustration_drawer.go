package application

import (
	"context"
	_ "embed"
	"fmt"
	"regexp"
	"strings"
	"time"

	"authoring/internal/domain"
)

// CR-044 — the AI drawer: one model call per drawing, held to the channel's
// style rules, checked by the renderer, and sent back with the renderer's
// line-numbered errors until it passes (at most maxDrawAttempts calls).

const (
	maxDrawAttempts   = 3
	maxDrawReferences = 5 // 3 exemplars + up to 2 of the Creator's approved drawings
	drawerRole        = "illustration_drawer"
)

//go:embed drawer_prompt_vi.txt
var drawerPromptVI string

// DrawRequest asks the AI drawer for a new drawing.
type DrawRequest struct {
	Description string // what to draw, in the Creator's words
	FolderID    string
	Name        string // optional; the model proposes one otherwise
	Model       string // optional model override
}

type illustrationDrawer struct {
	llm       LLMProviderPort
	recorder  *LLMUsageRecorder
	maxTokens int
}

// WithDrawer enables Draw and Redraw.
func (uc *IllustrationsUseCase) WithDrawer(llm LLMProviderPort, recorder *LLMUsageRecorder, maxTokens int) *IllustrationsUseCase {
	uc.drawer = &illustrationDrawer{llm: llm, recorder: recorder, maxTokens: maxTokens}
	return uc
}

// ErrDrawerDisabled is returned when no model is configured.
var ErrDrawerDisabled = fmt.Errorf("AI vẽ chưa được bật (thiếu kết nối model)")

// drawnReply is the parsed answer of the model.
type drawnReply struct {
	Name, Title, Tags, Description, Usage, Code string
}

var (
	replyFieldRe = regexp.MustCompile(`(?m)^(TÊN|TÊN HIỂN THỊ|THẺ|MÔ TẢ|CÁCH GỌI):\s*(.+)$`)
	replyCodeRe  = regexp.MustCompile("(?s)```(?:tsx|typescript|ts|jsx)?\\s*\\n(.*?)```")
	exportNameRe = regexp.MustCompile(`(?m)^export function ([A-Z][A-Za-z0-9]*)\s*\(`)
)

func parseDrawnReply(text string) (drawnReply, error) {
	var r drawnReply
	for _, m := range replyFieldRe.FindAllStringSubmatch(text, -1) {
		v := strings.TrimSpace(m[2])
		switch m[1] {
		case "TÊN":
			r.Name = v
		case "TÊN HIỂN THỊ":
			r.Title = v
		case "THẺ":
			r.Tags = v
		case "MÔ TẢ":
			r.Description = v
		case "CÁCH GỌI":
			r.Usage = v
		}
	}
	code := replyCodeRe.FindStringSubmatch(text)
	if code == nil {
		return r, fmt.Errorf("câu trả lời không có khối ```tsx")
	}
	r.Code = strings.TrimSpace(code[1]) + "\n"
	// The exported function is the truth; a TÊN line that disagrees is ignored.
	if m := exportNameRe.FindStringSubmatch(r.Code); m != nil {
		r.Name = m[1]
	}
	return r, nil
}

// references is the reference text: the exemplars, then the Creator's own
// approved drawings in the same folder (newest style decisions first).
func (uc *IllustrationsUseCase) references(ctx context.Context, folderID, skipID string) string {
	var b strings.Builder
	n := 0
	add := func(i domain.Illustration, why string) {
		if n >= maxDrawReferences || i.Code == "" || i.ID == skipID {
			return
		}
		n++
		fmt.Fprintf(&b, "### %s — %s (%s)\n```tsx\n%s```\n\n", i.Name, i.Title, why, i.Code)
	}
	for _, e := range domain.ExemplarIllustrations() {
		add(e, "hình mẫu chuẩn")
	}
	if folderID != "" {
		if mine, err := uc.repo.ListIllustrations(ctx, IllustrationFilter{FolderID: folderID, Status: domain.IllustrationApproved}); err == nil {
			for _, i := range mine {
				if !i.Builtin {
					add(i, "hình Creator đã duyệt, cùng thư mục")
				}
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func (uc *IllustrationsUseCase) drawerSystem(ctx context.Context, folderID, skipID string) string {
	return strings.NewReplacer(
		"{{style}}", domain.IllustrationStyleGuide(),
		"{{helpers}}", domain.IllustrationHelpers(),
		"{{references}}", uc.references(ctx, folderID, skipID),
	).Replace(drawerPromptVI)
}

// draw runs the model until the renderer accepts the drawing. user is the
// first turn; each failed attempt adds the code it produced and the errors.
func (uc *IllustrationsUseCase) draw(ctx context.Context, system, user, model string) (drawnReply, IllustrationPreview, error) {
	if uc.drawer == nil {
		return drawnReply{}, IllustrationPreview{}, ErrDrawerDisabled
	}
	turn := user
	var lastErr error
	for attempt := 1; attempt <= maxDrawAttempts; attempt++ {
		started := time.Now()
		res, err := uc.drawer.llm.Chat(ctx, ChatRequest{
			System: system, User: turn, MaxTokens: uc.drawer.maxTokens, Temperature: 0.5, Model: model,
		})
		if uc.drawer.recorder != nil {
			rec := RecordFor(uc.drawer.llm.Name(), drawerRole, "illustration", "", res.Usage, started, err)
			rec.Phase = fmt.Sprintf("attempt_%d", attempt)
			uc.drawer.recorder.Record(ctx, rec)
		}
		if err != nil {
			return drawnReply{}, IllustrationPreview{}, err
		}
		reply, perr := parseDrawnReply(res.Content)
		if perr == nil {
			if verr := domain.ValidateIllustrationName(reply.Name); verr != nil {
				perr = verr
			}
		}
		if perr != nil {
			lastErr = perr
			turn = user + "\n\nLần trước câu trả lời sai khuôn: " + perr.Error() + ". Trả lời lại đúng khuôn."
			continue
		}
		preview, rerr := uc.render(ctx, reply.Name, reply.Code)
		if rerr == nil {
			return reply, preview, nil
		}
		lastErr = rerr
		turn = fmt.Sprintf("%s\n\nCode lần trước của bạn:\n```tsx\n%s```\n\nChưa qua kiểm tra: %s\nSửa đúng các lỗi đó (số dòng tính trong code trên), giữ nguyên ý hình, trả lời lại đầy đủ theo khuôn.",
			user, reply.Code, rerr.Error())
	}
	return drawnReply{}, IllustrationPreview{}, fmt.Errorf("AI vẽ %d lần vẫn chưa qua kiểm tra: %w", maxDrawAttempts, lastErr)
}

// Draw asks the model for a new drawing and stores it as a draft.
func (uc *IllustrationsUseCase) Draw(ctx context.Context, req DrawRequest) (domain.Illustration, error) {
	req.Description = strings.TrimSpace(req.Description)
	if req.Description == "" {
		return domain.Illustration{}, fmt.Errorf("mô tả hình cần vẽ là bắt buộc")
	}
	if err := uc.folderExists(ctx, req.FolderID); err != nil {
		return domain.Illustration{}, err
	}
	user := "Vẽ hình: " + req.Description
	if n := strings.TrimSpace(req.Name); n != "" {
		user += "\nTên component bắt buộc: " + n
	}
	reply, preview, err := uc.draw(ctx, uc.drawerSystem(ctx, req.FolderID, ""), user, req.Model)
	if err != nil {
		return domain.Illustration{}, err
	}
	i := domain.Illustration{
		Name: reply.Name, Title: reply.Title, FolderID: req.FolderID, Tags: strings.Split(reply.Tags, ","),
		Description: reply.Description, Usage: reply.Usage, Code: reply.Code,
	}
	if i, err = normalizeIllustration(i); err != nil {
		return i, err
	}
	i.Status, i.Version, i.Warnings = domain.IllustrationDraft, 1, preview.Warnings
	saved, err := uc.repo.CreateIllustration(ctx, i)
	if err != nil {
		return saved, err
	}
	if err := uc.repo.SaveIllustrationPreview(ctx, saved.ID, saved.Version, preview.PNG, preview.GIF); err != nil {
		return saved, err
	}
	saved.HasPreview = true
	return saved, nil
}

// Redraw asks the model to redo one of the Creator's drawings, following a
// note ("bánh xe to hơn"). The name stays; the result is a new draft version.
func (uc *IllustrationsUseCase) Redraw(ctx context.Context, id, note, model string) (domain.Illustration, error) {
	existing, err := uc.repo.GetIllustration(ctx, id)
	if err != nil {
		return existing, err
	}
	if existing.Builtin {
		return existing, ErrIllustrationReadOnly
	}
	user := fmt.Sprintf("Vẽ lại hình %s (%s): %s\nGiữ nguyên tên component %s và các prop đang có.\nCode hiện tại:\n```tsx\n%s```",
		existing.Name, existing.Title, existing.Description, existing.Name, existing.Code)
	if n := strings.TrimSpace(note); n != "" {
		user += "\nYêu cầu sửa của Creator: " + n
	} else {
		user += "\nCreator chưa ưng hình này: vẽ một phiên bản khác, đúng luật style hơn."
	}
	reply, preview, err := uc.draw(ctx, uc.drawerSystem(ctx, existing.FolderID, existing.ID), user, model)
	if err != nil {
		return existing, err
	}
	if reply.Name != existing.Name {
		return existing, fmt.Errorf("AI đổi tên component thành %s — vẽ lại phải giữ tên %s", reply.Name, existing.Name)
	}
	existing.Code = reply.Code
	if reply.Usage != "" {
		existing.Usage = reply.Usage
	}
	existing.Status, existing.Version, existing.Warnings = domain.IllustrationDraft, existing.Version+1, preview.Warnings
	saved, err := uc.repo.UpdateIllustration(ctx, existing)
	if err != nil {
		return saved, err
	}
	if err := uc.repo.SaveIllustrationPreview(ctx, id, saved.Version, preview.PNG, preview.GIF); err != nil {
		return saved, err
	}
	saved.HasPreview = true
	return saved, nil
}
