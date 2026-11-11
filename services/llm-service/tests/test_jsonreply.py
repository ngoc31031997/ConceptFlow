from app.jsonreply import first_json_object


def test_raw_newline_inside_a_string_is_read():
    obj, why = first_json_object('```json\n{"a": "dòng một\ndòng hai\tcó tab", "b": [1]}\n```')
    assert why == ""
    assert obj == {"a": "dòng một\ndòng hai\tcó tab", "b": [1]}


def test_text_after_the_object_is_ignored_and_a_broken_object_still_fails():
    assert first_json_object('{"a": 1} ghi chú }')[0] == {"a": 1}
    obj, why = first_json_object('{"a": ')
    assert obj is None and why
