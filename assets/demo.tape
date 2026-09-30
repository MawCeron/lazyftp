# Regenerate assets/demo.gif:
#
#   go build -o assets/bin/lazyftp .
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
# assets/demo-env.sh gives the recording a throwaway HOME, with an ssh_config
# that names the server, an SSH key the server accepts, a theme, and copies of
# the two fixture directories; nothing in the real ~/.ssh or lazyftp config is
# read or written, and the transfers leave nothing behind in the repo.
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
Type "cd assets && source demo-env.sh && clear" Enter
Sleep 1500ms
Show

# A name from ~/.ssh/config on the command line: no form to fill in. The server
# is new to this HOME, so lazyftp shows its fingerprint and asks before trusting it.
Type "lazyftp --no-nerd-fonts demo" Enter
Sleep 2500ms
Type "y"
Sleep 2500ms

# Connected, with the SSH key: the Log says which method worked. Dotfiles are
# hidden until Ctrl+H.
Ctrl+H
Sleep 1800ms
Ctrl+H
Sleep 600ms

# Mark a local file and upload it.
Down 3
Sleep 500ms
Space
Sleep 400ms
Type "U"
Sleep 1800ms

# Switch to Remote, mark a file there, and download it back. The cursor stays
# on its file when the panel reloads.
Tab
Sleep 500ms
Space
Sleep 400ms
Type "D"
Sleep 1800ms

# Create and delete a directory on the remote side; each leaves a line in the Log.
Ctrl+N
Sleep 400ms
Type "drafts"
Sleep 400ms
Enter
Sleep 1600ms

# Up to the new directory, then delete it: the Log counts what was deleted.
Up
Sleep 500ms
Type "d"
Sleep 900ms
Enter
Sleep 1800ms
Tab
Sleep 600ms

Hide
Ctrl+C
Type "kill $SFTPD_PID" Enter
Sleep 500ms
Show
