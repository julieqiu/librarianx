# Librarian

Librarian is a tool for managing Google Cloud client libraries across multiple
languages. It handles code generation from API definitions, version management,
and publishing to package registries.

This tour walks through realistic workflows for Go and Python libraries,
highlighting how to manage different types of libraries: fully generated,
fully handwritten, and hybrid (a mix of both).

## Installation

Start by installing librarian:

```
$ go install github.com/julieqiu/librarian@latest
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

Initialize a Go repository with `librarian init`:

```
$ librarian init go
Created librarian.yaml
```

This creates a repository configuration file. Let's see what's inside:

```
$ cat librarian.yaml
version: v1
language: go

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/main.tar.gz
    sha256: ...

# Default directory for newly generated libraries.
generate:
  dir: ./

release:
  tag_format: '{name}/v{version}'
```

The config defines the language, sources, release tag formats, and a default
directory (`generate.dir`) for generated libraries.

### Library Types

Librarian supports three types of libraries, distinguished by the keys used in
their `librarian.yaml` entry.

#### 1. Fully Handwritten

A library is **Fully Handwritten** if it only has `name` and `path`. The
generator will never touch this directory.

```yaml
- name: pubsub
  path: pubsub/
```

#### 2. Fully Generated

A library is **Fully Generated** if it has `name`, `path`, and a `generate` block.
The generator will delete and recreate the contents of the `path` directory on
every run.

```yaml
- name: secretmanager
  path: secretmanager/
  generate:
    apis:
      - path: google/cloud/secretmanager/v1
```

#### 3. Hybrid (Generated + Handwritten)

A library is a **Hybrid** if it has `name`, `path`, `generate`, and a `patch`
block. The `patch` block lists files and directories to protect from being
overwritten during generation.

```yaml
- name: bigquery
  path: bigquery/
  generate:
    apis:
      - path: google/cloud/bigquery/storage/v1
  patch:
    - bigquery/client.go
    - bigquery/samples/
```

### Create Your First Library

Create the Secret Manager library. Since we are not providing a `--path` flag,
it will be created in the default `generate.dir` (`./secretmanager/`).

```
$ librarian create secretmanager --apis google/cloud/secretmanager/v1
Added library "secretmanager" to librarian.yaml
Generated secretmanager/
Successfully created library "secretmanager"
```

This command adds a full entry to `librarian.yaml`, including an explicit `path`:

```yaml
libraries:
  - name: secretmanager
    path: secretmanager/
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
          # Other generation details are automatically discovered...
```

You can override the default location by providing the `--path` flag.

### Working with Handwritten and Hybrid Libraries

Now, let's add a handwritten Pub/Sub library and a hybrid BigQuery library.

Create the files for the handwritten library:
```
$ mkdir -p pubsub
$ echo "package pubsub\n\nfunc NewClient() {}" > pubsub/client.go
```

Add the handwritten library to your configuration:
```
$ librarian add pubsub --path pubsub/
Added library "pubsub" to librarian.yaml
```

This adds a simple entry to `librarian.yaml`:
```yaml
- name: pubsub
  path: pubsub/
```

Next, add a hybrid BigQuery library. You can start by generating it, and then
add a `patch` section to `librarian.yaml` to protect the files you intend to
customize.

```
$ librarian create bigquery --apis google/cloud/bigquery/storage/v1
```

Now, edit `librarian.yaml` to add the `patch` section:
```yaml
- name: bigquery
  path: bigquery/
  generate:
    apis:
      - path: google/cloud/bigquery/storage/v1
  patch:
    - bigquery/client.go # This file will now be protected
```

Now, when you run `librarian generate bigquery`, `bigquery/client.go` will be
left untouched.

## Python Libraries

The workflow is the same for Python. A typical `librarian.yaml` for Python
will set `generate.dir` to `packages/`.

Initialize a Python repository:
```
$ librarian init python
Created librarian.yaml
```
Your `librarian.yaml` will look like this:
```yaml
...
generate:
  dir: packages/
...
```

Create a fully generated Python library. It will be placed in `packages/google-cloud-secret-manager/` by default.
```
$ librarian create google-cloud-secret-manager --apis google/cloud/secretmanager/v1
```

This creates the following entry in `librarian.yaml`:
```yaml
- name: google-cloud-secret-manager
  path: packages/google-cloud-secret-manager/
  generate:
    apis:
      - path: google/cloud/secretmanager/v1
```

The `release` workflow is the same as for Go, but will publish to PyPI.

## Summary

Librarian provides a consistent workflow across languages, with a clear and
unambiguous configuration for managing generated, handwritten, and hybrid
libraries.

1.  **Initialize** - `librarian init <language>`
2.  **Create** - `librarian create <name> --apis <apis...>` (uses default path)
3.  **Create (override path)** - `librarian create <name> --path <path> --apis <apis...>`
4.  **Add Handwritten** - `librarian add <name> --path <path>`
5.  **Regenerate** - `librarian generate <name>` or `librarian generate --all`
6.  **Release** - `librarian release <name>`

The type of a library is determined by its structure in `librarian.yaml`:
- **Handwritten**: `name` + `path`
- **Generated**: `name` + `path` + `generate`
- **Hybrid**: `name` + `path` + `generate` + `patch`

This design gives you both convenience and full control over the location and
content of your libraries.
