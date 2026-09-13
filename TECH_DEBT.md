# Technical debt

What is still open, what is deliberately left and what only looks like debt.

Every item in this file is a behaviour-preserving internal concern. Nothing here reverts a feature or
changes what the user sees. Read it against `ARCHITECTURE.md` and the structural tests, which are the
authority on the invariants an item might threaten.

Open items are numbered sections. A numbered heading is the definition of an open item, so a scan for
`## <number>.` is the machine check for whether this file is clear. The two standing sections below are
deliberately unnumbered and are not open items.

History is not recorded here. A resolved item is deleted outright, never rewritten as done and never
archived. A resolution worth remembering belongs in the release notes.

There is no open technical debt.

## Looks like debt, not worth touching

Nothing yet.

## Not debt (do not "fix" these)

**The absence of a GitHub Pages site.** This repository is private and the product may be sold, so
there is deliberately no public site and no Pages deployment. A documentation pass must not create one.

**The setup program's drawn header mark, now that the artwork exists.** The setup page still carries an
inline SVG mark behind the real icon; it swaps to the image only once that has actually loaded. That is
not a leftover. The page has no bundler, so it loads its icon as a plain file; if that file were ever
missing the drawing is what keeps the header looking finished rather than showing a broken image.

**The setup program holding no install logic of its own.** Every act the setup window performs goes
through `internal/infrastructure/setup`, which is where the extraction, the registry writes, the
shortcuts and the process handling live. `installer/app.go` decides only which screen to open and when
to refuse, such as while the application is running, then calls it. The portable half of `setup` is
unit tested; `installer` has no tests of its own, since every method on it acts on the machine.
