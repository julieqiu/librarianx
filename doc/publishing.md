# Publishing Section Analysis

This document analyzes the publishing sections in googleapis service config YAML files.

## Service Config Fields Used by Sidekick

This section lists the fields that Sidekick actually uses from the service config YAML files.

### Top-level fields

**`name`** - The service name (e.g., `secretmanager.googleapis.com`).

Used in:
- `internal/sidekick/parser/protobuf.go:328`
- `internal/sidekick/parser/openapi.go:85`
- `internal/sidekick/parser/discovery/discovery.go:62`

**`title`** - The service title.

Used in:
- `internal/sidekick/parser/protobuf.go:197`
- `internal/sidekick/parser/openapi.go:86`
- `internal/sidekick/parser/discovery/discovery.go:63`

**`documentation`** - Service-level documentation.

**`documentation.summary`** - Service description.

Used in:
- `internal/sidekick/parser/protobuf.go:198-199`
- `internal/sidekick/parser/openapi.go:87-88`
- `internal/sidekick/parser/discovery/discovery.go:64-65`

**`apis`** - List of API definitions. Used to determine mixins and extract package names.

Used in:
- `internal/sidekick/parser/svcconfig/svcconfig.go:36`
- `internal/sidekick/parser/mixin.go:55`

### Publishing configuration

**`publishing.method_settings`** - Method-specific settings.

**`selector`** - Method selector (e.g., `.google.cloud.secretmanager.v1.SecretManagerService.CreateSecret`).

**`auto_populated_fields`** - List of field names that are auto-populated per [AIP-4235](https://google.aip.dev/client-libraries/4235).

Used in:
- `internal/sidekick/parser/auto_populated.go:33-44`

### HTTP routing

**`http.rules`** - HTTP routing rules.

**`selector`** - Method selector.

Used in:
- `internal/sidekick/parser/mixin.go:110`
- `internal/sidekick/parser/mixin.go:131`

### Documentation rules

**`documentation.rules`** - Method-level documentation overrides.

**`selector`** - Method selector.

**`description`** - Documentation text.

Used in:
- `internal/sidekick/parser/mixin.go:147`

### Implementation notes

The service config is loaded in `internal/sidekick/parser/service_config.go:29-34` and uses `DiscardUnknown: true` when unmarshalling (line 52), so any other fields in the service config are ignored.

The parser focuses on:
- **Service metadata**: name, title, description
- **API definitions**: for mixin detection and package name extraction
- **Publishing settings**: for auto-populated fields (AIP-4235)
- **HTTP routing**: for REST endpoint definitions (especially for mixins)
- **Documentation overrides**: for method-level documentation

## Publishing Section Statistics

## Coverage

Out of 425 service config YAML files:
- **214 (50.4%)** have publishing sections filled out
- **211 (49.6%)** do not have publishing sections

## Fields

The publishing section contains these fields (by frequency):

| Field | Count | Description |
|-------|-------|-------------|
| library_settings | 190 | Per-language library configuration |
| documentation_uri | 174 | External documentation link |
| new_issue_uri | 172 | Issue tracker link |
| organization | 165 | Organizational group (CLOUD, GEO, SHOPPING, etc.) |
| github_label | 165 | GitHub label for categorization |
| api_short_name | 159 | Short identifier for the API |
| doc_tag_prefix | 158 | Documentation tag prefix |
| proto_reference_documentation_uri | 61 | Protocol buffer reference documentation |
| method_settings | 23 | Per-method configuration (e.g., long_running, auto_populated_fields) |
| rest_reference_documentation_uri | 1 | REST API reference (rare) |
| codeowner_github_teams | 1 | Code owner teams (rare) |

## Common Patterns

### Organizations

| Organization | Count | Percentage |
|-------------|-------|-----------|
| CLOUD | 124 | 73% |
| SHOPPING | 28 | 17% |
| GEO | 8 | 5% |
| GENERATIVE_AI | 3 | 2% |
| ADS | 3 | 2% |
| CLIENT_LIBRARY_ORGANIZATION_UNSPECIFIED | 1 | <1% |

### Language Support

Language support in library_settings:

| Language | Count | Percentage |
|----------|-------|-----------|
| Java | 175 | 91% |
| Python | 158 | 82% |
| Go | 146 | 76% |
| Node.js | 141 | 73% |
| .NET | 137 | 71% |
| PHP | 126 | 66% |
| Ruby | 124 | 65% |
| C++ | 101 | 53% |

### Launch Stages

| Stage | Count | Percentage |
|-------|-------|-----------|
| GA (General Availability) | 90 | 63% |
| BETA | 45 | 31% |
| ALPHA | 13 | 9% |
| EARLY_ACCESS | 5 | 3% |
| PRELAUNCH | 2 | 1% |

### Destinations

| Destination | Count | Percentage |
|------------|-------|-----------|
| PACKAGE_MANAGER | 1,043 | 97% |
| GITHUB | 16 | 3% |

### Method Settings

**Distribution:**
- Files with method_settings: 23
- Files with BOTH method_settings AND library_settings: 8
- Files with ONLY method_settings (no library_settings): 15

**auto_populated_fields:**
- request_id: 16 occurrences (only auto-populated field found)

**long_running settings** (100 total occurrences):
- **initial_poll_delay**: Most common: 60s (41), then 30s (16), 5s (14), 10s (9), 1s (7)
- **poll_delay_multiplier**: 1.5 (67 occurrences), 2.0 (33 occurrences)
- **max_poll_delay**: 120s (22), 45s (20), 360s (18), 60s (17), 180s (7)
- **total_poll_timeout**: 1200s (18), 7200s (15), 4800s (15), 86400s (8), 300s (8)

## Exceptions

### smartdevicemanagement_v1.yaml
All publishing fields are EMPTY strings:
- organization = CLIENT_LIBRARY_ORGANIZATION_UNSPECIFIED
- Has codeowner_github_teams field (no values)
- Has empty library_settings list
- Appears to be a template or placeholder configuration

### routeoptimization_v1.yaml
- Only service with rest_reference_documentation_uri field
- Uses REST-specific reference instead of proto reference

### redis/cluster/v1
- Only service with GITHUB destination in addition to PACKAGE_MANAGER
- All 8 languages include both destinations

### Method-settings-only APIs
15 files do not include library_settings and focus entirely on per-method configuration:
- Examples: Cloud Optimization, Video Intelligence (all versions), Vision (beta), Edge Container
- These are specialized APIs with specific polling requirements

## Examples

### Typical Publishing Section

From Maps API - Solar (`google/maps/solar/v1/solar_service.yaml`):

```yaml
publishing:
  new_issue_uri: https://issuetracker.google.com/issues/new?component=1356349
  documentation_uri: https://developers.google.com/maps/documentation/solar/overview
  api_short_name: solar
  github_label: 'api: solar'
  doc_tag_prefix: solar
  organization: GEO
  library_settings:
  - version: google.maps.solar.v1
    launch_stage: GA
    java_settings:
      common:
        destinations:
        - PACKAGE_MANAGER
    python_settings:
      common:
        destinations:
        - PACKAGE_MANAGER
    node_settings:
      common:
        destinations:
        - PACKAGE_MANAGER
    go_settings:
      common:
        destinations:
        - PACKAGE_MANAGER
  proto_reference_documentation_uri: https://developers.google.com/maps/documentation/solar/reference/rest
```

### Long-Running Operations

From Redis v1 (`google/cloud/redis/v1/redis.yaml`):

```yaml
publishing:
  method_settings:
  - selector: google.cloud.redis.v1.CloudRedis.CreateInstance
    long_running:
      initial_poll_delay: 60s
      poll_delay_multiplier: 1.5
      max_poll_delay: 360s
      total_poll_timeout: 7200s
  - selector: google.cloud.redis.v1.CloudRedis.UpdateInstance
    long_running:
      initial_poll_delay: 60s
      poll_delay_multiplier: 1.5
      max_poll_delay: 360s
      total_poll_timeout: 7200s
  new_issue_uri: https://issuetracker.google.com/issues/new?component=1288776&template=1161103
  documentation_uri: https://cloud.google.com/memorystore/docs/redis
  api_short_name: redis
  github_label: 'api: redis'
  doc_tag_prefix: redis
  organization: CLOUD
  library_settings:
  - version: google.cloud.redis.v1
    launch_stage: GA
    java_settings:
      common:
        destinations:
        - PACKAGE_MANAGER
```

### Auto-Populated Fields

From Storage Control v2 (`google/storage/control/v2/storage_control.yaml`):

```yaml
publishing:
  method_settings:
  - selector: google.storage.control.v2.StorageControl.CreateFolder
    auto_populated_fields:
    - request_id
  - selector: google.storage.control.v2.StorageControl.DeleteFolder
    auto_populated_fields:
    - request_id
  new_issue_uri: https://issuetracker.google.com/issues/new?component=187243&template=1162869
  documentation_uri: https://cloud.google.com/storage/docs/overview
```

### GitHub Destinations

From Redis Cluster (`google/cloud/redis/cluster/v1/cloudrediscluster.yaml`):

```yaml
library_settings:
- version: google.cloud.redis.cluster.v1
  launch_stage: GA
  java_settings:
    common:
      destinations:
      - PACKAGE_MANAGER
      - GITHUB
  python_settings:
    common:
      destinations:
      - PACKAGE_MANAGER
      - GITHUB
```

## Key Insights

1. **Maturity**: 50% of services have complete publishing configurations, indicating structured documentation and client library support for half the APIs.

2. **Language Support**: Strong polyglot support with Java leading at 91%, but coverage varies significantly across languages (C++ at lowest with 53%).

3. **Configuration Strategies**: APIs use two distinct approaches:
   - **Library + Metadata**: Most (192) combine library_settings with documentation/organization metadata
   - **Method-Specific**: Some (15) focus purely on method-level configuration like polling strategies

4. **Polling Patterns**: Consistent patterns emerge in long-running operations, with 60s initial delay and 1.5x multiplier as the most common (70% of cases).

5. **Incomplete Metadata**: One service (smartdevicemanagement) has placeholder publishing config, suggesting manual intervention needed.

## Field Derivability Analysis

### documentation_uri Patterns

Out of 175 files with documentation_uri:

**Path Endings for cloud.google.com URLs (121 total):**

| Path Ending | Count | Percentage |
|-------------|-------|------------|
| `/docs` (no suffix) | 30 | 25% |
| `/docs/overview` | 19 | 16% |
| `/docs/{page}` (single page) | 23 | 19% |
| `/docs/{path}/{page}` (deep path) | 23 | 19% |
| Other `/docs` variations | 20 | 17% |
| No `/docs` in path | 6 | 5% |

**Base Domain Distribution:**

| Domain | Count | Percentage |
|--------|-------|------------|
| `cloud.google.com` | 121 | 69% |
| `developers.google.com` | 50 | 29% |
| `ai.google.dev` | 3 | 2% |
| Other | 1 | <1% |

**Predictability:** Only 8% of documentation_uri values follow a predictable pattern that could be derived from api_short_name. The remaining 92% require explicit specification due to:
- Inconsistent naming transformations (e.g., `visionai` → `vision-ai`, `datacatalog` → `data-catalog`)
- Varied URL structures (shallow vs deep paths)
- Nested products (e.g., `parametermanager` → `secret-manager/parameter-manager/docs/overview`)
- Domain differences by organization

**Examples of unpredictable mappings:**
- `workspaceevents` → `developers.google.com/workspace/events` (different naming)
- `cloudquotas` → `cloud.google.com/docs/quotas/api-overview` (no product segment)
- `notebooks` → `cloud.google.com/vertex-ai/docs/workbench/instances/introduction` (nested, deep path)

**Conclusion:** documentation_uri must remain explicit in configs.

### new_issue_uri Factorization

Out of 368 files with new_issue_uri:

**Pattern Consistency:**
- **343 (93.2%)** use `https://issuetracker.google.com/issues/new?` with predictable query parameters
- **25 (6.8%)** use alternative formats (GitHub URLs, cloud.google.com docs, or empty strings)

**Query Parameters (for issuetracker.google.com):**

| Parameter | Count | Percentage |
|-----------|-------|------------|
| `component=` | 343/343 | 100% |
| `template=` | 228/343 | 66.5% |
| `pli=` | 2/343 | 0.6% |

**Parameter Combinations:**
- `component=` only: 115 URIs (33.5%)
- `component=&template=`: 226 URIs (65.9%)
- `component=&template=&pli=`: 2 URIs (0.6%)

**Exceptions:**
- GitHub URLs: 17 entries (e.g., `https://github.com/google/generative-ai-python/issues/new`)
- Google Cloud docs: 4 entries (e.g., `https://cloud.google.com/certificate-manager/docs/getting-support`)
- Empty values: 4 entries

**Conclusion:** The base URL `https://issuetracker.google.com/issues/new?` can be factored out for 93.2% of cases.

**Recommended approach:**
Store only the variable parts:
```yaml
publishing:
  issue_component: "784854"     # Required for issuetracker URLs
  issue_template: "1380926"     # Optional
  custom_issue_uri: ""           # For GitHub/other URLs (overrides component-based)
```

This would:
- Reduce redundancy across 343 files
- Make component/template IDs easily searchable
- Simplify validation (component IDs are always numeric)
- Allow easier migration if the base URL changes

### Auto-Derivable Fields Summary

| Field | Derivable | Confidence | Notes |
|-------|-----------|------------|-------|
| api_short_name | YES | 95% | From service name or file path |
| github_label | YES | 100% | Formula: 'api: {api_short_name}' |
| doc_tag_prefix | YES | 100% | Same as api_short_name |
| organization | PARTIAL | 60% | From path prefix for CLOUD/GEO |
| documentation_uri | NO | 0% | Custom per service |
| new_issue_uri (base) | YES | 93% | Can factor out base URL |
| new_issue_uri (params) | NO | 0% | Service-specific component IDs |
| library_settings.version | YES | 95% | From apis.name field |
| library_settings.launch_stage | PARTIAL | 70% | Heuristic from version naming |
| library_settings.destinations | YES | 80% | Default to PACKAGE_MANAGER |
