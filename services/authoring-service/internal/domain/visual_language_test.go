package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestDetailEnoughFollowsTheTierRule(t *testing.T) {
	cases := []struct {
		detail DetailLevel
		tier   Tier
		want   bool
	}{
		{DetailRich, TierHero, true}, {DetailRich, TierSupport, true}, {DetailRich, TierBackground, true},
		{DetailMedium, TierHero, false}, {DetailMedium, TierSupport, true}, {DetailMedium, TierBackground, true},
		{DetailSilhouette, TierHero, false}, {DetailSilhouette, TierSupport, false}, {DetailSilhouette, TierBackground, true},
		{"", TierBackground, false}, {"unknown", TierSupport, false},
	}
	for _, c := range cases {
		if got := DetailEnough(c.detail, c.tier); got != c.want {
			t.Errorf("DetailEnough(%q, %q) = %v, want %v", c.detail, c.tier, got, c.want)
		}
	}
}

func TestADrawingIsMadeWithTheDetailOfItsTier(t *testing.T) {
	for tier, want := range map[Tier]DetailLevel{TierHero: DetailRich, TierSupport: DetailMedium, TierBackground: DetailSilhouette} {
		if got := DetailFor(tier); got != want {
			t.Errorf("DetailFor(%q) = %q, want %q", tier, got, want)
		}
		if !DetailEnough(DetailFor(tier), tier) {
			t.Errorf("a drawing made for %q is not enough for %q", tier, tier)
		}
	}
	if TierOrSupport("star") != TierSupport || TierOrSupport("") != TierSupport || TierOrSupport(TierHero) != TierHero {
		t.Error("an unknown tier is not the supporting one")
	}
}

func TestEveryKitFigureNamesItsKitAndADetailLevel(t *testing.T) {
	for _, b := range BuiltinIllustrations() {
		if b.KitKey != PhatsangKitKey || !ValidDetail(b.DetailLevel) {
			t.Errorf("%s: kit %q detail %q", b.Name, b.KitKey, b.DetailLevel)
		}
	}
	byName := map[string]DetailLevel{}
	for _, b := range BuiltinIllustrations() {
		byName[b.Name] = b.DetailLevel
	}
	if byName["Airplane"] != DetailRich || byName["Person"] != DetailRich || byName["Cloud"] != DetailSilhouette {
		t.Errorf("detail levels: %v", byName)
	}
}

func TestRulesWithAnEmptyTextAreRefused(t *testing.T) {
	good := PhatsangRules()
	for name, mutate := range map[string]func(*VisualLanguageRules){
		"illustration": func(r *VisualLanguageRules) { r.Illustration = "  " },
		"backdrop":     func(r *VisualLanguageRules) { r.Backdrop = "" },
		"scene":        func(r *VisualLanguageRules) { r.Scene = "\n" },
	} {
		r := good
		mutate(&r)
		err := r.Validate()
		if !errors.Is(err, ErrInvalidVisualLanguage) || !strings.Contains(err.Error(), "luật") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestSampleCompositionsNameTheRenderingCompositions(t *testing.T) {
	if got := SampleComposition("fig", "wide"); got != "sample-fig-wide" {
		t.Fatalf("composition id %q", got)
	}
	if len(SampleScenes)*len(SampleFrames) != 4 {
		t.Fatal("the language page shows four samples")
	}
}
