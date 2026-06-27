# Receipts, in plain English

This page is for everyone — you do not need to be a programmer to read it.

## What problem does this solve?

AI tools (ChatGPT, Claude, Copilot, etc.) write Python code for people. Most of the
time it works. But sometimes the AI **makes up** a function that sounds real but does
not exist, or uses one that was **removed** from a library in a newer version. When
that code runs, it crashes.

The frustrating part: the code *looks* perfectly fine. A human reviewing it can't tell
that `pandas.DataFrame.append` was deleted in 2023 just by reading it.

**Receipts checks the AI's code against the packages you actually have installed,
before you run it.** It's a spell-checker for AI code — but instead of spelling, it
checks "does this function really exist?"

## The three answers it gives

For every function, method, or option the code uses, Receipts gives one of three answers:

| Word it uses    | Plain meaning                                                        |
|-----------------|---------------------------------------------------------------------|
| **grounded**    | ✅ Real and correct. This exists in your installed package.          |
| **contradicted**| ❌ Wrong. This does NOT exist — it was removed, renamed, or made up. |
| **unverifiable**| ⚠️ Couldn't check it. Maybe the package isn't installed. Not proof of a problem. |

It is **careful on purpose**: if it isn't *sure* something is wrong, it says
"unverifiable" (couldn't check), never "wrong". It would rather stay quiet than cry
wolf — because a tool that raises false alarms gets ignored.

It also **never runs your code**. It only reads it, the way you'd proofread a letter
without mailing it.

## How to read the report

Run it with the friendly format and you get a report anyone can follow:

```
.\receipts.exe check --format plain --python python yourfile.py
```

Example output:

```
Receipts — plain-English review
================================

File checked:    yourfile.py
Checked against: your installed python packages

What I found:
  OK      4  - confirmed real and correct
  WRONG   1  - uses something that does NOT exist
  ?       0  - couldn't be checked (not necessarily wrong)

Details:

  [WRONG] pandas.DataFrame.append   (line 6)
         This uses a name that isn't in the installed package - it was likely
         removed or renamed. Running it would crash with an AttributeError.
         technical detail: 'pandas.DataFrame.append' not found

Bottom line: DO NOT use this code yet. It calls something that does not exist
and will crash. Fix the WRONG items above first.
```

- **OK / WRONG / ?** are just the friendly names for grounded / contradicted / unverifiable.
- The **technical detail** line is the exact reason, kept for developers.
- The **Bottom line** tells you what to actually do.

## Using it inside an AI assistant (the "/" command)

If you cloned this project and opened it in an AI coding client (like Claude Code),
you can type:

```
/receipts yourfile.py
```

The assistant will run the check for you and show you the plain-English review — so
**every time the AI writes code, there's an automatic fact-check between the AI and
you.** You decide what to do with code only *after* seeing whether it's real.

There is also an **MCP tool** (`receipts_check_code`) that an AI can call on its own,
silently, before it ever shows you code. If it finds something made-up, a well-behaved
assistant fixes the code first — you only see the clean version.

## What was added in this round (for the curious)

This round focused on **making the tool understandable to normal people**, not just
developers:

1. **A plain-English report mode** (`--format plain`). The old output used words like
   "grounded" and "AttributeError". The new mode explains every finding in one normal
   sentence and ends with a clear "do this / don't do this" line.
2. **A `/receipts` command** so anyone who clones the project can fact-check code from
   inside their AI assistant by typing one line — no setup knowledge needed.
3. **The AI ↔ human review layer.** The MCP tool now returns the plain-English review
   *first* (for you) and the technical report *second* (for tooling), so the human in
   the loop always gets the readable version.
4. **This guide.**

None of this changes the actual verdicts. The friendly wording sits *on top of* the
same careful, deterministic checks — it explains the result, it never decides it.
