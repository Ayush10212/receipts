# `benchmark/` — proof that it actually works

A tool that catches hallucinations is only trustworthy if you can *measure* how well it does.
This folder is a **50-case corpus** of real Python snippets with known-correct answers, plus a
harness that runs Receipts over all of them and computes two numbers:

- **Catch rate** — of the genuinely broken cases, how many did we flag? Gate: **≥ 80%** (currently **100%**).
- **False-positive rate** — how often did we cry wolf on code that was actually fine? Gate: **< 10%** (currently **0%**).

Both are enforced as required checks in CI, so a change that makes Receipts dumber or more
paranoid fails the build.

The corpus (`cases/*.json`) covers removed NumPy aliases, removed pandas APIs, valid modern
APIs that must stay `grounded`, and tricky cases (import errors, C-extensions) that must come
back `unverifiable` — never a false `contradicted`.

```bash
# Run it (slow — it spawns a Python subprocess per case)
make benchmark
```

It's behind a `//go:build benchmark` tag so it stays out of the everyday `go test ./...`.
