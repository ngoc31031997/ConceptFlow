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
		Name:    "  Phẳng ấm  ",
		Summary: "\n màu đất, nét mảnh \t",
		Tags:    []string{" ấm ", "Ấm", "", "phẳng", "ấm"},
	}.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	want := ReferenceStyleInput{Name: "Phẳng ấm", Summary: "màu đất, nét mảnh", Tags: []string{"ấm", "phẳng"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestNormalize_NoTagsIsAnEmptyList(t *testing.T) {
	got, err := ReferenceStyleInput{Name: "n", Summary: "d"}.Normalize()
	if err != nil || got.Tags == nil || len(got.Tags) != 0 {
		t.Fatalf("tags %#v err %v, want an empty non-nil list", got.Tags, err)
	}
}

func TestNormalize_CountsCharactersNotBytes(t *testing.T) {
	if _, err := (ReferenceStyleInput{Name: strings.Repeat("ộ", 80), Summary: "d"}).Normalize(); err != nil {
		t.Fatalf("80 Vietnamese characters must fit the name: %v", err)
	}
}

func TestNormalize_RejectsBrokenRules(t *testing.T) {
	nineTags := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	cases := map[string]ReferenceStyleInput{
		"empty name":       {Name: "   ", Summary: "d"},
		"long name":        {Name: strings.Repeat("a", 81), Summary: "d"},
		"empty desc":       {Name: "n", Summary: " "},
		"long description": {Name: "n", Summary: strings.Repeat("a", 1001)},
		"too many tags":    {Name: "n", Summary: "d", Tags: nineTags},
		"long tag":         {Name: "n", Summary: "d", Tags: []string{strings.Repeat("a", 25)}},
	}
	for name, in := range cases {
		if _, err := in.Normalize(); !errors.Is(err, ErrInvalidReferenceStyle) {
			t.Errorf("%s: err %v, want ErrInvalidReferenceStyle", name, err)
		}
	}
	eightAfterDedup := append([]string{"a", "A"}, nineTags[1:8]...)
	if _, err := (ReferenceStyleInput{Name: "n", Summary: "d", Tags: eightAfterDedup}).Normalize(); err != nil {
		t.Errorf("repeated tags must not count towards the limit: %v", err)
	}
}

func TestNormalize_OptionalFieldsMayBeEmpty(t *testing.T) {
	if _, err := (ReferenceStyleInput{Name: "n", Summary: "s"}).Normalize(); err != nil {
		t.Fatalf("only name and summary are required: %v", err)
	}
}

func TestNormalize_TrimsDetailFields(t *testing.T) {
	got, err := ReferenceStyleInput{
		Name: "n", Summary: "s",
		Source:  ReferenceSource{URL: " https://example.com/a ", Title: " t ", Kind: " video "},
		Drawing: ReferenceDrawing{Shape: " tròn "},
		Layout:  ReferenceLayout{Structure: " lưới "},
	}.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if got.Source.URL != "https://example.com/a" || got.Source.Title != "t" || got.Source.Kind != "video" || got.Drawing.Shape != "tròn" || got.Layout.Structure != "lưới" {
		t.Errorf("got %+v", got)
	}
}

func TestNormalize_NamesTheFieldAtFault(t *testing.T) {
	long := strings.Repeat("ộ", 301)
	cases := map[string]ReferenceStyleInput{
		"drawing.shape":              {Name: "n", Summary: "s", Drawing: ReferenceDrawing{Shape: long}},
		"layout.persistent_elements": {Name: "n", Summary: "s", Layout: ReferenceLayout{PersistentElements: long}},
		"source.title":               {Name: "n", Summary: "s", Source: ReferenceSource{Title: long}},
		"source.url":                 {Name: "n", Summary: "s", Source: ReferenceSource{URL: "ftp://x.com/a"}},
		"source.kind":                {Name: "n", Summary: "s", Source: ReferenceSource{Kind: "film"}},
		"name":                       {Summary: "s"},
		"summary":                    {Name: "n"},
	}
	for field, in := range cases {
		_, err := in.Normalize()
		if !errors.Is(err, ErrInvalidReferenceStyle) || !strings.Contains(err.Error(), field+":") {
			t.Errorf("%s: err %v", field, err)
		}
	}
	if _, err := (ReferenceStyleInput{Name: "n", Summary: "s", Drawing: ReferenceDrawing{Shape: strings.Repeat("ộ", 300)}}).Normalize(); err != nil {
		t.Errorf("300 characters must fit: %v", err)
	}
}
