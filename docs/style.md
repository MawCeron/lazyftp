# Code style

What a patch is expected to look like. `gofmt` settles most of it; this covers the rest, and
every rule here is one the current code already follows.

None of it is checked by a linter. There is no `.golangci.yml` and no plan for one — the project
is small enough that review catches these, and a linter that has to be configured is another file
to keep current.

## Formatting

`gofmt` decides. Tabs, its brace placement, its alignment, no arguments.

There is no line-length limit. The longest line in the project is 137 characters and it is an
error message; nothing else comes near 100. If a line is long because of nesting rather than
because of a string, the nesting is the problem.

`go vet` runs in CI and its findings are not suggestions. It has already caught a real defect
here: `fmt.Sprintf("%s:%d", host, port)` produces an address unusable for IPv6, which is why
`SFTPClient.Connect` uses `net.JoinHostPort`.

## Comments

**Code is not explained or justified in comments. It explains itself.** A comment exists to record
what the code *cannot* say. Rename the variable, split the function, drop the clever line — and
when none of that is enough, write the comment.

What earns one:

- **Library behaviour that is invisible at the call site.** `// Must be a seeker: goftp resumes an
  upload only when the source is one.`
- **A deliberate absence** — no buffer, no `InsecureSkipVerify`, no next tick asked for. Nothing
  in the code shows a thing that isn't there. `// Ticking stops by not asking for the next one.`
- **A constraint the type does not enforce.** `// It is called once the update loop is free, never
  from inside a blocking call.`
- **Why the long way instead of the obvious one.** `// Neither client takes a context, so the
  attempt is let go of rather than cancelled: it ends on its own timeout.`

What does not:

- Repeating the function's name in a sentence.
- Narrating what the next line does.
- Comparing the code to whatever was there before. That is what the commit is for.

Comments are about two percent of the code by line, and that is the intended density, not an
accident of hurry. `internal/model` has none at all and needs none.

Exported identifiers get a doc comment when what they are for is not obvious from the name —
`LineBuffer` has a long one because nothing about the name explains why it exists. `Drain` has a
one-liner. `Protocol.Next` has none.

## Naming

**Receivers are one letter, matching the type**: `a App`, `c *FTPClient`, `p Panel`, `m *Manager`,
`b *LineBuffer`. When two types in a package would collide, the newer one gives way.

**Message handlers on `App` are `handleX(msg XMsg)`**, one per message type. The type switch in
`Update` stays a dispatch table and the logic lives in the handler — `handleConnect`,
`handleNavigate`, `handleTransfer`.

**Test names are sentences about behaviour**, not the name of the function under test:

```go
func TestAnAbandonedAttemptDoesNotConnect(t *testing.T)
func TestRemotePathsStayPOSIX(t *testing.T)
func TestAPanickingTransferDoesNotTakeTheProcessDown(t *testing.T)
```

A failing test should say what broke from its name alone, before anyone reads the assertion.

**Messages are unexported** unless they cross a package boundary, in which case they belong in
`internal/shared`. Some existing UI messages are exported and shouldn't be; don't take that as
precedent.

## Values and pointers

The split is not a preference, and picking the wrong side produces bugs that look like something
else.

**UI types are values.** `App`, `Panel`, `ConnectionBar`, `ProcessesPanel`, `LogPanel` all take
value receivers, and a method that changes one returns a new one: `WithFiles`, `SetSize`,
`AddTransfer`, `MarkError`. Bubble Tea keeps whatever `Update` returned; a pointer receiver here
mutates a copy the loop is about to discard, or a value the loop has already replaced. Either way
the change appears to work and then vanishes.

**Types holding a live resource are pointers.** `FTPClient`, `SFTPClient`, `Manager`,
`LineBuffer`, `ProgressReader` own a connection, a mutex or a running count — copying one is
either meaningless or a race.

A new type that doesn't clearly belong to the second group belongs to the first.

## Errors

Lowercase, no trailing punctuation, wrapped with `%w`, and naming the thing that failed:

```go
return fmt.Errorf("unable to connect to %s: %w", c.host, err)
return nil, fmt.Errorf("error listing %s: %w", path, err)
```

A path, a host, a filename — an error that says only `permission denied` sends the reader back to
the code to find out which file.

Name the operation, without an `error` prefix. Much of the existing code starts its messages with
`error …`, which reads badly once wrapping stacks them up — `error listing /pub: error opening
local file: …` — so new code says `listing /pub: %w`.

**The `client` package returns errors and never prints.** What the user is told, and whether it
reaches the log panel or a file, is the UI's decision.

## Tests

No framework and no fixtures. `testify` appears in `go.sum` as an indirect dependency of something
else; no file in this project imports it, and none should.

Leave a test behind for anything with real logic — a branch, a loop, a parser. One test that fails
if the logic breaks is enough. Trivial code needs none.

Two patterns cover almost everything:

**`stubClient`** in `internal/ui/app_test.go` implements `client.Client` with canned returns, so
the update loop and the transfer manager can be exercised with no network. Anything added to the
interface goes there too, or the package stops compiling.

**Driving the model with messages.** An `App` is a value: construct one, send messages through
`Update`, assert on what comes back. No terminal and no `tea.Program` required.

Anything that needs a live server is verified by hand against one, and said plainly when it has
not been.

## Dependencies

Seven direct dependencies, three of them Charm's. An eighth needs to be something the standard
library cannot do and a few lines here cannot either.

Replacing one is a decision with a reason: the FTP client moved from `jlaffaye/ftp` to
`secsy/goftp` because connections through NAT did not work, and the commit says so.

## What is not here

[architecture.md](architecture.md) has the rules that break things rather than merely reading
badly — what may not run inside `Update`, where paths must not share code, which goroutines need
their own `recover`. Read those before a first patch; this file can wait until review.

[CONTRIBUTING.md](CONTRIBUTING.md) covers picking up an issue, branching, commit messages and what
is in scope for the project at all.
