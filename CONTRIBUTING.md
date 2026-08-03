# Contributing

Thanks for helping out. lazyftp is a small project, so this is short.

## Picking something up

Comment on the issue you want and it will be assigned to you, so two people don't write the
same patch.

Issues labelled [`good first issue`](https://github.com/MawCeron/lazyftp/labels/good%20first%20issue)
are self-contained and don't need any prior knowledge of the codebase.

[ROADMAP.md](ROADMAP.md) says what each release is for and why the issues are ordered the way
they are. If what you have in mind isn't there, open an issue before writing code.

## Branches

Work lands on `develop`. Branch from it and target it with your pull request. `main` only moves
when a release is tagged.

## Before opening a pull request

```bash
go build ./...
go vet ./...
go test ./...
```

All three have to pass. There's no CI yet ([#20](https://github.com/MawCeron/lazyftp/issues/20)),
so this is the only check there is.

Leave a test behind for anything with real logic — a branch, a loop, a parser. One test that
fails if the logic breaks is enough; no frameworks or fixtures.

## Commits

Conventional commits, with the subject in English:

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
