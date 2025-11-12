# Convert Command

The `convert` command converts the old `.librarian` format to the new `librarian.yaml` format.

## Usage

```bash
go run ./cmd/convert <input-dir> <output-file>
```

## Arguments

- `<input-dir>`: Directory containing the `.librarian` subdirectory with `config.yaml` and `state.yaml` files
- `<output-file>`: Path where the new `librarian.yaml` file should be written

## Example

Convert the google-cloud-go repository format:

```bash
go run ./cmd/convert /path/to/google-cloud-go data/go/librarian.yaml
```

This reads:
- `/path/to/google-cloud-go/.librarian/config.yaml`
- `/path/to/google-cloud-go/.librarian/state.yaml`

And outputs:
- `data/go/librarian.yaml`

## Conversion Mapping

The command converts the old format to the new format as follows:

### Container Image
- Old: `image: registry/image:tag`
- New:
  ```yaml
  container:
    image: registry/image
    tag: tag
  ```

### Libraries
- Old: `id` → New: `name`
- Old: `apis[].path` → New: `generate.apis[].path`
- Old: `preserve_regex` → New: `generate.keep`
- Old: `tag_format` → New: `release.tag_format` (global setting, with `{id}` converted to `{name}`)

### Example

Old format (`.librarian/state.yaml`):
```yaml
image: us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go:latest
libraries:
  - id: secretmanager
    version: 1.15.0
    apis:
      - path: google/cloud/secretmanager/v1
        service_config: secretmanager_v1.yaml
    preserve_regex:
      - secretmanager/CHANGES.md
    tag_format: "{id}/v{version}"
```

New format (`librarian.yaml`):
```yaml
version: v1
language: go
container:
  image: us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go
  tag: latest
generate:
  output: '{name}/'
release:
  tag_format: '{name}/v{version}'
libraries:
  - name: secretmanager
    version: 1.15.0
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
      keep:
        - secretmanager/CHANGES.md
```
