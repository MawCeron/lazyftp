# Contributing

Thanks for helping out. lazyftp is a small project, so this is short.

## Picking something up

Comment on the issue you want and it will be assigned to you, so two people don't write the
same patch.

Issues labelled [`good first issue`](https://github.com/MawCeron/lazyftp/labels/good%20first%20issue)
are self-contained and don't need any prior knowledge of the codebase.

[ROADMAP.md](../ROADMAP.md) says what each release is for and why the issues are ordered the way
they are. If what you have in mind isn't there, open an issue before writing code.

[architecture.md](architecture.md) is the map of the codebase: where things live, how a
keystroke becomes a transfer, and the rules that are easy to break. Its first two sections are
enough for most changes.

[style.md](style.md) says what a patch is expected to look like — comments, naming, errors, tests.

## Branches

Work lands on `develop`. Branch from it and target it with your pull request. `main` only moves
when a release is tagged, by fast-forward, so the history stays a straight line. A release that is
still far off, such as v0.5.0, is built on a `features/` branch until it is ready to join
`develop`; if you are not sure where a change belongs, ask in the issue.

Pull requests are merged with "Rebase and merge". A squash would flatten your commits and a merge
commit would break the line, so merge commits are switched off in the repository. Keep one logical
change per commit.

## Before opening a pull request

```bash
go build ./...
go vet ./...
go test ./...
test -z "$(gofmt -l .)"
```

All four have to pass, with Go 1.26.10 or later: the standard library of earlier 1.26 releases has
vulnerabilities, and `go.mod` requires the patched one. CI runs them on Linux and Windows for every
pull request, so running them first saves a round trip. It also runs `govulncheck`, which fails
when a vulnerability is reachable from the code.

Leave a test behind for anything with real logic — a branch, a loop, a parser. One test that
fails if the logic breaks is enough; no frameworks or fixtures.

## Commits

Conventional commits, with the subject in English and no scope: `fix:`, not `fix(ui):`.

```
fix: correct visible rows calculation in processes panel
```

If the change needs explaining, use the body to say **what** changed and **why**. Not how — the
diff already shows that.

## Scope

lazyftp moves files between local and remote, from the keyboard. Changes that turn it into a
different program — a file viewer, an rsync, a session manager — get closed, and several already
have. Anything a user must configure in order to connect is treated as a bug rather than an
option.

If you're unsure whether an idea fits, open an issue and ask before building it.
