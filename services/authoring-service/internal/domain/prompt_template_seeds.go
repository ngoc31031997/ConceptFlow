package domain

// DefaultPromptTemplates returns the seed rows of the prompt library: the
// roles the running pipeline writes with a shipped wording. They are
// Vietnamese only: these are instructions the Creator reads, while the video's
// narration language is chosen per project.
func DefaultPromptTemplates() []PromptTemplate {
	return []PromptTemplate{
		{Role: RoleCreativeBrief, Language: "vi", Version: 1, TemplateText: creativeBriefVI},
		{Role: RoleResearchQuestions, Language: "vi", Version: 1, TemplateText: researchQuestionsVI},
		{Role: RoleResearchPrompt, Language: "vi", Version: 1, TemplateText: researchPromptVI},
		{Role: RoleStoryApproaches, Language: "vi", Version: 1, TemplateText: storyApproachesVI},
		{Role: RoleStoryFrame, Language: "vi", Version: 1, TemplateText: storyFrameVI},
	}
}

// DefaultPromptTemplate returns the built-in default for one role/language,
// or false if the pair has no shipped default. Backs the admin screen's
// "restore the shipped wording" action: seeding itself is insert-if-absent, so
// this is the only way a running database gets a shipped default back, and it
// happens because an editor asked for it rather than because a process
// restarted.
func DefaultPromptTemplate(role PromptRole, language string) (PromptTemplate, bool) {
	for _, t := range DefaultPromptTemplates() {
		if t.Role == role && t.Language == language {
			return t, true
		}
	}
	return PromptTemplate{}, false
}
