# Librarianx

Librarianx is a tool for managing Google Cloud client libraries across multiple
languages. It handles code generation from API definitions, version management,
and publishing to package registries.

This tour walks through realistic workflows for Go, Python, and Rust libraries.
You'll see how to set up a repository, generate your first library, handle
updates, and publish releases.

## Installation

Start by installing librarianx:

```
$ go install github.com/julieqiu/librarianx@latest
```

## Your First Library: Go Secret Manager

Let's build a Go client library for Google Cloud Secret Manager. First,
create a workspace:

```
$ mkdir libraries
$ cd libraries
$ mkdir google-cloud-go
$ cd google-cloud-go
```

### Initialize the Repository

Initialize a Go repository with `librarianx init`:

```
$ librarianx init go
Created librarian.yaml
```

This creates a repository configuration file. Let's see what's inside:

```
$ cat librarian.yaml
version: v0.5.0
language: go

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/9fcfbea0aa5b50fa22e190faceb073d74504172b.tar.gz
    sha256: 81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98

container:
  image: us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/go-librarian-generator
  tag: latest

generate:
  output_dir: ./

release:
  tag_format: '{id}/v{version}'
```

The config defines:
- `sources` - External source repositories (googleapis)
- `container` - Container image for code generation
- `generate` - Generation configuration (output directory, defaults)
- `release` - How to format release tags

### Install Dependencies

Before generating code, install the Go generator dependencies. You can either
install them locally or use a Docker container:

```
$ librarianx install go --use-container
Using Docker container for code generation
Container image: us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/go-librarian-generator:latest
```

The `--use-container` flag ensures consistent generation across different
environments. You can omit it to install dependencies locally instead.

### Create Your First Library

Create the Secret Manager library:

```
$ librarianx new secretmanager google/cloud/secretmanager/v1
Parsing googleapis BUILD.bazel files...
Created library entry in librarian.yaml
Downloading googleapis...
Running generator container...
Generated secretmanager/
```

This command:
1. Downloads googleapis (if needed)
2. Reads `google/cloud/secretmanager/v1/BUILD.bazel` to extract configuration
3. Creates an library entry in `librarian.yaml`
4. Generates the code immediately

Notice that Go uses directory names without prefixes (secretmanager, not
google-cloud-secretmanager). This matches Go module conventions.

Let's look at what was created in the configuration:

```
$ cat librarian.yaml
# ... (top-level config)

librarys:
  - name: secretmanager
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
          grpc_service_config: secretmanager_grpc_service_config.json
          service_yaml: secretmanager_v1.yaml
          transport: grpc+rest
          name_pretty: "Secret Manager"
          product_documentation: "https://cloud.google.com/secret-manager/docs"
          release_level: "stable"
```

All the protoc configuration was extracted from BUILD.bazel and saved as an library entry.

Let's see what was created:

```
$ ls secretmanager/
apiv1/
  secret_manager_client.go
  secret_manager_client_test.go
go.mod
go.sum
README.md
```

Your first client library is ready!

### Add Another API Version

Secret Manager has a beta API. You can add it by manually editing the config or recreating:

```
$ nano librarian.yaml
# Find the secretmanager library and add google/cloud/secretmanager/v1beta2 to the apis list
```

The library now has two APIs:

```
$ grep "path:" librarian.yaml | grep secretmanager
    - path: google/cloud/secretmanager/v1
    - path: google/cloud/secretmanager/v1beta2
```

Regenerate to include the beta API:

```
$ librarianx generate secretmanager
Running generator container...
Generated secretmanager/
```

Now you have both stable and beta APIs in one module:

```
$ ls secretmanager/
apiv1/
  secret_manager_client.go
apiv1beta2/
  secret_manager_client.go
go.mod
go.sum
README.md
```

### Release Your Library

First, commit your changes:

```
$ git add .
$ git commit -m "feat(secretmanager): add Secret Manager client library"
```

See what would be released (dry-run mode):

```
$ librarianx release secretmanager
Analyzing secretmanager for release...

Pending changes since last release:
  feat(secretmanager): add Secret Manager client library

Proposed version: null → 0.1.0 (initial release)

Would perform:
  ✓ Update secretmanager/internal/version.go: set to 0.1.0
  ✓ Create secretmanager/CHANGELOG.md
  ✓ Create commit: chore(release): secretmanager v0.1.0
  ✓ Create git tag: secretmanager/v0.1.0
  ✓ Push tag to origin
  ✓ Publish to pkg.go.dev (auto-indexed from tag)

To proceed, run:
  librarianx release secretmanager --execute
```

Actually perform the release:

```
$ librarianx release secretmanager --execute
Releasing secretmanager...

✓ Updated secretmanager/internal/version.go: 0.1.0
✓ Created secretmanager/CHANGELOG.md
✓ Created commit: chore(release): secretmanager v0.1.0
✓ Created tag: secretmanager/v0.1.0
✓ Pushed tag to origin
✓ Published to pkg.go.dev

Release complete!
Track indexing: https://pkg.go.dev/cloud.google.com/go/secretmanager/apiv1
```

For Go, publishing happens automatically when you push git tags. pkg.go.dev
indexes the module automatically.

## Adding More Librarys

Let's add Access Approval to our Go repository:

```
$ librarianx new accessapproval google/cloud/accessapproval/v1
Created library entry in librarian.yaml
Generated accessapproval/
```

### Updating Everything

Time passes. You want to update to the latest googleapis and regenerate all
librarys. This is common when googleapis adds new methods or fixes bugs.

Update the googleapis source:

```
$ librarianx update --googleapis
Fetching latest googleapis commit...
Updated librarian.yaml:
  sources.googleapis.url: https://github.com/googleapis/googleapis/archive/a1b2c3d4...tar.gz
  sources.googleapis.sha256: 867048ec8f0850a4d77ad836319e4c0a0c624928611af8a900cd77e676164e8e
```

Regenerate all librarys:

```
$ librarianx generate --all
Generated secretmanager/
Generated accessapproval/
```

Commit the changes:

```
$ git add .
$ git commit -m "feat: update to googleapis a1b2c3d4"
```

Release everything that changed:

```
$ librarianx release --all
Analyzing all librarys for release...

Found 2 librarys with pending releases:
  - secretmanager: 0.1.0 → 0.2.0 (minor - new features)
  - accessapproval: null → 0.1.0 (initial release)

Would perform releases for both librarys.
To proceed, run:
  librarianx release --all --execute
```

Execute the release:

```
$ librarianx release --all --execute
Releasing 2 librarys...

✓ Released secretmanager v0.2.0
✓ Released accessapproval v0.1.0

Done!
```

## Python Librarys

Let's try Python. Python requires installing dependencies before generation.

```
$ cd ../
$ mkdir google-cloud-python
$ cd google-cloud-python
```

Initialize a Python repository:

```
$ librarianx init python
Created librarian.yaml
```

Python projects typically use a `packages/` directory for generated librarys.
Edit the config to set this:

```
$ nano librarian.yaml
# Set generate.dir to packages/
```

Install Python generator dependencies:

```
$ librarianx install python --use-container
Using Docker container for code generation
Container image: us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/python-librarian-generator:latest
```

Create Secret Manager library:

```
$ librarianx new google-cloud-secret-manager google/cloud/secretmanager/v1 google/cloud/secretmanager/v1beta2
Created library entry in librarian.yaml
Generated packages/google-cloud-secret-manager/
```

Notice Python uses package names with prefixes (google-cloud-secret-manager).
This matches PyPI naming conventions.

Check the output:

```
$ ls packages/google-cloud-secret-manager/
google/
  cloud/
    secretmanager_v1/
      __init__.py
      services/
      types/
    secretmanager_v1beta2/
tests/
setup.py
README.rst
```

Release workflow is similar to Go, but publishes to PyPI:

```
$ git add .
$ git commit -m "feat(secretmanager): add Secret Manager Python client"
$ librarianx release google-cloud-secret-manager --execute
Releasing google-cloud-secret-manager...

✓ Updated version to 0.1.0
✓ Created CHANGELOG.md
✓ Created commit: chore(release): google-cloud-secret-manager v0.1.0
✓ Created tag: google-cloud-secret-manager/v0.1.0
✓ Pushed tag to origin
✓ Built distribution
✓ Uploaded to PyPI

Release complete!
Published: https://pypi.org/project/google-cloud-secret-manager/0.1.0/
```

## Rust Librarys

Rust works similarly:

```
$ cd ../
$ mkdir google-cloud-rust
$ cd google-cloud-rust
```

Initialize a Rust repository:

```
$ librarianx init rust
Created librarian.yaml
```

Rust typically uses a `generated/` directory. Edit the config:

```
$ nano librarian.yaml
# Set generate.dir to generated/
```

Install Rust generator dependencies:

```
$ librarianx install rust --use-container
Using Docker container for code generation
Container image: us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/rust-librarian-generator:latest
```

Create librarys:

```
$ librarianx new secretmanager google/cloud/secretmanager/v1
Created library entry in librarian.yaml
Generated generated/google-cloud-secretmanager-v1/

$ librarianx new accessapproval google/cloud/accessapproval/v1
Created library entry in librarian.yaml
Generated generated/google-cloud-accessapproval-v1/
```

Check the output:

```
$ ls generated/google-cloud-secretmanager-v1/
src/
  lib.rs
  client.rs
  types.rs
Cargo.toml
README.md
```

Release workflow is similar, but publishes to crates.io:

```
$ git add .
$ git commit -m "feat: add Rust client librarys"
$ librarianx release --all --execute
Releasing 2 librarys...

✓ Released google-cloud-secretmanager-v1 v0.1.0
  - Ran cargo semver-checks
  - Published to crates.io
  - https://crates.io/crates/google-cloud-secretmanager-v1/0.1.0

✓ Released google-cloud-accessapproval-v1 v0.1.0
  - Ran cargo semver-checks
  - Published to crates.io
  - https://crates.io/crates/google-cloud-accessapproval-v1/0.1.0

Done!
```

## Working with Handwritten Code

Not all code needs to be generated. You can use librarian just for release
management of handwritten code.

Go back to the Go repository and create some handwritten librarys:

```
$ cd ../google-cloud-go
$ mkdir -p storage
$ mkdir -p pubsub
$ echo "package storage\n\nfunc NewClient() {}" > storage/client.go
$ echo "package pubsub\n\nfunc NewClient() {}" > pubsub/client.go
```

Add them to librarian with the `--location` flag to specify where the code lives:

```
$ librarian add storage --location storage/
Added handwritten library "storage" at storage/

$ librarian add pubsub --location pubsub/
Added handwritten library "pubsub" at pubsub/
```

This creates library entries with explicit locations (handwritten librarys):

```
$ grep -A2 "name: storage" librarian.yaml
  - name: storage
    location: storage/

$ grep -A2 "name: pubsub" librarian.yaml
  - name: pubsub
    location: pubsub/
```

Notice there's no `apis` field - this tells librarian the code is handwritten
and doesn't need generation. The `location` field tells librarian where to
find the code for release purposes.

Now you can release them like any other library:

```
$ git add .
$ git commit -m "feat: add storage and pubsub"
$ librarian release storage --execute
Releasing storage...

✓ Updated version to 0.1.0
✓ Created CHANGELOG.md
✓ Created commit: chore(release): storage v0.1.0
✓ Created tag: storage/v0.1.0
✓ Pushed tag to origin

Release complete!

$ librarian release pubsub --execute
Releasing pubsub...

✓ Updated version to 0.1.0
✓ Created CHANGELOG.md
✓ Created commit: chore(release): pubsub v0.1.0
✓ Created tag: pubsub/v0.1.0
✓ Pushed tag to origin

Release complete!
```

## Language and Workflow Flexibility

Each `librarian.yaml` file is configured for a single language (Go, Python, or Rust).
However, you can use librarian for different workflows:

- **Generation + Release**: Generate code from APIs and manage releases (most common)
- **Release-only**: Manage releases of handwritten code without generation
- **Mixed**: Manage both generated and handwritten librarys in the same repository

For example, a Go repository might have:
- Generated librarys from googleapis APIs (with `apis` field)
- Handwritten tools or utilities (with `location` field)
- All released and versioned consistently

If you need to work with multiple languages, create separate repositories with
their own `librarian.yaml` files:

```
google-cloud-go/librarian.yaml      # language: go
google-cloud-python/librarian.yaml  # language: python
google-cloud-rust/librarian.yaml    # language: rust
```

## Summary

Librarianx provides a consistent workflow across languages:

1. **Initialize** - `librarianx init <language>`
2. **Install** - `librarianx install <language> --use-container`
3. **Create** - `librarianx new <name> <api-paths>` (creates and generates)
4. **Regenerate** - `librarianx generate <name>` or `librarianx generate --all`
5. **Test** - `librarianx test <name>` or `librarianx test --all`
6. **Update Sources** - `librarianx update --googleapis` or `librarianx update --all`
7. **Release** - `librarianx release <name>` (dry-run) or `librarianx release <name> --execute`

The same commands work for Go, Python, and Rust. Configuration lives in the
`librarian.yaml` file, making everything transparent and version-controlled.

Key differences by language:
- **Go**: Modules auto-publish to pkg.go.dev when tags are pushed
- **Python**: Uses `packages/` directory, publishes to PyPI
- **Rust**: Uses `generated/` directory, publishes to crates.io with semver checks

Try it out! Feedback and bug reports are welcome at
https://github.com/julieqiu/librarianx/issues.
