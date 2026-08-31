# Regenerate assets/demo.gif:
#
#   go build -o assets/demo/lazyftp .
#   go build -o assets/sftpd/sftpd ./assets/sftpd
#   vhs assets/demo.tape
#   gifsicle -O3 -o assets/demo.gif assets/demo.gif
#
# assets/demo/ (the Local side) and assets/demo-remote/ (served over SFTP as
# the Remote side) are small fixture directories committed so the recording
# is reproducible without staging files by hand -- kept as siblings, not
# nested, so "remote" never shows up inside the Local panel's own listing.
# assets/sftpd is a throwaway SFTP server used only to record this -- see
# its own header -- so the GIF can show a real connect/upload/download
# instead of a disconnected file browser.
# --no-nerd-fonts keeps the recording legible on any machine, without
# depending on a patched font being installed where this is rerun.

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
Type "../sftpd/sftpd ../demo-remote & SFTPD_PID=$!; disown" Enter
Sleep 300ms
Type "clear" Enter
Show

Type "./lazyftp --no-nerd-fonts" Enter
Sleep 1200ms

# Fill in the connection form: SFTP is two Right presses past the FTP
# default, then Tab walks Host -> Port -> User -> Pass in visual order.
Right
Sleep 200ms
Right
Sleep 500ms
Tab
Type "127.0.0.1"
Sleep 300ms
Tab
Type "2022"
Sleep 300ms
Tab
Type "demo"
Sleep 300ms
Tab
Type "demo"
Sleep 500ms
Enter
Sleep 1800ms

# Connected: both panels are real now. Mark a local file and upload it.
Down 3
Sleep 500ms
Space
Sleep 400ms
Type "U"
Sleep 1800ms

# Switch to Remote, mark a file there, and download it back.
Tab
Sleep 500ms
Space
Sleep 400ms
Type "D"
Sleep 1800ms
Tab
Sleep 600ms

Hide
Ctrl+C
Type "kill $SFTPD_PID" Enter
Show
