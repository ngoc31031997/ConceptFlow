package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const validKitJSON = `{"outline":0,"radius":0.18,"rimWidth":0.035,"rimStrength":0.7,"shadowStrength":0.35,"aoStrength":0.25,
"specular":{"glossy":0.5,"metal":0.8,"glass":0.9},
"texture":{"fur":0.25,"fabric":0.12,"wood":0.2,"stone":0.22,"organic":0.15,"leaf":0.12},
"sss":0.4,"glowBlur":0.08,"bloom":0.5,"haze":{"far":0.35,"mid":0.15},"dof":{"sky":6,"far":3,"mid":1,"near":10},
"grain":0.1,"vignette":0.2,"ink":"#2B2118",
"previewLook":{"light":{"x":0.3,"y":0.2,"color":"#FFE7B0","intensity":0.8},"ambient":"#8899AA",
"atmosphere":{"color":"#CCDDEE","density":0.2},"focus":"figures","grade":{"warmth":0.5,"contrast":0.5,"saturation":0.5}}}`

func decodeKit(t *testing.T, raw string) (ReferenceKit, error) {
	t.Helper()
	var k ReferenceKit
	err := json.Unmarshal([]byte(raw), &k)
	return k, err
}

func TestReferenceKit_ValidKitNormalizes(t *testing.T) {
	k, err := decodeKit(t, validKitJSON)
	if err != nil {
		t.Fatal(err)
	}
	k.PreviewLook.Grade.Contrast = -1
	got, err := k.Normalize()
	if err != nil || got.RimWidth != 0.035 || got.DOF.Near != 10 || got.PreviewLook.Focus != "figures" {
		t.Fatalf("got %+v, err %v", got, err)
	}
}

func TestReferenceKit_MissingOrUnknownKeyIsRejected(t *testing.T) {
	cases := map[string]string{
		"kit.grain":                   strings.Replace(validKitJSON, `"grain":0.1,`, "", 1),
		"kit.previewLook.light.color": strings.Replace(validKitJSON, `"color":"#FFE7B0",`, "", 1),
		"kit.zzz":                     strings.Replace(validKitJSON, `"outline":0,`, `"outline":0,"zzz":1,`, 1),
		"kit.dof":                     strings.Replace(validKitJSON, `"dof":{"sky":6,"far":3,"mid":1,"near":10}`, `"dof":null`, 1),
		"kit.haze.mid":                strings.Replace(validKitJSON, `,"mid":0.15}`, `}`, 1),
	}
	for want, raw := range cases {
		_, err := decodeKit(t, raw)
		if !errors.Is(err, ErrInvalidReferenceStyle) || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err %v", want, err)
		}
	}
	if _, err := decodeKit(t, strings.Replace(validKitJSON, `"outline":0`, `"outline":"x"`, 1)); !errors.Is(err, ErrInvalidReferenceStyle) {
		t.Errorf("wrong type: err %v", err)
	}
}

func TestReferenceKit_RangesColoursAndFocus(t *testing.T) {
	cases := map[string]string{
		"kit.rimWidth":                 strings.Replace(validKitJSON, `"rimWidth":0.035`, `"rimWidth":1.5`, 1),
		"kit.texture.fur":              strings.Replace(validKitJSON, `"fur":0.25`, `"fur":-0.1`, 1),
		"kit.dof.near":                 strings.Replace(validKitJSON, `"near":10`, `"near":-1`, 1),
		"kit.ink":                      strings.Replace(validKitJSON, `"#2B2118"`, `"red"`, 1),
		"kit.previewLook.ambient":      strings.Replace(validKitJSON, `"#8899AA"`, `"#89A"`, 1),
		"kit.previewLook.focus":        strings.Replace(validKitJSON, `"figures"`, `"back"`, 1),
		"kit.previewLook.grade.warmth": strings.Replace(validKitJSON, `"warmth":0.5`, `"warmth":-1.5`, 1),
	}
	for want, raw := range cases {
		k, err := decodeKit(t, raw)
		if err != nil {
			t.Fatalf("%s: decode: %v", want, err)
		}
		if _, err := k.Normalize(); !errors.Is(err, ErrInvalidReferenceStyle) || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err %v", want, err)
		}
	}
}

func TestReferenceStyleInput_KitIsOptionalAndChecked(t *testing.T) {
	var in ReferenceStyleInput
	if err := json.Unmarshal([]byte(`{"style_name":"a","intent":"d","kit":null}`), &in); err != nil {
		t.Fatal(err)
	}
	if out, err := in.Normalize(); err != nil || out.Kit != nil {
		t.Fatalf("empty kit: %+v, %v", out.Kit, err)
	}
	bad := `{"style_name":"a","intent":"d","kit":` + strings.Replace(validKitJSON, `"grain":0.1`, `"grain":2`, 1) + `}`
	var in2 ReferenceStyleInput
	if err := json.Unmarshal([]byte(bad), &in2); err != nil {
		t.Fatal(err)
	}
	if _, err := in2.Normalize(); !errors.Is(err, ErrInvalidReferenceStyle) {
		t.Errorf("bad kit: err %v", err)
	}
}
