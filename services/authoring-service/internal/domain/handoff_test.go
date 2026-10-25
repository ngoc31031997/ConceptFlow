package domain

import (
	"testing"
	"time"
)

func TestParseHandoffReply(t *testing.T) {
	ok := "{\"must\":[\"A\"],\"terms\":[\"  \"],\"glossary\":[{\"term\":\"ficin\",\"say\":\"phi-xin\",\"draw\":\"quầng\",\"avoid\":\"\"}]}"
	r, err := ParseHandoffReply(ok)
	if err != nil || len(r.Note.Must) != 1 || len(r.Note.Terms) != 0 || len(r.Terms) != 1 {
		t.Fatalf("reply = %+v err=%v", r, err)
	}
	for name, bad := range map[string]string{
		"no json":    "x",
		"empty":      "{\"must\":[]}",
		"empty term": "{\"must\":[\"a\"],\"glossary\":[{\"term\":\" \"}]}",
	} {
		if _, err := ParseHandoffReply(bad); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestGlossaryMergeKeepsExistingRows(t *testing.T) {
	g := Glossary{Terms: []GlossaryTerm{{Term: "Ficin", Say: "Creator", Mine: true}}}
	g = g.Merge([]GlossaryTerm{{Term: "ficin", Say: "AI"}, {Term: "ostiole"}})
	if len(g.Terms) != 2 || g.Terms[0].Say != "Creator" {
		t.Fatalf("glossary = %+v", g)
	}
}

func lockedFixture(t *testing.T, mutate func(map[string]any)) ScriptVersion {
	t.Helper()
	vals := map[string]any{
		PartBrief:          CreativeBrief{Tone: "ấm", OutOfScope: "chính trị"},
		PartOrder:          []string{"b1", "b2"},
		PartGlossary:       Glossary{Terms: []GlossaryTerm{{Term: "ficin"}}},
		PartBeat + "b1":    Beat{Fn: "Hook", Title: "T1", Goal: "G1", Next: "N1"},
		PartBeat + "b2":    Beat{Fn: "Payoff", Title: "T2", Goal: "G2"},
		PartScript + "b1":  ScriptBeat{Lines: []ScriptLine{{ID: "b1.1", Text: "Một."}}},
		PartScript + "b2":  ScriptBeat{Lines: []ScriptLine{{ID: "b2.1", Text: "Hai."}}},
		PartHandoff + "b1": HandoffNote{Must: []string{"m1"}, Terms: []string{"t1"}, Exact: []string{"e1"}, Free: []string{"f1"}, Unsure: []string{"u1"}},
		PartHandoff + "b2": HandoffNote{Must: []string{"m2"}},
	}
	if mutate != nil {
		mutate(vals)
	}
	v := ScriptVersion{Version: 1, LockedAt: time.Unix(0, 0)}
	for k, val := range vals {
		p, err := NewVersionPart(k, val)
		if err != nil {
			t.Fatal(err)
		}
		v.Parts = append(v.Parts, p)
	}
	return v
}

func TestHandoffForGivesEachRoleOnlyItsSections(t *testing.T) {
	v := lockedFixture(t, nil)
	sb, err := HandoffFor(HandoffRoleStoryboard, v)
	if err != nil || len(sb.Beats) != 2 {
		t.Fatalf("storyboard = %+v err=%v", sb, err)
	}
	b := sb.Beats[0]
	if len(b.Lines) != 1 || b.Goal != "G1" || b.Next != "N1" {
		t.Errorf("storyboard must read lines, goal and next: %+v", b)
	}
	for _, sec := range []string{HandoffMust, HandoffTerms, HandoffFree, HandoffUnsure} {
		if len(b.Notes[sec]) != 1 {
			t.Errorf("storyboard must read %s, got %v", sec, b.Notes[sec])
		}
	}
	if _, has := b.Notes[HandoffExact]; has {
		t.Error("storyboard must not read 'exact'")
	}
	code, _ := HandoffFor(HandoffRoleCode, v)
	if len(code.Beats[0].Notes) != 1 || len(code.Beats[0].Notes[HandoffFree]) != 1 || len(code.Beats[0].Lines) != 0 {
		t.Errorf("code reads only 'free': %+v", code.Beats[0])
	}
	art, _ := HandoffFor(HandoffRoleArtWorld, v)
	if art.Brief["tone"] != "ấm" || art.Brief["out_of_scope"] != "chính trị" || len(art.Glossary) != 1 {
		t.Errorf("art reads tone, scope and glossary: %+v", art)
	}
	color, _ := HandoffFor(HandoffRoleArtColor, v)
	if color.Beats[1].Fn != "Payoff" || color.Beats[1].Goal != "" {
		t.Errorf("colour reads only the function: %+v", color.Beats[1])
	}
	if _, err := HandoffFor("bogus", v); err == nil {
		t.Error("an unknown role must fail")
	}
}

func TestChangedPartsNamesOnlyWhatDiffers(t *testing.T) {
	v := lockedFixture(t, nil)
	now := lockedFixture(t, func(m map[string]any) {
		m[PartScript+"b2"] = ScriptBeat{Lines: []ScriptLine{{ID: "b2.1", Text: "Hai đổi."}}}
		delete(m, PartHandoff+"b1")
	})
	got := ChangedParts(v, now.Parts)
	set := map[string]bool{}
	for _, k := range got {
		set[k] = true
	}
	if len(got) != 2 || !set[PartScript+"b2"] || !set[PartHandoff+"b1"] {
		t.Fatalf("changed = %v, want script:b2 and handoff:b1", got)
	}
	label := PartLabel(PartScript+"b2", func(k string) int {
		if k == "b2" {
			return 2
		}
		return 0
	})
	if label != "Lời thoại · B2" {
		t.Errorf("label = %q", label)
	}
}
