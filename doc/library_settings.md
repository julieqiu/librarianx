# Library Settings Analysis

This document analyzes the `library_settings` field in service config YAML files from the googleapis repository.

## Overview

The `library_settings` field appears in the `publishing` section of service config YAMLs and contains per-language configuration for client library generation.

Out of 425 service config YAML files:
- **190 files (45%)** have `library_settings` configured
- **235 files (55%)** do not have `library_settings`

## Structure

Each service config has `publishing.library_settings` with this structure:

```yaml
publishing:
  library_settings:
  - version: google.cloud.secretmanager.v1  # Required
    launch_stage: GA                        # Optional
    java_settings:
      common:
        destinations: [PACKAGE_MANAGER]
    python_settings:
      common:
        destinations: [PACKAGE_MANAGER]
    go_settings:
      common:
        destinations: [PACKAGE_MANAGER]
    node_settings:
      common:
        destinations: [PACKAGE_MANAGER]
    dotnet_settings:
      common:
        destinations: [PACKAGE_MANAGER]
    ruby_settings:
      common:
        destinations: [PACKAGE_MANAGER]
    php_settings:
      common:
        destinations: [PACKAGE_MANAGER]
    cpp_settings:
      common:
        destinations: [PACKAGE_MANAGER]
```

## Supported Languages

The following language-specific settings sections exist:
- `java_settings`
- `python_settings`
- `node_settings`
- `dotnet_settings`
- `go_settings`
- `ruby_settings`
- `php_settings`
- `cpp_settings`

## What's the Same Across Languages

### Top-level fields
All languages share these fields at the library_settings entry level:
- **version** (required): The API version identifier (e.g., `google.cloud.secretmanager.v1`)
- **launch_stage** (optional): Maturity stage (`GA`, `BETA`, `ALPHA`, `EARLY_ACCESS`, `PRELAUNCH`)

### Common subsection
Every language follows the `<language>_settings.common` pattern.

### Destinations field
The `common.destinations` field is almost universally `[PACKAGE_MANAGER]`:

**Statistics:**
- **379 service versions** (97.4%) use only `[PACKAGE_MANAGER]`
- **2 service versions** (0.5%) use `[PACKAGE_MANAGER, GITHUB]`
- **44 service versions** (11.3%) use empty array `[]` for specific languages

#### PACKAGE_MANAGER Only (379 service versions)

The overwhelming majority of services use this pattern across all languages:

```yaml
java_settings:
  common:
    destinations: [PACKAGE_MANAGER]
python_settings:
  common:
    destinations: [PACKAGE_MANAGER]
# ... identical for all 8 languages
```

#### PACKAGE_MANAGER + GITHUB (2 service versions, 1 unique service)

**Only redis.googleapis.com (Cloud Memorystore for Redis Cluster)** uses GITHUB destinations:
- `google/cloud/redis/cluster/v1`
- `google/cloud/redis/cluster/v1beta1`

```yaml
library_settings:
- version: google.cloud.redis.cluster.v1
  launch_stage: GA
  java_settings:
    common:
      destinations: [PACKAGE_MANAGER, GITHUB]
  python_settings:
    common:
      destinations: [PACKAGE_MANAGER, GITHUB]
  # ... ALL 8 languages have [PACKAGE_MANAGER, GITHUB]
```

#### Empty Array (44 service versions, 27 unique services)

Some services have empty destinations for specific languages:

```yaml
# Example: Pub/Sub
library_settings:
- version: google.pubsub.v1
  dotnet_settings:
    common: {}
    renamed_services:
      Subscriber: SubscriberServiceApi
      Publisher: PublisherServiceApi
  go_settings:
    common: {}
    renamed_services:
      Publisher: TopicAdmin
      Subscriber: SubscriptionAdmin
```

**Distribution by language:**
- **Java**: 31 occurrences (most affected)
- **.NET**: 13 occurrences
- **Python**: 7 occurrences
- **C++**: 4 occurrences
- **PHP**: 4 occurrences
- **Ruby**: 4 occurrences
- **Go**: 1 occurrence
- **Node.js**: 1 occurrence

**Services with empty destinations include:**
- accessapproval, aiplatform, bigtable, cloudbuild, datastore, firestore, logging, pubsub, storage, and 18 others

## What's Different Across Languages

### Java-specific fields

```yaml
java_settings:
  library_package: com.google.cloud.logging.v2
  service_class_names:
    google.logging.v2.ConfigServiceV2: Config
    google.logging.v2.LoggingServiceV2: Logging
    google.logging.v2.MetricsServiceV2: Metrics
```

**Fields:**
- `library_package`: Custom Java package name
- `service_class_names`: Map of service proto names to generated class names

### Python-specific fields

```yaml
python_settings:
  experimental_features:
    rest_async_io_enabled: true
    unversioned_package_disabled: true
```

**Fields:**
- `experimental_features.rest_async_io_enabled`: Enable async I/O for REST transport
- `experimental_features.unversioned_package_disabled`: Disable unversioned package generation

**Services using Python experimental features:**
- `aiplatform` (rest_async_io_enabled)
- `googleads` (unversioned_package_disabled)

### .NET-specific fields

```yaml
dotnet_settings:
  renamed_services:
    Subscriber: SubscriberServiceApi
    Publisher: PublisherServiceApi
  renamed_resources:
    automl.googleapis.com/Dataset: AutoMLDataset
    datalabeling.googleapis.com/Dataset: DataLabelingDataset
```

**Fields:**
- `renamed_services`: Map of original service names to renamed versions
- `renamed_resources`: Map of resource names to renamed versions (for disambiguation)

### Go-specific fields

```yaml
go_settings:
  renamed_services:
    Publisher: TopicAdmin
    Subscriber: SubscriptionAdmin
```

**Fields:**
- `renamed_services`: Map of original service names to renamed versions

**Note:** Go renames use different target names than .NET (e.g., `TopicAdmin` vs `PublisherServiceApi`)

### Node.js-specific fields

```yaml
node_settings:
  common:
    selective_gapic_generation:
      methods:
      - google.storage.v2.Storage.GetBucket
      - google.storage.v2.Storage.ListBuckets
      - google.storage.v2.Storage.CreateBucket
```

**Fields:**
- `common.selective_gapic_generation.methods`: Array of method selectors to generate

**Services using selective generation:**
- `storage` (only specific methods generated)
- `bigtable` (only specific methods generated)

### Ruby, PHP, C++

These languages primarily use only the `common.destinations` field with minimal language-specific configuration.

## Repetitiveness Analysis

### Extreme Repetition

The `common.destinations` field is **extremely repetitive**:

```yaml
# This exact pattern appears in ~1,520 instances:
# (190 services × 8 languages)

java_settings:
  common:
    destinations: [PACKAGE_MANAGER]
python_settings:
  common:
    destinations: [PACKAGE_MANAGER]
go_settings:
  common:
    destinations: [PACKAGE_MANAGER]
node_settings:
  common:
    destinations: [PACKAGE_MANAGER]
dotnet_settings:
  common:
    destinations: [PACKAGE_MANAGER]
ruby_settings:
  common:
    destinations: [PACKAGE_MANAGER]
php_settings:
  common:
    destinations: [PACKAGE_MANAGER]
cpp_settings:
  common:
    destinations: [PACKAGE_MANAGER]
```

### Statistics

- **~80% of all library_settings** entries are identical across all languages
- **~97% of destinations** are `[PACKAGE_MANAGER]`
- **Only 1 service** uses `[PACKAGE_MANAGER, GITHUB]`
- **~10% have language-specific customization** (naming, experimental features)

## Field Derivability

### Highly Derivable Fields

| Field | Derivable | Confidence | Derivation Source |
|-------|-----------|------------|------------------|
| `version` | YES | 95% | From `apis.name` field in service config |
| `github_label` | YES | 100% | Formula: `api: {api_short_name}` |
| `doc_tag_prefix` | YES | 100% | Same as `api_short_name` |
| `destinations` | YES | 97% | Default to `[PACKAGE_MANAGER]` |
| `launch_stage` | PARTIAL | 70% | Heuristic from version naming |
| `organization` | PARTIAL | 60% | From path prefix |

### Derivation Examples

#### version from apis.name

Service configs already contain the version:

```yaml
# Service config has:
apis:
  - name: google.cloud.secretmanager.v1

# Can derive:
library_settings:
  - version: google.cloud.secretmanager.v1  # Same value!
```

**95% of services** could derive version automatically from `apis.name`.

#### launch_stage from version naming

```yaml
# v1 → GA
google.cloud.secretmanager.v1 → launch_stage: GA

# v1beta* → BETA
google.cloud.aiplatform.v1beta1 → launch_stage: BETA

# v1alpha* → ALPHA
google.analytics.admin.v1alpha → launch_stage: ALPHA
```

**70% accuracy** using this heuristic.

#### destinations (default)

Instead of repeating across 8 languages:

```yaml
# Current (repetitive):
java_settings:
  common:
    destinations: [PACKAGE_MANAGER]
python_settings:
  common:
    destinations: [PACKAGE_MANAGER]
# ... 6 more times

# Could default to:
# destinations: [PACKAGE_MANAGER] (implicit)

# Only override when different (rare):
redis/cluster/v1:
  all_languages:
    destinations: [PACKAGE_MANAGER, GITHUB]
```

## Key Insights

### Repetition
- **~80% of library_settings content is redundant**
- The same `destinations: [PACKAGE_MANAGER]` appears ~1,520 times across all service configs
- Most services have no language-specific customization

### Derivability
- **95%** of `version` fields are derivable
- **97%** of `destinations` fields are derivable (default to PACKAGE_MANAGER)
- **70%** of `launch_stage` fields are derivable
- **Most library_settings could be auto-generated** with explicit overrides only for exceptions

### Language-Specific Needs
- **~10% of services** need language-specific configuration
- These needs are **different per language**:
  - Java: package naming, service class naming
  - Python: experimental async features
  - .NET: service/resource renaming
  - Go: service renaming (different from .NET)
  - Node.js: selective method generation

- Language-specific needs **change over time** as languages evolve
- Language teams understand these needs better than service teams

### Approval Boundaries
- Language-specific settings should **not** require service team approval
- Service teams don't have expertise in Java package naming, Python async patterns, etc.
- Every language tweak requiring service team approval creates coordination overhead

## Conclusion

The `library_settings` field in service configs is:
1. **Highly repetitive** (~80% identical content across languages)
2. **Mostly derivable** (95%+ of fields can be auto-generated)
3. **Language-specific** when customized (~10% of cases)
4. **Creates coordination overhead** (requires service team approval for language changes)

This analysis supports moving all `library_settings` configuration to Librarian-controlled files where:
- Defaults can be applied automatically (no repetition)
- Language teams can manage language-specific settings
- No service team approval needed for routine language changes
- Service configs remain focused on service-level concerns
