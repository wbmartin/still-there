# PS0005 — Security Considerations

Branch `TKT-PS0005-agent-guidance-consolidation`. Docs and agent guidance
only; no code, no config, no endpoints. No profile applies in the
mechanical sense; two points worth recording:

- **The explicit-security-checks rule is now in `CLAUDE.md`** ("called
  explicitly in each handler … not hidden in middleware. Non-negotiable").
  This is the control that keeps an agent from "helpfully" moving JWT /
  CSRF / authorization into middleware where an exception path is invisible
  at the handler. Recording it where every session reads it is the
  security value of this ticket.
- **`doc/900` no longer points at another application's schema file**
  (`/atticus-web-snipper/…`). An agent following the old line could have
  pulled column names — and by extension data-model assumptions — from a
  different product's schema. It now points at this repo's migrations.

`opencode.json` removal drops a config file that named two non-existent
instruction files; nothing read it. No secrets involved anywhere in this
change.

## What did I miss?

- `CLAUDE.md` grows ~150 words, read on every session start. Brandon's
  ceiling was "≤ 12 lines" of rules; this is six bullets. Measured, not
  guessed.
