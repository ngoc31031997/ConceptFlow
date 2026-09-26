package domain

import (
	"strings"
	"testing"
)

func TestSystemArchetypesAreAtoDAndComplete(t *testing.T) {
	got := SystemVideoArchetypes()
	if len(got) != 4 {
		t.Fatalf("want 4 system kinds, got %d", len(got))
	}
	for i, a := range got {
		if a.Code != string(rune('A'+i)) || !a.IsSystem || a.ID != "system-"+a.Code {
			t.Errorf("kind %d: unexpected %+v", i, a)
		}
		if a.Name == "" || a.WhenToUse == "" || a.Playbook == "" {
			t.Errorf("kind %s has an empty field", a.Code)
		}
	}
}

func TestBuildVideoArchetypesSectionListsMenuThenPlaybooks(t *testing.T) {
	out := BuildVideoArchetypesSection([]VideoArchetype{
		{Code: "A", Name: "Một", WhenToUse: "x", Playbook: "pa"},
		{Code: "E", Name: "Năm", WhenToUse: "y", Playbook: "pe"},
	})
	for _, want := range []string{"- A — MỘT: x", "- E — NĂM: y", "### Playbook A — Một\npa", "### Playbook E — Năm\npe"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Index(out, "- E —") > strings.Index(out, "### Playbook A") {
		t.Error("the menu must come before the playbooks")
	}
	if !strings.Contains(BuildVideoArchetypesSection(nil), "chưa có kiểu video") {
		t.Error("an empty table must say so")
	}
}
