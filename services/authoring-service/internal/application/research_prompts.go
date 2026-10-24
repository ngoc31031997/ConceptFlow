package application

import (
	"context"
	"fmt"
	"strings"

	"authoring/internal/domain"
)

const (
	researchHeading      = "CÂU HỎI CẦN TRA"
	followUpHeading      = "CÂU HỎI CẦN TRA THÊM"
	followUpFactsHeading = "DỮ KIỆN CẦN NGUỒN KHÁC"
)

// ResearchPrompt builds the text the Creator copies into an outside research
// tool: the topic and brief, every research question, the rules for sources and
// the JSON shape to answer in.
func (uc *ResearchDeskUseCase) ResearchPrompt(ctx context.Context, projectID string) (string, error) {
	state, err := uc.promptInputs(ctx, projectID)
	if err != nil {
		return "", err
	}
	return uc.renderResearchPrompt(ctx, state, researchHeading, state.questions, "", "")
}

// FollowUpPrompt builds the prompt for another round: only the questions that
// still lack two independent matching sources or hold a fact whose quote was
// not found, the facts that need a different source, and the code to continue
// from. It is an error when every question is well sourced.
func (uc *ResearchDeskUseCase) FollowUpPrompt(ctx context.Context, projectID string) (string, error) {
	in, err := uc.promptInputs(ctx, projectID)
	if err != nil {
		return "", err
	}
	facts, err := uc.facts(ctx, projectID)
	if err != nil {
		return "", err
	}
	weak := map[string]bool{}
	for _, c := range coverageOf(in.questions, facts) {
		if c.Weak {
			weak[c.Question] = true
		}
	}
	var questions []QuestionView
	for _, q := range in.questions {
		if weak[q.ID] {
			questions = append(questions, q)
		}
	}
	if len(questions) == 0 {
		return "", fmt.Errorf("%w: mọi câu hỏi đã đủ nguồn", domain.ErrInvalidStatus)
	}
	var extra strings.Builder
	var ids []string
	for _, f := range facts {
		ids = append(ids, f.ID)
		if f.Keep && f.Link.State == domain.LinkBad {
			if extra.Len() == 0 {
				extra.WriteString("\n" + followUpFactsHeading + "\n")
			}
			extra.WriteString(f.ID + ". " + f.ClaimVI + " (đoạn trích cũ không khớp trang nguồn)\n")
		}
	}
	note := ". Đánh mã tiếp từ " + domain.NextFactID(ids)
	return uc.renderResearchPrompt(ctx, in, followUpHeading, questions, extra.String(), note)
}

// promptInputs is what both prompts are built from.
type promptInputs struct {
	topic     string
	idea      domain.ProjectIdea
	brief     domain.ProjectBrief
	duration  string
	questions []QuestionView
}

func (uc *ResearchDeskUseCase) promptInputs(ctx context.Context, projectID string) (promptInputs, error) {
	idea, brief, err := uc.brief(ctx, projectID)
	if err != nil {
		return promptInputs{}, err
	}
	topic, err := uc.units.GetAuthoringTopic(ctx, projectID)
	if err != nil {
		return promptInputs{}, fmt.Errorf("load topic: %w", err)
	}
	questions, err := uc.questions(ctx, projectID)
	if err != nil {
		return promptInputs{}, err
	}
	if len(questions) == 0 {
		return promptInputs{}, fmt.Errorf("%w: chưa có câu hỏi nào để tra", domain.ErrInvalidStatus)
	}
	return promptInputs{topic: topic, idea: idea, brief: brief, duration: durationLabel(brief), questions: questions}, nil
}

func (uc *ResearchDeskUseCase) renderResearchPrompt(ctx context.Context, in promptInputs, heading string, questions []QuestionView, extra, answerNote string) (string, error) {
	prompt, err := uc.prompts.GetActive(ctx, domain.RoleResearchPrompt)
	if err != nil {
		return "", fmt.Errorf("load research prompt: %w", err)
	}
	lines := make([]string, 0, len(questions))
	for _, q := range questions {
		lines = append(lines, q.ID+". "+q.Text)
	}
	return domain.RenderPromptTemplate(prompt.TemplateText, map[string]string{
		"topic":             in.topic,
		"central_question":  in.idea.CentralQuestion,
		"audience":          in.brief.Audience,
		"target_duration":   in.duration,
		"questions_heading": heading,
		"questions":         strings.Join(lines, "\n"),
		"extra":             extra,
		"answer_note":       answerNote,
	}), nil
}
