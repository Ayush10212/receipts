"""Test AST extraction — file is never imported (Step 3.3 done-when)."""

from receipts_python_symbols.extract import extract
from tests.fixture_claims import (
    ATTRIBUTE_APPEND,
    ATTRIBUTE_MERGE,
    IMPORT_PANDAS,
    NUMPY_UFUNC,
    KWARG_REMOVED,
)


def claims_of(source: str, subtype: str = None):
    cs = extract(source, "test.py")
    if subtype:
        cs = [c for c in cs if c.subtype == subtype]
    return cs


def test_import_claim_locus():
    claims = claims_of(IMPORT_PANDAS, "import")
    assert any(c.symbol == "pandas" for c in claims)
    imp = next(c for c in claims if c.symbol == "pandas")
    assert imp.locus.file == "test.py"
    assert imp.locus.line >= 1


def test_attribute_append_detected():
    claims = claims_of(ATTRIBUTE_APPEND, "attribute")
    symbols = [c.symbol for c in claims]
    assert any("append" in s for s in symbols), f"expected append in {symbols}"


def test_attribute_merge_detected():
    claims = claims_of(ATTRIBUTE_MERGE, "attribute")
    symbols = [c.symbol for c in claims]
    assert any("merge" in s for s in symbols), f"expected merge in {symbols}"


def test_kwarg_claim_emitted():
    claims = claims_of(KWARG_REMOVED, "kwarg")
    assert len(claims) > 0, "expected at least one kwarg claim"
    kwarg_names = [c.kwarg for c in claims]
    assert "error_bad_lines" in kwarg_names


def test_numpy_attribute_detected():
    claims = claims_of(NUMPY_UFUNC, "attribute")
    symbols = [c.symbol for c in claims]
    assert any("add" in s for s in symbols), f"expected numpy.add in {symbols}"


def test_untraceable_receiver_not_in_claims():
    # Local variable assignment — receiver is untraceable.
    source = "df = some_func()\ndf.append({'a':1})\n"
    claims = claims_of(source, "attribute")
    # Should produce no attribute claims because df is not an import binding.
    assert all("append" not in c.symbol for c in claims)


def test_syntax_error_returns_empty():
    claims = extract("def (broken syntax", "bad.py")
    assert claims == []
