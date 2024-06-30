# Contributing to go-path/di

Thanks for considering contributing to **go-path/di**! This guide has a few tips and guidelines to make contributing to the  project as easy as possible.

## Bug Reports

Any bugs (or things that look like bugs) can be reported on the [GitHub issue tracker](https://github.com/go-path/di/issues).

Make sure you check to see if someone has already reported your bug first! Don't fret about it; if we notice a duplicate 
we'll send you a link to the right issue!

## Feature Requests

If there are any features you think are missing from **go-path/di**, you can start a [Discussion](https://github.com/go-path/di/discussions).

Just like bug reports, take a peak at the issue tracker for duplicates before opening a new discussion.

## Documentation
[go-path/di documentation](https://go-path.github.io/di/) is built using [Docsify](https://docsify.js.org/#/), a fairly simple documentation generator.

## Working on go-path/di
To get started working on go-path/di, you'll need:
* Golang >= 1.21

You can run all of go-path/di tests with:

```sh
go test -v
```

## Pull Requests

Before starting a pull request, open an issue about the feature or bug. This helps us prevent duplicated and wasted effort. These issues are a great place to ask for help if you run into problems!

### Code Style

In short:

- **TAB** for indentation (https://pkg.go.dev/cmd/gofmt)

### Tests
When submitting a bug fix, create a test that verifies the broken behavior and that the bug fix works. This helps us avoid regressions!

When submitting a new feature, add tests for all functionality.

We'd like it to be as close to 100% as possible, but it's not always possible. Adding tests just for the purpose of getting coverage isn't useful; we should strive to make only useful tests!
