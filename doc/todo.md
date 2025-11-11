# TODO List

- [x] Add set / unset for `tag_format`, `generate.output`, etc.
- [x] Add librarian add <library> <api>...
- [x] Implement template expansion for `generate.output` with `{name}` and `{api.path}`
- [] Add librarian update googleapis
- [] Add librarian update --all
- [x] Add librarian generate <library> (command structure in place, needs generator integration)
- [] Add librarian release <library>
- [] Add librarian publish <library>


See [design-librarys.md](design-librarys.md) for the library paths and locations design.

## Decisions to make later

### Should librarian support releasing gcloud-mcp and other things?

Instead of support release for non-libraries, let's just make a different tool
that reuses the same logic. Then we can call it a library.

librarian generate <library>
librarian release <library>

<something> release library

Can use the same config structure, but instead of calling it librarian.yaml we
call it something else
