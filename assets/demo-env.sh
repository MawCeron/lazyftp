# Sourced, from assets/, by the demo tapes. It gives the recording a throwaway
# HOME, so it never reads or writes the real ~/.ssh or lazyftp config, and works
# on copies of the fixtures, so the transfers leave nothing behind in the repo:
#
#   $HOME/demo          the Local side, a copy of assets/demo
#   $HOME/demo-remote   served by sftpd as the Remote side
#
# sftpd is started the way the recording needs it:
#
#   - key authentication: a new key pair is made in the temporary HOME, and its
#     public half is what sftpd accepts for the demo user;
#   - a host key kept in a file, so a restarted server is still the same server;
#   - the first connection dropped after DEMO_DROP_FIRST, if that is set, to
#     show a session that was cut being reopened.
#
# lazyftp is taken from assets/bin. SFTPD_PID is left set, and a watcher stops
# the server when this shell ends: VHS closes the session as soon as the last
# command is sent, too soon for the tape to stop it itself.

# A server left over from an earlier recording would answer with its own keys.
if (exec 3<>/dev/tcp/127.0.0.1/2022) 2>/dev/null; then
  echo "demo-env: port 2022 is already in use (a sftpd from an earlier recording?)" >&2
  return 1
fi

assets="$PWD"
export HOME="$(mktemp -d /tmp/d.XXXX)"
export PATH="$assets/bin:$PATH"
unset SSH_AUTH_SOCK

mkdir -p "$HOME/.ssh" "$HOME/.config/lazyftp" "$HOME/demo" "$HOME/demo-remote"
cp -r "$assets/demo/." "$HOME/demo"
cp -r "$assets/demo-remote/." "$HOME/demo-remote"
cp "$assets/demo-home/ssh_config" "$HOME/.ssh/config"
cp "$assets/demo-home/config.toml" "$HOME/.config/lazyftp/config.toml"
ssh-keygen -q -t ed25519 -N '' -C demo -f "$HOME/.ssh/id_ed25519"
cp "$HOME/.ssh/id_ed25519.pub" "$HOME/authorized_keys"

"$assets/sftpd/sftpd" -hostkey "$HOME/host_key" -authorized "$HOME/authorized_keys" \
  ${DEMO_DROP_FIRST:+-drop-first "$DEMO_DROP_FIRST"} "$HOME/demo-remote" >/dev/null 2>&1 &
SFTPD_PID=$!
( while kill -0 $$ 2>/dev/null; do sleep 1; done; kill "$SFTPD_PID" 2>/dev/null ) >/dev/null 2>&1 &
disown -a
cd "$HOME/demo"
