"""
Static AST extraction of import/attribute/kwarg claims.
The source file is NEVER imported or executed — only ast.parse is used.
"""

import ast
import uuid
from dataclasses import dataclass, field
from typing import Optional


@dataclass
class Locus:
    file: str
    line: int
    col: int
    end_line: Optional[int] = None
    end_col: Optional[int] = None


@dataclass
class RawClaim:
    subtype: str          # "import" | "attribute" | "kwarg"
    text: str             # human-readable description
    locus: Locus
    module: str           # top-level module (e.g. "pandas")
    symbol: str           # dotted symbol path to introspect (e.g. "pandas.DataFrame.append")
    kwarg: Optional[str] = None   # the kwarg name, for subtype=="kwarg"
    id: str = field(default_factory=lambda: str(uuid.uuid4()))


def extract(source: str, filename: str) -> list[RawClaim]:
    """
    Parse source and return raw claims.
    - import claims: one per imported top-level module
    - attribute claims: attribute/method refs whose receiver traces to an import (direct, from-import, one-hop alias)
    - kwarg claims: keyword args on resolved calls
    Untraceable receivers → caller should mark as unverifiable.
    """
    try:
        tree = ast.parse(source, filename=filename)
    except SyntaxError:
        return []

    visitor = _ClaimVisitor(filename)
    visitor.visit(tree)
    return visitor.claims


class _ClaimVisitor(ast.NodeVisitor):
    def __init__(self, filename: str) -> None:
        self.filename = filename
        self.claims: list[RawClaim] = []
        # name → (module, dotted_path)  e.g. "pd" → ("pandas", "pandas")
        self._bindings: dict[str, tuple[str, str]] = {}

    # ── Import collection ─────────────────────────────────────────────────────

    def visit_Import(self, node: ast.Import) -> None:
        for alias in node.names:
            mod = alias.name                         # e.g. "pandas"
            top = mod.split(".")[0]
            local = alias.asname or mod              # e.g. "pd" or "pandas"
            self._bindings[local] = (top, mod)
            self.claims.append(RawClaim(
                subtype="import",
                text=f"import {mod}",
                locus=self._locus(node),
                module=top,
                symbol=mod,
            ))
        self.generic_visit(node)

    def visit_ImportFrom(self, node: ast.ImportFrom) -> None:
        if node.module is None:
            return
        top = node.module.split(".")[0]
        for alias in node.names:
            name = alias.name
            local = alias.asname or name
            dotted = f"{node.module}.{name}"
            self._bindings[local] = (top, dotted)
            self.claims.append(RawClaim(
                subtype="import",
                text=f"from {node.module} import {name}",
                locus=self._locus(node),
                module=top,
                symbol=dotted,
            ))
        self.generic_visit(node)

    # ── Attribute / method / kwarg extraction ─────────────────────────────────

    def visit_Attribute(self, node: ast.Attribute) -> None:
        resolved = self._resolve_name(node.value)
        if resolved is not None:
            module, base_symbol = resolved
            symbol = f"{base_symbol}.{node.attr}"
            self.claims.append(RawClaim(
                subtype="attribute",
                text=f"{symbol}",
                locus=self._locus(node),
                module=module,
                symbol=symbol,
            ))
        self.generic_visit(node)

    def visit_Call(self, node: ast.Call) -> None:
        # Collect kwarg claims for resolved calls.
        resolved_symbol = self._resolve_call_symbol(node.func)
        if resolved_symbol is not None:
            module, symbol = resolved_symbol
            for kw in node.keywords:
                if kw.arg is None:
                    continue   # **kwargs unpacking — not a static kwarg
                self.claims.append(RawClaim(
                    subtype="kwarg",
                    text=f"{symbol}({kw.arg}=...)",
                    locus=self._locus(kw.value),
                    module=module,
                    symbol=symbol,
                    kwarg=kw.arg,
                ))
        self.generic_visit(node)

    # ── Helpers ───────────────────────────────────────────────────────────────

    def _resolve_name(self, node: ast.expr) -> Optional[tuple[str, str]]:
        """
        Resolve a node to (module, dotted_symbol) if it traces to an import.
        Handles: Name (direct binding), Attribute (one-hop alias).
        Returns None for untraceable receivers.
        """
        if isinstance(node, ast.Name):
            return self._bindings.get(node.id)
        if isinstance(node, ast.Attribute):
            parent = self._resolve_name(node.value)
            if parent is not None:
                module, base = parent
                return (module, f"{base}.{node.attr}")
        return None

    def _resolve_call_symbol(self, func: ast.expr) -> Optional[tuple[str, str]]:
        """Resolve the callable in a Call node to (module, dotted_symbol)."""
        if isinstance(func, ast.Attribute):
            parent = self._resolve_name(func.value)
            if parent is not None:
                module, base = parent
                return (module, f"{base}.{func.attr}")
        if isinstance(func, ast.Name):
            binding = self._bindings.get(func.id)
            if binding:
                return binding
        return None

    def _locus(self, node: ast.AST) -> Locus:
        return Locus(
            file=self.filename,
            line=getattr(node, "lineno", 1),
            col=getattr(node, "col_offset", 0),
            end_line=getattr(node, "end_lineno", None),
            end_col=getattr(node, "end_col_offset", None),
        )
