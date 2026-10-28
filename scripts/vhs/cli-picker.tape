Output .local/cli-evidence/after/picker-and-confirm.gif
Set Shell "bash"
Set FontSize 16
Set Width 1000
Set Height 650
Set Theme "Builtin Dark"
Set TypingSpeed 30ms
Hide
Type "unset NO_COLOR"
Enter
Type "export TERM=xterm-256color COLORTERM=truecolor"
Enter
Type "clear"
Enter
Show
Type "DP_CLI_PREVIEW=instances .local/cli-evidence/preview.test -test.run '^TestCLIPreview$'"
Enter
Sleep 1s
Type "/stag"
Sleep 1s
Screenshot .local/cli-evidence/after/instances.png
Sleep 500ms
Enter
Enter
Sleep 500ms
Hide
Type "clear"
Enter
Show
Type "DP_CLI_PREVIEW=confirm .local/cli-evidence/preview.test -test.run '^TestCLIPreview$'"
Enter
Sleep 1s
Screenshot .local/cli-evidence/after/confirm.png
Sleep 500ms
Tab
Enter
Sleep 500ms
Hide
Type "clear"
Enter
Show
Type "DP_CLI_PREVIEW=check .local/cli-evidence/preview.test -test.run '^TestCLIPreview$'"
Enter
Sleep 2s
Screenshot .local/cli-evidence/after/release-notes.png
Sleep 500ms
