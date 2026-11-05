# Round-two setup notes

- Worktree: /home/zach/wt/luna-idiom-r2
- Branch: luna/idiomatic-go-round2
- Code base SHA before the round-two brief commit: 5e9556a8789c4125bed8913d34d16e6f2ed68fac (round-one PR #2275 tip). No rebase.
- Setup commit: 92ccf5232, which copied BRIEF.md and PREDICTIONS.md from origin/main.
- Metrics command was run from the nested metrics module. baseline.tsv records 73,540 lines and 2,730 functions in scope.
- Focused linter returned exit 1 with existing findings: dupl 26, funlen 191, gocognit 111, gocyclo 4. Raw output: lint-baseline.txt.
- Installed gopls with go install golang.org/x/tools/gopls@latest. Version: golang.org/x/tools/gopls v0.24.0; the installer switched to Go 1.27.3 because that version requires Go >=1.27.
- Open PR file lists were refreshed from GitHub. PR #2275 owns six package files plus its run evidence; PR #2271 owns pkg/game/mobprogs.go, pkg/game/spec_procs4.go, and other listed files. See open-pr-files.md. Round-two code edits will avoid those paths.
- Round two uses Codex/GPT-6, as in round one. The pre-registered predictions are unchanged.
