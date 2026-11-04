# Security policy

## Reporting a vulnerability

Report it privately through GitHub: open the repository's **Security** tab and choose
**Report a vulnerability**, or go straight to
[the advisory form](https://github.com/MawCeron/lazyftp/security/advisories/new).

Please do not open a public issue or pull request for a vulnerability that has not been fixed yet.
Anyone can read them.

A good report has:

- the lazyftp version (`lazyftp --version`), the protocol (FTP, FTPS or SFTP) and your system
- the steps to reproduce it, or a server or file name that triggers it
- what an attacker gains

A log from `--log-file` helps, but read it first: see [What the log holds](#what-the-log-holds).

## What to expect

You get an answer within 7 days. Fixes go into a patch release, and the report is made public
once that release is out. If you want credit for the find, say how you want to be named.

## Supported versions

Only the latest release gets security fixes. If you run an older one, update.

## Passwords

Passwords live in the system keyring. They are not written to `config.toml`, and lazyftp never
takes one from the command line, so none ends up in your shell history.

## What the log holds

The file written by `--log-file` can contain host names, user names, local and remote paths, the
names of the files you transfer and the text servers send back. With `--verbose` it also holds the
FTP control dialogue, where the password is masked. From v0.4.2 the file is created readable only
by you. Remove what you do not want to share before attaching a log to a report.
