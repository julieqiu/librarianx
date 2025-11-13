# Service Team Onboarding

## Objective

Simplify the onboarding process for new client libraries and remove manual work.

## Background

Today, every language team (google-cloud-go, google-cloud-python, etc.) has a separate onboarding process. When a service team wants to publish a new client library:

1. The service team files a bug with each language team
2. Each language team does manual work to onboard the library
3. Each language team runs generation separately
4. Each language team handles releases independently

This is fragmented and creates unnecessary manual work. The Librarian team cannot easily discover which services want client libraries. They must manually track service team requests and coordinate with multiple language teams.

This document proposes adding a `publish_client_libraries` flag to service config YAMLs. This allows the Librarian team to own the entire onboarding, generation, and release process for all languages. Language teams no longer need to think about onboarding, generation, or releases.

## Overview

Service teams set `publish_client_libraries: true` in their service config YAML under the `publishing` section. When this flag is set and the configuration is mirrored to googleapis, running `librarian generate --all --update` discovers and generates the new library automatically.

This enables:
- **Centralized ownership**: Librarian team owns onboarding, generation, and releases for all languages
- **Automatic discovery**: Librarian finds new libraries without manual coordination
- **Full control**: Language-specific settings live in Librarian-controlled files, not service configs
- **Reduced coordination**: No service team approval needed for language-specific changes
- **Explicit intent**: The flag makes it clear which services want client libraries

## Detailed Design

### Service Config Changes

Add a new boolean field `publish_client_libraries` to the `publishing` section:

```yaml
publishing:
  publish_client_libraries: true
  method_settings: [...]
  # ... other publishing fields
```

When `publish_client_libraries: true`, the service opts into client library generation for wildcard mode.

When `publish_client_libraries: false` or absent, the service opts out (unless in the legacy list).

### Wildcard Mode Behavior

When `librarian.yaml` contains `libraries: ['*']`, Librarian generates libraries where:
- `publish_client_libraries: true` in the service config, OR
- The service appears in the legacy library list

This allows the Librarian team to automatically discover and generate all services that want client libraries, while maintaining backward compatibility for existing services.

### Legacy Library List

Create `data/{language}/legacy-libraries.yaml` in the Librarian repository:

```yaml
# Libraries that existed before the publish_client_libraries flag.
# These are treated as if they have publish_client_libraries: true.
#
# Generated on 2025-01-XX from googleapis snapshot.
legacy_libraries:
  - google.cloud.secretmanager.v1
  - google.cloud.vision.v1
  - google.cloud.translate.v3
  # ... ~200 existing services
```

Why a legacy list:
- Existing services should not break when we deploy this change
- Service teams should not need to update ~200 service configs retroactively
- The list is frozen at a point in time and does not grow
- New services must use the flag

The legacy list lives in the Librarian repository, not in googleapis. This keeps the approval boundary clear: only Librarian maintainers need to approve changes to the legacy list.

### Explicit Library Overrides

When someone explicitly lists a library in `librarian.yaml`, it always generates regardless of the flag:

```yaml
libraries:
  - '*'

  # Generate even if publish_client_libraries is false
  - packages/google-cloud-experimental:
      override_publish_flag: true
      reason: "Testing new API before public launch"
```

The `override_publish_flag` field is optional but recommended. It documents that we are intentionally overriding the service team's decision.

Why allow overrides:
- Librarian maintainers may need to test APIs before public launch
- Some experimental workflows may require generating unlisted libraries
- The override is explicit and documented in version control

When an override is active, `librarian generate` logs a warning:

```
WARNING: Overriding publish_client_libraries for packages/google-cloud-experimental
Reason: Testing new API before public launch
```

### Handwritten Libraries

Handwritten libraries (like `pubsub/`, `storage/`, `auth/`) do not have service configs. The flag does not apply to them.

They continue to work as they do today: list them explicitly in `librarian.yaml`.

### Language Settings

The service config YAML currently includes `publishing.library_settings` with per-language configuration:

```yaml
publishing:
  library_settings:
    - version: google.cloud.secretmanager.v1
      launch_stage: GA
      java_settings:
        common:
          destinations:
            - PACKAGE_MANAGER
      python_settings:
        common:
          destinations:
            - PACKAGE_MANAGER
```

Librarian does not use these fields. All language-specific settings are managed in files controlled entirely by the Librarian team (like `librarian.yaml` or separate configuration files in the Librarian repository).

Why language settings must not be in service configs:
- Every language-specific change would require service team approval
- Creates coordination overhead for routine changes
- Language teams know their ecosystems better than service teams
- Librarian team needs full control over generation and release processes

Librarian ignores `library_settings` during parsing (it already uses `DiscardUnknown: true`).

Over time, `library_settings` can be deprecated from service config YAMLs since no tool will use them.

### The --update Flag

The `--update` flag on `librarian generate` fetches the latest googleapis tarball:

```bash
$ librarian generate --all --update
Fetching latest googleapis...
Downloaded googleapis-abc123.tar.gz (sha256: 81e6057...)
Updated librarian.yaml with new source hash
Discovering APIs from googleapis...
Found 238 APIs (1 new: google.cloud.newservice.v1)
  ✓ packages/google-cloud-newservice/  [NEW]
  ✓ packages/google-cloud-secretmanager/
  ... 236 more ...
Done.
```

This enables the workflow:
1. Service team sets `publish_client_libraries: true`
2. Change merges and mirrors to googleapis
3. Librarian maintainer runs `librarian generate --all --update`
4. New library appears automatically

### Discovery Algorithm

When running `librarian generate --all`:

1. Load service configs from the googleapis tarball
2. For each service config:
   - Check if `publishing.publish_client_libraries` is `true`
   - OR check if the service is in the legacy list
   - If yes, mark it for generation
3. Discover API definitions (protos/discovery docs) for marked services
4. Generate libraries for discovered APIs
5. Skip APIs where the flag is `false` or absent (and not in legacy list)

## Alternatives Considered

### Alternative 1: Explicit list in librarian.yaml

Instead of using the service config, add a list of enabled services to `librarian.yaml`:

```yaml
libraries:
  - packages/google-cloud-secretmanager
  - packages/google-cloud-vision
  # ... 200 explicit entries
```

Why we rejected this:
- Librarian team must manually update configuration for every new service
- Does not reduce manual work
- Duplicates information that should live with the service definition

### Alternative 2: Opt-out instead of opt-in

Default to generating all libraries unless `publish_client_libraries: false`:

```yaml
publishing:
  publish_client_libraries: false  # Opt out
```

Why we rejected this:
- Less safe: new services appear automatically without explicit intent
- Service teams may not realize their service will publish client libraries
- Opt-in is more explicit and intentional

### Alternative 3: Separate discovery manifest file

Create a new `client-libraries.yaml` file in googleapis that lists all services:

```yaml
client_libraries:
  - google.cloud.secretmanager.v1
  - google.cloud.vision.v1
```

Why we rejected this:
- Separates the flag from the service definition
- Harder to maintain consistency
- Service teams must edit two files instead of one
- Does not leverage existing service config structure

### Alternative 4: Language-specific flags

Allow service teams to control which languages get generated:

```yaml
publishing:
  publish_client_libraries:
    python: true
    go: true
    rust: false
```

Why we rejected this:
- Puts language decisions in service config, requiring service team approval for language changes
- Violates the principle that language settings belong in Librarian-controlled files
- Adds coordination overhead when adding/removing language support
- Service teams typically want all languages or none
- Librarian team needs full control over which languages to support

### Alternative 5: No legacy list, require retroactive updates

Require all ~200 existing services to add the flag to their configs.

Why we rejected this:
- High coordination cost: need to file ~200 PRs
- Risk of breaking existing libraries if PRs are not merged
- No benefit: existing services already work
- Legacy list is simpler and safer
