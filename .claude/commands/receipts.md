---
description: Fact-check Python code against installed packages and show a plain-English review
argument-hint: [file.py | "paste code" | (nothing = check the current diff)]
allowed-tools: Bash(go build:*), Bash(./receipts*), Bash(git diff:*), Read, Write
---

You are the **review layer between AI-written code and the human**. Your job: run
Receipts over the Python in question and hand the human a plain-English verdict so
they never have to trust code that calls something which does not exist.

## What to check

The argument is: `$ARGUMENTS`

- If it is a path to a `.py` file → check that file.
- If it is a snippet of code → write it to a temp `.py` file, then check that.
- If it is empty → check the Python files in the current git diff
  (`git diff --name-only -- '*.py'`). If the diff is empty, check staged files
  (`receipts check --staged`).

## How to run it

1. Build the CLI if `./receipts` (or `./receipts.exe` on Windows) is missing:
   `go build -o receipts ./cmd/receipts/`
2. Run with the **plain** format so the output is human-readable:
   `./receipts check --format plain --python python <file.py>`
   - On Windows use `.\receipts.exe` and `--python python`.
   - If the verifier fails to start, the user likely hasn't installed the plugin:
     tell them to run `pip install -e "verifiers/python-symbols"`.

## How to report back

Show the human the plain-English review verbatim, then add a one-line recommendation:

- Any **WRONG** (contradicted) item → say clearly: *do not ship this, here's what's
  broken and the likely fix*. Suggest the correct API if you know it
  (e.g. `df.append(...)` → `pd.concat([...])`).
- Only **?** (unverifiable) items → say it's probably fine but couldn't be fully
  checked, and why.
- All **OK** → say it checks out against the installed packages.

Never override Receipts' verdict. It checks the *installed* package; you are
explaining its result, not re-judging it. The exit code is the source of truth:
`0` = pass/warn, `1` = something is contradicted, `2` = tool error.
