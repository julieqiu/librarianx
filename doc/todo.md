# TODO

This document tracks tasks and design decisions that need to be addressed.

## Commands

- [ ] Implement `librarian disable <library> <issue-url> <reason>` - Add a command to disable a library's generation. The command should add `disabled: true` to the library configuration and automatically insert a comment with the issue URL and reason. This makes disabling libraries easier and ensures the issue link requirement is always met. The inverse command `librarian enable <library>` should remove the `disabled` field and the associated comment.

## Configuration

- [ ] Rethink `GenerateDefaults` - The current design for repository-level defaults may need reconsideration. Evaluate whether defaults should be more granular, whether they should apply differently to different types of libraries, or if the current three fields (transport, rest_numeric_enums, release_level) are sufficient.
