"""The kit object every drawing takes: complete, numbers in range, colours as #RRGGBB."""

import pytest

from domain.kit import InvalidKitError, validate_kit
from tests.kit_fixtures import phatsang_kit


def test_the_phatsang_values_make_a_valid_kit():
    kit = phatsang_kit()
    assert validate_kit(kit) is kit


@pytest.mark.parametrize("kit", [None, {}])
def test_a_missing_kit_is_refused(kit):
    with pytest.raises(InvalidKitError, match="thiếu kit"):
        validate_kit(kit)


def test_a_non_object_is_refused():
    with pytest.raises(InvalidKitError, match="phải là object"):
        validate_kit("phatsang")


def test_a_missing_field_is_named():
    kit = phatsang_kit()
    del kit["haze"]["mid"]
    with pytest.raises(InvalidKitError, match=r"thiếu trường haze\.mid"):
        validate_kit(kit)


def test_an_unknown_field_is_named():
    kit = phatsang_kit()
    kit["previewLook"]["extra"] = 1
    with pytest.raises(InvalidKitError, match=r"previewLook có trường lạ: extra"):
        validate_kit(kit)


@pytest.mark.parametrize("path, value", [
    (("grain",), 1.01), (("grain",), -0.1), (("specular", "metal"), 2),
    (("previewLook", "light", "x"), -1), (("previewLook", "light", "intensity"), 1.5),
])
def test_a_number_outside_zero_to_one_is_refused(path, value):
    kit = phatsang_kit()
    node = kit
    for key in path[:-1]:
        node = node[key]
    node[path[-1]] = value
    with pytest.raises(InvalidKitError, match="kit." + r"\.".join(path)):
        validate_kit(kit)


@pytest.mark.parametrize("field", ["warmth", "contrast", "saturation"])
def test_the_grade_is_signed_from_minus_one_to_one(field):
    kit = phatsang_kit()
    for ok in (-1, -0.4, 1):
        kit["previewLook"]["grade"][field] = ok
        assert validate_kit(kit)
    for bad in (-1.01, 1.5):
        kit["previewLook"]["grade"][field] = bad
        with pytest.raises(InvalidKitError, match=rf"previewLook\.grade\.{field}"):
            validate_kit(kit)


def test_the_blur_radius_is_in_px_so_above_one_is_fine_but_negative_is_not():
    kit = phatsang_kit()
    kit["dof"]["near"] = 24
    assert validate_kit(kit)
    kit["dof"]["near"] = -1
    with pytest.raises(InvalidKitError, match=r"kit\.dof\.near"):
        validate_kit(kit)


@pytest.mark.parametrize("value", [True, "0.5", None, float("nan")])
def test_a_value_that_is_not_a_number_is_refused(value):
    kit = phatsang_kit()
    kit["bloom"] = value
    with pytest.raises(InvalidKitError, match=r"kit\.bloom"):
        validate_kit(kit)


@pytest.mark.parametrize("color", ["red", "#fff", "#12345G", "141A3A", 5])
def test_a_colour_must_be_rrggbb(color):
    kit = phatsang_kit()
    kit["ink"] = color
    with pytest.raises(InvalidKitError, match=r"kit\.ink"):
        validate_kit(kit)


def test_the_focus_is_one_of_the_five_layers():
    kit = phatsang_kit()
    for layer in ("sky", "far", "mid", "near", "figures"):
        kit["previewLook"]["focus"] = layer
        assert validate_kit(kit)
    kit["previewLook"]["focus"] = "back"
    with pytest.raises(InvalidKitError, match=r"previewLook\.focus"):
        validate_kit(kit)
