# TODO

This document tracks tasks and design decisions that need to be addressed.

## Commands

- [ ] Implement `librarian disable <library> <issue-url> <reason>` - Add
  a command to disable a library's generation.
  The command should add `disabled: true` to the library configuration and
  automatically insert a comment with the issue URL and reason.
  This makes disabling libraries easier and ensures the issue link requirement is always met.
  The inverse command `librarian enable <library>` should remove the `disabled`
  field and the associated comment.
- [ ] internal/release/rust
- [ ] internal/release/python
- [ ] internal/release/go
- [ ] https://github.com/googleapis/librarian/commit/99e1dc14d2d25f4a4ed777fd04c3324d2646a5b6
- [ ] caching of tarball
- [ ] caching of tree traversal
- [ ] error handling:
    - Unknown library paths in explicit mode → error
    - Unknown overrides in wildcard mode → warning (ignored)
    - Invalid 'keep' patterns → error
    - Missing or malformed googleapis tarball → error


## Configuration

- [ ] Rethink `GenerateDefaults` - The current design for repository-level
  defaults may need reconsideration.
  Evaluate whether defaults should be more granular,
  whether they should apply differently to different types of libraries,
  or if the current three fields (transport,
  rest_numeric_enums, release_level) are sufficient.
