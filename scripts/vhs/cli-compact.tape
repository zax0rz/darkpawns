Output .local/cli-evidence/after/compact.gif
Set Shell "bash"
Set FontSize 16
Set Width 650
Set Height 540
Set Theme "Builtin Light"
Set TypingSpeed 10ms
Hide
Type "unset NO_COLOR"
Enter
Type "export TERM=xterm-256color COLORTERM=truecolor"
Enter
Type "clear"
Enter
Type "DP_CLI_PREVIEW=watch .local/cli-evidence/preview.test -test.run '^TestCLIPreview$'"
Enter
Show
Sleep 3s
Screenshot .local/cli-evidence/after/status-compact.png
Sleep 500ms
Type "?"
Sleep 1s
Screenshot .local/cli-evidence/after/help-compact.png
Sleep 500ms
Type "q"
Sleep 500ms
