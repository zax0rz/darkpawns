# Vendored and adapted from Capital One VulnHunter

Source: https://github.com/capitalone/vulnhunter @ 4042d34 (Apache-2.0;
full license in LICENSE). The methodology — SKILL.md structure and all
phase files — is upstream's work.

Adaptations for the ZCode harness (this file's raison d'être):
- Tool mapping header added to SKILL.md; phase files otherwise verbatim
  except one model-name example in phase4_report.md.
- Claude Code runtime references (model gate, /cost, install.sh error
  path, CLAUDE_SKILL_DIR variable) replaced with harness-neutral or
  Zcode equivalents in SKILL.md only.
- Phases and analysis methodology are upstream's; findings produced with
  this skill inherit upstream's intended scope (first-party production
  code only) and disclosure discipline.
