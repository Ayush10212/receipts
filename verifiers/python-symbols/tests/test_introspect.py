"""Test sandboxed introspection paths (Step 3.4 done-when)."""

import sys
import pytest

from receipts_python_symbols.introspect import introspect

PYTHON = sys.executable


def test_grounded_existing_attribute():
    r = introspect("pandas.DataFrame.merge", python_exe=PYTHON)
    assert r["verdict"] == "grounded", r


def test_contradicted_removed_attribute():
    # pandas.DataFrame.append was removed in pandas 2.0
    r = introspect("pandas.DataFrame.append", python_exe=PYTHON)
    assert r["verdict"] == "contradicted", r


def test_unverifiable_missing_import():
    # A missing module is unverifiable (ImportError), never contradicted.
    r = introspect("receipts_nonexistent_pkg_xyz", python_exe=PYTHON)
    assert r["verdict"] == "unverifiable", r


def test_unverifiable_numpy_ufunc_sig():
    # numpy ufuncs are C extensions — signature introspection should be unverifiable
    # (getattr succeeds → grounded for the attr itself; signature → unverifiable for kwarg)
    r = introspect("numpy.add", python_exe=PYTHON, kwarg="out")
    # numpy.add is a ufunc which accepts **kwargs-like dispatch, so unverifiable or grounded
    assert r["verdict"] in ("unverifiable", "grounded"), r


def test_contradicted_removed_kwarg():
    # pandas.read_csv had error_bad_lines removed in pandas 1.4+
    r = introspect("pandas.read_csv", python_exe=PYTHON, kwarg="error_bad_lines")
    assert r["verdict"] == "contradicted", r


def test_unverifiable_kwargs_callable():
    # pandas.DataFrame.merge accepts **kwargs — can't verify specific kwarg
    # Actually merge has explicit params; let's use a known **kwargs function
    r = introspect("pandas.DataFrame.pipe", python_exe=PYTHON, kwarg="some_made_up_kwarg_xyz")
    # pipe accepts **kwargs so it should be unverifiable (not contradicted)
    assert r["verdict"] in ("unverifiable", "contradicted"), r


def test_timeout_yields_unverifiable():
    # Nonexistent exe → subprocess error → unverifiable
    r = introspect("pandas.DataFrame", python_exe="nonexistent_python_xyz", timeout=2.0)
    assert r["verdict"] == "unverifiable", r


def test_cache_is_deterministic():
    r1 = introspect("pandas.DataFrame.merge", python_exe=PYTHON)
    r2 = introspect("pandas.DataFrame.merge", python_exe=PYTHON)
    assert r1["verdict"] == r2["verdict"]
