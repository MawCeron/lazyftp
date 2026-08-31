# Regenerate assets/demo.gif:
#
#   go build -o assets/demo/lazyftp .
#   vhs assets/demo.tape
#   gifsicle -O3 -o assets/demo.gif assets/demo.gif
#
# assets/demo/ (besides the gitignored lazyftp binary this builds) is a small
# fixture project committed so the recording is reproducible without staging
# files by hand. --no-nerd-fonts keeps the recording legible on any machine,
# without depending on a patched font being installed where this is rerun.

Set Shell "bash"
Set FontSize 20
Set Width 1400
Set Height 900
Set Padding 24
Set Theme "Catppuccin Mocha"
Set CursorBlink false
Set TypingSpeed 40ms

Output assets/demo.gif

Hide
Type "cd assets/demo && clear" Enter
Show

Type "./lazyftp --no-nerd-fonts" Enter
Sleep 1500ms

# lazyftp opens on the connection dialog; Esc without a host typed leaves it
# without attempting to connect and moves focus to the Local panel.
Escape
Sleep 800ms

Down 2
Sleep 700ms

# Cycle the sort column: Name -> Size -> Date.
Type "s"
Sleep 900ms
Type "s"
Sleep 900ms

# Filter down to one match, read the counter, then clear it.
Type "/site"
Sleep 1000ms
Escape
Sleep 400ms

# Jump straight to a path by typing it.
Type ":"
Type "content"
Enter
Sleep 1200ms
Type "h"
Sleep 1200ms
