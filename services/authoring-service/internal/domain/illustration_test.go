package domain

import (
	"regexp"
	"strings"
	"testing"
)

// Every built-in row names a component the Remotion Engineer is told about
// (the kit section baked into its prompt), in a folder that exists.
func TestBuiltinIllustrationsMatchTheKitAndLiveInSystemFolders(t *testing.T) {
	folders := map[string]bool{}
	for _, f := range SystemIllustrationFolders() {
		if ValidateFolderID(f.ID) != nil {
			t.Errorf("folder id %q is not a slug", f.ID)
		}
		folders[f.ID] = true
	}
	documented := map[string]bool{}
	for _, m := range regexp.MustCompile(`<([A-Z]\w*)`).FindAllStringSubmatch(illustrationKitVI, -1) {
		documented[m[1]] = true
	}
	delete(documented, "AbsoluteFill")
	seen := map[string]bool{}
	for _, b := range BuiltinIllustrations() {
		if !folders[b.FolderID] {
			t.Errorf("%s is filed in unknown folder %q", b.Name, b.FolderID)
		}
		if !documented[b.Name] {
			t.Errorf("%s is not in the kit the prompt documents", b.Name)
		}
		if ValidateIllustrationName(b.Name) != nil || seen[b.Name] {
			t.Errorf("bad or duplicate built-in name %q", b.Name)
		}
		seen[b.Name] = true
	}
	for name := range documented {
		if !seen[name] {
			t.Errorf("kit component %s has no library row", name)
		}
	}
}

func TestStyleGuideKeepsItsRules(t *testing.T) {
	for _, rule := range []string{"[S1]", "[S9]", "[S13]", "[S19]", "[S24]"} {
		if !regexp.MustCompile(regexp.QuoteMeta(rule)).MatchString(IllustrationStyleGuide()) {
			t.Errorf("style guide lost rule %s", rule)
		}
	}
}

func TestExemplarFolderIsASystemFolderAndExemplarsAreReadOnly(t *testing.T) {
	found := false
	for _, f := range SystemIllustrationFolders() {
		found = found || (f.ID == ExemplarFolderID && f.IsSystem)
	}
	if !found {
		t.Fatal("no system Hình mẫu folder")
	}
	kit := Illustration{Builtin: true}
	original := Illustration{Exemplar: true, HomeFolderID: "dong-vat"}
	cp := Illustration{Exemplar: true, SourceID: "i1"}
	orphan := Illustration{Exemplar: true} // its source was deleted
	mine := Illustration{}
	if !kit.ReadOnly() || !original.ReadOnly() || !cp.ReadOnly() || mine.ReadOnly() {
		t.Fatal("read-only rule is wrong")
	}
	if original.IsExemplarCopy() || !cp.IsExemplarCopy() || !orphan.IsExemplarCopy() || mine.IsExemplarCopy() {
		t.Fatal("exemplar copy rule is wrong")
	}
}

func TestExemplarCopyNameAddsMauThenANumber(t *testing.T) {
	taken := map[string]bool{"BusMau": true, "BusMau2": true}
	if got := ExemplarCopyName("Bus", func(n string) bool { return taken[n] }); got != "BusMau3" {
		t.Fatalf("got %s", got)
	}
	long := "A" + strings.Repeat("b", 40) // 41 chars, the longest valid name
	got := ExemplarCopyName(long, func(string) bool { return false })
	if ValidateIllustrationName(got) != nil || !strings.HasSuffix(got, "Mau") {
		t.Fatalf("copy name of a long name is invalid: %s", got)
	}
}

func TestRenameExportRenamesOnlyTheExportAndTheUsageTags(t *testing.T) {
	code := "import React from 'react';\n\nexport function Bus({color}: P) {\n  return <BusWheel />;\n}\n"
	gotCode, gotUsage, err := RenameExport(code, "<Bus color /> — 320×210, cạnh <BusStop />", "BusMau")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotCode, "export function BusMau({color}: P) {") || !strings.Contains(gotCode, "<BusWheel />") {
		t.Fatalf("code: %s", gotCode)
	}
	if gotUsage != "<BusMau color /> — 320×210, cạnh <BusStop />" {
		t.Fatalf("usage: %s", gotUsage)
	}
	if _, _, err := RenameExport("const Bus = () => null;", "", "BusMau"); err == nil {
		t.Fatal("code without an export function must be refused")
	}
}

func TestOnlyProjectsPastTheResultScreenAreDoneWithTheLibrary(t *testing.T) {
	for _, s := range []ProjectStatus{StatusReadyToPublish, StatusPublishing, StatusPublished, StatusFailedPublishVideo, StatusDeleting} {
		if !s.DoneWithLibrary() {
			t.Errorf("%s should be done with the library", s)
		}
	}
	for _, s := range []ProjectStatus{StatusDraft, StatusAwaitingReview, StatusRendering, StatusFailedRenderScenes, StatusFailedAssembleVideo} {
		if s.DoneWithLibrary() {
			t.Errorf("%s may still render its drawings", s)
		}
	}
}
