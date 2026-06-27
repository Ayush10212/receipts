"""Resolve the Python interpreter for introspection. Loud on failure — never silent."""

import os
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


@dataclass
class ResolvedEnv:
    prefix: str       # sys.prefix of the chosen interpreter
    python: str       # absolute path to the python executable
    version: str      # e.g. "3.11.4"
    language: str = "python"


class EnvResolutionError(RuntimeError):
    """Raised when no suitable Python environment can be found.  Never swallowed."""


def resolve(workdir: str, python_flag: str | None = None) -> ResolvedEnv:
    """
    Resolve in order:
      1. --python flag (explicit path)
      2. $VIRTUAL_ENV
      3. <workdir>/.venv  or  <workdir>/venv
      4. poetry / uv / pdm managed venv
      5. FAIL — never fall back to system Python silently
    """
    candidates = [
        ("--python flag",   lambda: _from_explicit(python_flag) if python_flag else None),
        ("$VIRTUAL_ENV",    lambda: _from_env_var()),
        (".venv/venv",      lambda: _from_local_venv(workdir)),
        ("poetry",          lambda: _from_poetry(workdir)),
        ("uv",              lambda: _from_uv(workdir)),
        ("pdm",             lambda: _from_pdm(workdir)),
    ]

    tried = []
    for source, fn in candidates:
        try:
            result = fn()
            if result is not None:
                return result
        except Exception as exc:
            tried.append(f"  {source}: {exc}")
            continue
        tried.append(f"  {source}: not found")

    raise EnvResolutionError(
        "receipts: no Python environment found. Tried:\n"
        + "\n".join(tried)
        + "\nSet --python, activate a venv, or run inside a poetry/uv/pdm project."
    )


def _probe(python_exe: str) -> ResolvedEnv:
    """Run the candidate interpreter to get its prefix and version."""
    result = subprocess.run(
        [python_exe, "-c",
         "import sys; print(sys.prefix); print(sys.version.split()[0])"],
        capture_output=True, text=True, timeout=10,
    )
    if result.returncode != 0:
        raise EnvResolutionError(f"{python_exe} exited {result.returncode}: {result.stderr.strip()}")
    lines = result.stdout.strip().splitlines()
    prefix, version = lines[0], lines[1]
    return ResolvedEnv(prefix=prefix, python=python_exe, version=version)


def _from_explicit(path: str) -> ResolvedEnv | None:
    if not path:
        return None
    p = Path(path)
    if not p.exists():
        raise EnvResolutionError(f"--python path does not exist: {path}")
    return _probe(str(p))


def _from_env_var() -> ResolvedEnv | None:
    venv = os.environ.get("VIRTUAL_ENV")
    if not venv:
        return None
    exe = _venv_python(venv)
    if not exe:
        raise EnvResolutionError(f"$VIRTUAL_ENV={venv} has no python executable")
    return _probe(exe)


def _from_local_venv(workdir: str) -> ResolvedEnv | None:
    for name in (".venv", "venv"):
        candidate = Path(workdir) / name
        if candidate.is_dir():
            exe = _venv_python(str(candidate))
            if exe:
                return _probe(exe)
    return None


def _from_poetry(workdir: str) -> ResolvedEnv | None:
    try:
        result = subprocess.run(
            ["poetry", "env", "info", "--path"],
            cwd=workdir, capture_output=True, text=True, timeout=15,
        )
        if result.returncode == 0:
            path = result.stdout.strip()
            exe = _venv_python(path)
            if exe:
                return _probe(exe)
    except FileNotFoundError:
        pass
    return None


def _from_uv(workdir: str) -> ResolvedEnv | None:
    # uv creates .venv by default in the project directory.
    venv = Path(workdir) / ".venv"
    if venv.is_dir():
        exe = _venv_python(str(venv))
        if exe:
            return _probe(exe)
    return None


def _from_pdm(workdir: str) -> ResolvedEnv | None:
    try:
        result = subprocess.run(
            ["pdm", "venv", "--path", "in-project"],
            cwd=workdir, capture_output=True, text=True, timeout=15,
        )
        if result.returncode == 0:
            path = result.stdout.strip()
            exe = _venv_python(path)
            if exe:
                return _probe(exe)
    except FileNotFoundError:
        pass
    return None


def _venv_python(venv_dir: str) -> str | None:
    """Return the python executable path inside a venv directory."""
    for rel in ("bin/python", "bin/python3", "Scripts/python.exe", "Scripts/python"):
        p = Path(venv_dir) / rel
        if p.exists():
            return str(p)
    return None
