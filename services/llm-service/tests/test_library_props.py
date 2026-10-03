"""The prop types of a library drawing are read from its code, never guessed."""

from app.pipeline.library_props import MAX_SIGNATURE_CHARS, prop_signature

GALAXY = """import React from 'react';
// core: 'normal' lõi vàng · 'flare' lõi đỏ rực · 'faded' nét vẽ trên giấy.
export function Galaxy({
  color = '#5B2C6F',
  core = 'normal',
  ...fig
}: FigureProps & {
  color?: string;
  core?: 'normal' | 'flare' | 'faded';
}) {
  return <g />;
}
"""


def test_inline_literal_union_is_kept():
    assert prop_signature(GALAXY, "Galaxy") == "color?: string; core?: 'normal' | 'flare' | 'faded'"


def test_a_literal_union_alias_is_replaced_by_its_values():
    code = ("type DoorOpen = 'closed' | 'ajar' | 'open';\n"
            "export function NarrowDoor({open = 'ajar', ...fig}: FigureProps & {open?: DoorOpen; panel?: string}) {}\n")
    assert prop_signature(code, "NarrowDoor") == "open?: 'closed' | 'ajar' | 'open'; panel?: string"


def test_a_named_props_parameter_is_read_too():
    code = "export function HatchPanel(\n  props: FigureProps & {\n    gap?: number;\n  }\n) {}\n"
    assert prop_signature(code, "HatchPanel") == "gap?: number"


def test_nested_and_function_types_stay_whole():
    code = ("export function Chart({a = 1 < 2, ...fig}: FigureProps & {\n"
            "  points?: Array<{x: number; y: number}>;\n  onDone?: () => void;\n}) {}\n")
    assert prop_signature(code, "Chart") == "points?: Array<{x: number; y: number}>; onDone?: () => void"


def test_no_signature_when_the_component_is_written_another_way():
    assert prop_signature("export const Galaxy = (p: {core: string}) => null;", "Galaxy") is None
    assert prop_signature("export function Galaxy({...fig}: FigureProps) {}", "Galaxy") is None
    assert prop_signature(GALAXY, "Nebula") is None


def test_no_signature_when_it_is_too_long():
    members = "".join(f"  prop{i}?: 'a' | 'b' | 'c';\n" for i in range(40))
    code = f"export function Big({{...fig}}: FigureProps & {{\n{members}}}) {{}}\n"
    assert len(members) > MAX_SIGNATURE_CHARS
    assert prop_signature(code, "Big") is None
