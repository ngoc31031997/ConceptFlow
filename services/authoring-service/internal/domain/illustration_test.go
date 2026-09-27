package domain

import (
	"regexp"
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
