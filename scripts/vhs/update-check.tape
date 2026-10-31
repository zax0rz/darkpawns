Output docs/images/generated/update-check.gif
Set Shell "bash"
Set FontSize 18
Set Width 1200
Set Height 720
Set Theme { "name": "Dark Pawns Paper", "black": "#1A1614", "red": "#A8201A", "green": "#3A8A3A", "yellow": "#A8201A", "blue": "#56504A", "magenta": "#7A1812", "cyan": "#56504A", "white": "#EFE7D6", "brightBlack": "#56504A", "brightRed": "#7A1812", "brightGreen": "#3A8A3A", "brightYellow": "#A8201A", "brightBlue": "#56504A", "brightMagenta": "#7A1812", "brightCyan": "#56504A", "brightWhite": "#E5DAC1", "background": "#EFE7D6", "foreground": "#1A1614", "selection": "#E5DAC1", "cursor": "#A8201A" }
Set TypingSpeed 25ms
Hide
Type "cd /tmp/darkpawns-docs-demo/bundle"
Enter
Type "export TERM=xterm-256color COLORTERM=truecolor"
Enter
Type "unset NO_COLOR"
Enter
Type "clear"
Enter
Sleep 200ms
Type "./darkpawns --instance ../world update --archive ../candidate.tar.gz --sha256 {{SHA256}} --check"
Show
Sleep 400ms
Enter
Sleep 3s
