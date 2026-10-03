import json

from app import storyboard as sbm
from app.pipeline import autofix, merger
from app.pipeline.checker import Diagnostic

SB = sbm.parse(json.dumps({
    "hero": "h",
    "palette": [{"role": "accent", "hex": "#F5B841"}],
    "scenes": [{"id": "hook", "shots": [{"id": "1.1", "visual": "v", "narration": "Câu một."},
                                         {"id": "1.2", "visual": "v", "narration": "Câu hai."}]}],
}))


def shot(i: str, body: str) -> str:
    return f"// Shot {i}\nfunction {merger.remotion_fn(i)}({{duration}}: ShotProps) {{\n  {body}\n}}"


def merged_with(body: str) -> tuple[dict[str, str], merger.Merged, int]:
    """Shot 1.1 with `body`, merged; returns the shots, the merged file and
    the file line `body` is on."""
    shots = {"1.1": shot("1.1", body), "1.2": shot("1.2", "return null;")}
    merged = merger.merge_remotion(SB, "const LAYOUT = {};", shots)
    line = next(n for n, text in enumerate(merged.code.splitlines(), 1) if body in text)
    return shots, merged, line


# --- PALETTE_<X> names ---------------------------------------------------------

def test_palette_name_matching_one_key_becomes_a_palette_member():
    code = "color: PALETTE_NEN_TROI,\nfill: PALETTE_nenHangDong,\nstroke: PALETTE_CHIM_PLACEHOLDER,"
    out, fixes = autofix.fix_palette_names("1.1", code, ["nenTroi", "nenHangDong", "chim"])
    assert out == "color: PALETTE.nenTroi,\nfill: PALETTE.nenHangDong,\nstroke: PALETTE.chim,"
    assert fixes == [
        autofix.Fix("1.1", 1, autofix.RULE_PALETTE_NAME, "PALETTE_NEN_TROI", "PALETTE.nenTroi"),
        autofix.Fix("1.1", 2, autofix.RULE_PALETTE_NAME, "PALETTE_nenHangDong", "PALETTE.nenHangDong"),
        autofix.Fix("1.1", 3, autofix.RULE_PALETTE_NAME, "PALETTE_CHIM_PLACEHOLDER", "PALETTE.chim"),
    ]


def test_placeholder_without_a_role_is_left_for_repair():
    code = "color: PALETTE_PLACEHOLDER,"
    assert autofix.fix_palette_names("1.1", code, ["nenTroi"]) == (code, [])


def test_name_matching_two_keys_is_left_alone():
    code = "color: PALETTE_NEN_TROI,"
    assert autofix.fix_palette_names("1.1", code, ["nenTroi", "nen_troi"]) == (code, [])


def test_palette_member_access_is_not_touched():
    code = "color: PALETTE.nenTroi, label: 'PALETTE.x', other: MY_PALETTE_NEN_TROI,"
    assert autofix.fix_palette_names("1.1", code, ["nenTroi", "x"]) == (code, [])


# --- tsc "Did you mean" ----------------------------------------------------------

def test_cannot_find_name_takes_the_frame_name_tsc_suggests():
    shots, merged, line = merged_with("const frame = useCurrentFrameSafe();")
    diag = Diagnostic("TS2552: Cannot find name 'useCurrentFrameSafe'. Did you mean 'useCurrentFrame'?", line,
                      rule="TS2552")
    applied: set = set()
    changed, fixes = autofix.fix_tsc_suggestions(shots, merged, [diag], merger.frame_names(), applied)
    assert list(changed) == ["1.1"]
    assert "const frame = useCurrentFrame();" in changed["1.1"] and "Safe" not in changed["1.1"]
    assert fixes == [autofix.Fix("1.1", 3, autofix.RULE_TSC_SUGGESTION, "useCurrentFrameSafe", "useCurrentFrame")]
    assert applied == {("1.1", "useCurrentFrameSafe", "useCurrentFrame")}


def test_suggestion_outside_the_frame_names_is_left_for_repair():
    shots, merged, line = merged_with("return <Shot1_3 />;")
    diag = Diagnostic("TS2552: Cannot find name 'Shot1_3'. Did you mean 'Shot1_2'?", line)
    assert autofix.fix_tsc_suggestions(shots, merged, [diag], merger.frame_names(), set()) == ({}, [])


def test_property_suggestion_differing_only_in_case_is_applied():
    shots, merged, line = merged_with("return <div style={{color: PALETTE.conNguoi}} />;")
    diag = Diagnostic("TS2551: Property 'conNguoi' does not exist on type '{ readonly connguoi: string; }'. "
                      "Did you mean 'connguoi'?", line, rule="TS2551")
    changed, fixes = autofix.fix_tsc_suggestions(shots, merged, [diag], merger.frame_names(), set())
    assert "color: PALETTE.connguoi" in changed["1.1"]
    assert [(f.before, f.after) for f in fixes] == [("conNguoi", "connguoi")]


def test_property_suggestion_with_another_name_is_left_for_repair():
    shots, merged, line = merged_with("return <div style={{color: PALETTE.conNguoi}} />;")
    diag = Diagnostic("TS2551: Property 'conNguoi' does not exist on type '{ nenpanel: string; }'. "
                      "Did you mean 'nenpanel'?", line)
    assert autofix.fix_tsc_suggestions(shots, merged, [diag], merger.frame_names(), set()) == ({}, [])


def test_a_suggestion_already_applied_is_not_applied_again():
    shots, merged, line = merged_with("const frame = useCurrentFrameSafe();")
    diag = Diagnostic("TS2552: Cannot find name 'useCurrentFrameSafe'. Did you mean 'useCurrentFrame'?", line)
    applied = {("1.1", "useCurrentFrameSafe", "useCurrentFrame")}
    assert autofix.fix_tsc_suggestions(shots, merged, [diag], merger.frame_names(), applied) == ({}, [])


def test_a_line_outside_every_shot_is_ignored():
    shots, merged, _ = merged_with("return null;")
    layout_line = merged.lines[merger.LAYOUT_KEY][0]
    diag = Diagnostic("TS2552: Cannot find name 'useCurrentFrameSafe'. Did you mean 'useCurrentFrame'?",
                      layout_line)
    assert autofix.fix_tsc_suggestions(shots, merged, [diag], merger.frame_names(), set()) == ({}, [])
