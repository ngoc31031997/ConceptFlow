package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestReferenceStyleID_PadsToTwoDigits(t *testing.T) {
	for seq, want := range map[int64]string{1: "R01", 12: "R12", 105: "R105"} {
		if got := ReferenceStyleID(seq); got != want {
			t.Errorf("seq %d: got %q, want %q", seq, got, want)
		}
	}
}

func TestNormalize_TrimsAndDropsRepeatedTags(t *testing.T) {
	got, err := ReferenceStyleInput{
		StyleName: "  Phẳng ấm  ",
		Intent:    "\n màu đất, nét mảnh \t",
		Tags:      []string{" ấm ", "Ấm", "", "phẳng", "ấm"},
	}.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	want := ReferenceStyleInput{StyleName: "Phẳng ấm", Intent: "màu đất, nét mảnh", Tags: []string{"ấm", "phẳng"},
		CoreRules: []string{}, Avoid: []string{}, Composition: ReferenceComposition{OptionalDevices: []string{}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestNormalize_NoTagsIsAnEmptyList(t *testing.T) {
	got, err := ReferenceStyleInput{StyleName: "n", Intent: "d"}.Normalize()
	if err != nil || got.Tags == nil || len(got.Tags) != 0 {
		t.Fatalf("tags %#v err %v, want an empty non-nil list", got.Tags, err)
	}
}

func TestNormalize_CountsCharactersNotBytes(t *testing.T) {
	if _, err := (ReferenceStyleInput{StyleName: strings.Repeat("ộ", 80), Intent: "d"}).Normalize(); err != nil {
		t.Fatalf("80 Vietnamese characters must fit the name: %v", err)
	}
}

func TestNormalize_RejectsBrokenRules(t *testing.T) {
	nineTags := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	cases := map[string]ReferenceStyleInput{
		"empty name":       {StyleName: "   ", Intent: "d"},
		"long name":        {StyleName: strings.Repeat("a", 81), Intent: "d"},
		"empty desc":       {StyleName: "n", Intent: " "},
		"long description": {StyleName: "n", Intent: strings.Repeat("a", 1001)},
		"too many tags":    {StyleName: "n", Intent: "d", Tags: nineTags},
		"long tag":         {StyleName: "n", Intent: "d", Tags: []string{strings.Repeat("a", 25)}},
	}
	for name, in := range cases {
		if _, err := in.Normalize(); !errors.Is(err, ErrInvalidReferenceStyle) {
			t.Errorf("%s: err %v, want ErrInvalidReferenceStyle", name, err)
		}
	}
	eightAfterDedup := append([]string{"a", "A"}, nineTags[1:8]...)
	if _, err := (ReferenceStyleInput{StyleName: "n", Intent: "d", Tags: eightAfterDedup}).Normalize(); err != nil {
		t.Errorf("repeated tags must not count towards the limit: %v", err)
	}
}

func TestNormalize_OptionalFieldsMayBeEmpty(t *testing.T) {
	if _, err := (ReferenceStyleInput{StyleName: "n", Intent: "s"}).Normalize(); err != nil {
		t.Fatalf("only name and summary are required: %v", err)
	}
}

func TestNormalize_TrimsDetailFields(t *testing.T) {
	got, err := ReferenceStyleInput{
		StyleName: "n", Intent: "s",
		Source:         ReferenceSource{URL: " https://example.com/a ", Title: " t ", Kind: " video "},
		Drawing:        ReferenceDrawing{Shape: " tròn "},
		CoreRules:      []string{" a ", "  ", "b"},
		Composition:    ReferenceComposition{Structure: " lưới ", OptionalDevices: []string{" inset "}},
		ColorRules:     ReferenceColorRules{Accents: " glow "},
		PromptTemplate: " {subject} ",
	}.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if got.Source.URL != "https://example.com/a" || got.Source.Title != "t" || got.Source.Kind != "video" || got.Drawing.Shape != "tròn" || got.Composition.Structure != "lưới" ||
		len(got.CoreRules) != 2 || got.CoreRules[0] != "a" || got.Composition.OptionalDevices[0] != "inset" || got.ColorRules.Accents != "glow" ||
		got.PromptTemplate != "{subject}" || got.Avoid == nil {
		t.Errorf("got %+v", got)
	}
}

func TestNormalize_NamesTheFieldAtFault(t *testing.T) {
	long := strings.Repeat("ộ", 301)
	cases := map[string]ReferenceStyleInput{
		"drawing.shape":                   {StyleName: "n", Intent: "s", Drawing: ReferenceDrawing{Shape: long}},
		"composition.optional_devices[0]": {StyleName: "n", Intent: "s", Composition: ReferenceComposition{OptionalDevices: []string{long}}},
		"color_rules.subject":             {StyleName: "n", Intent: "s", ColorRules: ReferenceColorRules{Subject: long}},
		"core_rules":                      {StyleName: "n", Intent: "s", CoreRules: make13()},
		"prompt_template":                 {StyleName: "n", Intent: "s", PromptTemplate: strings.Repeat("a", 1001)},
		"source.title":                    {StyleName: "n", Intent: "s", Source: ReferenceSource{Title: long}},
		"source.url":                      {StyleName: "n", Intent: "s", Source: ReferenceSource{URL: "ftp://x.com/a"}},
		"source.kind":                     {StyleName: "n", Intent: "s", Source: ReferenceSource{Kind: "film"}},
		"style_name":                      {Intent: "s"},
		"intent":                          {StyleName: "n"},
	}
	for field, in := range cases {
		_, err := in.Normalize()
		if !errors.Is(err, ErrInvalidReferenceStyle) || !strings.Contains(err.Error(), field+":") {
			t.Errorf("%s: err %v", field, err)
		}
	}
	if _, err := (ReferenceStyleInput{StyleName: "n", Intent: "s", Drawing: ReferenceDrawing{Shape: strings.Repeat("ộ", 300)}}).Normalize(); err != nil {
		t.Errorf("300 characters must fit: %v", err)
	}
}

// make13 returns one more list item than a reference list allows.
func make13() []string {
	out := make([]string, MaxReferenceListItems+1)
	for i := range out {
		out[i] = "rule"
	}
	return out
}
