# `llm/` — optional plain-English annotations

**Off by default.** When enabled (`--llm`, or `llm.enabled: true` in config), this layer asks
an LLM to write a one-line, human-friendly note explaining a finding.

The single most important rule in this folder:

> **The LLM never decides anything.** It runs *after* all verdicts are already frozen. It can
> only attach an advisory note (tagged `determinism: "subjective"`) to claims that are already
> `contradicted` or `unverifiable`. It can never create, change, or override a verdict, a
> count, or the final decision.

This is enforced by a CI test (the "honesty invariant"): toggling the LLM on and off must
produce byte-identical verdicts and summaries.

Providers: **Mistral** and **Grok (xAI)**, via one shared OpenAI-compatible client. API keys
come from environment variables (`MISTRAL_API_KEY`, `XAI_API_KEY`) — **never** from config
files. With no key, `--llm` simply prints a notice and falls back to deterministic output;
it never crashes.
