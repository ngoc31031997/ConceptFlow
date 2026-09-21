package postgres

import (
	"testing"

	"orchestrator/internal/domain"
)

// These cover the CR-027 FR84.8 migration rule, which is the whole risk of
// the two-layer split: get it wrong and either a Creator's saved wording is
// destroyed, or phantom overrides are manufactured that freeze the pipeline
// on stale text. No live Postgres needed — the rule is pure.

// TestShouldMigrateToOverride_UntouchedRowIsLeftAlone — the common case. A
// database that matches the binary needs no overrides at all.
func TestShouldMigrateToOverride_UntouchedRowIsLeftAlone(t *testing.T) {
	def, ok := domain.DefaultPromptTemplate(domain.RoleStoryArchitect, "vi")
	if !ok {
		t.Fatal("expected a built-in default for story_architect/vi")
	}

	stored := domain.PromptTemplate{
		Role: domain.RoleStoryArchitect, Language: "vi",
		TemplateText: def.TemplateText, Version: def.Version,
	}

	if ShouldMigrateToOverride(stored) {
		t.Fatal("a row identical to the shipped wording must not become an override")
	}
}

// TestShouldMigrateToOverride_VersionDriftIsNotAnEdit is the regression this
// rule exists for. Measured on the live database 2026-09-21: story_architect
// was at version 4 (vi) and 3 (en) while the binary shipped version 2, and
// the text matched byte for byte — because every Reset goes through Update,
// which bumps the version.
//
// A version-based rule would create two overrides here, switch them on, and
// freeze the pipeline on a copy of today's shipped text forever.
func TestShouldMigrateToOverride_VersionDriftIsNotAnEdit(t *testing.T) {
	for _, language := range []string{"vi", "en"} {
		def, ok := domain.DefaultPromptTemplate(domain.RoleStoryArchitect, language)
		if !ok {
			t.Fatalf("expected a built-in default for story_architect/%s", language)
		}

		stored := domain.PromptTemplate{
			Role: domain.RoleStoryArchitect, Language: language,
			TemplateText: def.TemplateText,
			Version:      def.Version + 2, // exactly the live drift
		}

		if ShouldMigrateToOverride(stored) {
			t.Fatalf("%s: a bumped version with identical text is not an edit", language)
		}
	}
}

// TestShouldMigrateToOverride_RealEditIsPreserved — the case that must never
// silently lose data.
func TestShouldMigrateToOverride_RealEditIsPreserved(t *testing.T) {
	def, _ := domain.DefaultPromptTemplate(domain.RoleManimEngineer, "vi")

	stored := domain.PromptTemplate{
		Role: domain.RoleManimEngineer, Language: "vi",
		TemplateText: def.TemplateText + "\n\nGHI CHÚ RIÊNG CỦA CREATOR",
		Version:      def.Version, // same version, different text
	}

	if !ShouldMigrateToOverride(stored) {
		t.Fatal("hand-edited wording must be preserved as an override")
	}
}

// TestShouldMigrateToOverride_EvenOneWhitespaceCounts — the comparison is
// exact on purpose. A trailing newline the Creator added is still their text,
// and "close enough" is not a safe standard when the cost of being wrong is
// losing it.
func TestShouldMigrateToOverride_EvenOneWhitespaceCounts(t *testing.T) {
	def, _ := domain.DefaultPromptTemplate(domain.RoleScriptReviewer, "en")

	stored := domain.PromptTemplate{
		Role: domain.RoleScriptReviewer, Language: "en",
		TemplateText: def.TemplateText + "\n",
	}

	if !ShouldMigrateToOverride(stored) {
		t.Fatal("an exact comparison must notice a trailing newline")
	}
}

// TestShouldMigrateToOverride_RoleTheBinaryNoLongerShips — nothing to compare
// against, so keep the wording rather than drop what cannot be recovered.
func TestShouldMigrateToOverride_RoleTheBinaryNoLongerShips(t *testing.T) {
	stored := domain.PromptTemplate{
		Role: domain.PromptRole("retired_role"), Language: "vi",
		TemplateText: "wording for a role this binary forgot",
	}

	if !ShouldMigrateToOverride(stored) {
		t.Fatal("wording for an unknown role must be kept, not discarded")
	}
}

// TestShouldMigrateToOverride_EveryShippedRowToday — a guard against the
// migration firing on a database that is simply up to date. This mirrors the
// measurement taken before the work started: all 12 rows matched the binary.
func TestShouldMigrateToOverride_EveryShippedRowToday(t *testing.T) {
	for _, def := range domain.DefaultPromptTemplates() {
		stored := domain.PromptTemplate{
			Role: def.Role, Language: def.Language,
			TemplateText: def.TemplateText, Version: def.Version,
		}
		if ShouldMigrateToOverride(stored) {
			t.Fatalf("%s/%s: a freshly seeded row must not migrate", def.Role, def.Language)
		}
	}
}
