Output .local/cli-evidence/after/launcher.gif
Set Shell "bash"
Set FontSize 16
Set Width 1100
Set Height 760
Set Theme "Builtin Dark"
Set TypingSpeed 25ms
Hide
Type "unset NO_COLOR"
Enter
Type "export TERM=xterm-256color COLORTERM=truecolor"
Enter
Type "clear"
Enter
Show
Type ".local/cli-evidence/after/darkpawns --instance .local/cli-evidence/instance doctor"
Enter
Sleep 3s
Hide
Type "clear"
Enter
Show
Type "DP_CLI_PREVIEW=watch .local/cli-evidence/preview.test -test.run '^TestCLIPreview$'"
Enter
Sleep 3s
Screenshot .local/cli-evidence/after/status.png
Sleep 500ms
Type "?"
Sleep 1s
Screenshot .local/cli-evidence/after/help.png
Sleep 500ms
Type "?"
Type "f"
Down
Sleep 1s
Type "f"
Sleep 1s
Type "q"
Sleep 500ms
Hide
Type "clear"
Enter
Show
Type "DP_CLI_PREVIEW=progress .local/cli-evidence/preview.test -test.run '^TestCLIPreview$'"
Enter
Sleep 2500ms
Screenshot .local/cli-evidence/after/progress.png
Sleep 500ms
Sleep 4s
