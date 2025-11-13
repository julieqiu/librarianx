# Library Settings Migration

## Objective

Move language-specific configuration from service config `library_settings` to `librarian.yaml`, allowing the Librarian team to control all language settings without requiring service team approval.

## Background

Currently, `library_settings` in service config YAMLs contains:
- **~80% redundant data** (`destinations: [PACKAGE_MANAGER]` repeated ~1,520 times)
- **~95% derivable fields** (version, launch_stage, destinations)
- **~10% language-specific customization** that changes over time

Every change to language settings requires service team approval, creating coordination overhead for routine language improvements.

## Overview

Migrate the ~10% of non-derivable, language-specific settings from `library_settings` to `librarian.yaml`. The remaining 90% will be derived automatically or use sensible defaults.

Service config `library_settings` can then be ignored or eventually deprecated.

## What Gets Migrated

### Language-Specific Settings to Preserve

These settings must be migrated to `librarian.yaml`:

#### Java (31 services with library_package, 14 with service_class_names)

```yaml
# Example: Logging
- logging:
    api: google/logging/v2
    java:
      package: com.google.cloud.logging.v2
      service_class_names:
        google.logging.v2.ConfigServiceV2: Config
        google.logging.v2.LoggingServiceV2: Logging
        google.logging.v2.MetricsServiceV2: Metrics
```

#### Python (2 services with async, 4 with unversioned package)

```yaml
# Example: AI Platform
- aiplatform:
    apis: [google/cloud/aiplatform/v1, google/cloud/aiplatform/v1beta1]
    python:
      rest_async_io_enabled: true

# Example: Google Ads
- google-ads-googleads:
    apis: [google/ads/googleads/v19, google/ads/googleads/v20, ...]
    python:
      unversioned_package_disabled: true
```

#### .NET (8 services with renamed_services, 3 with renamed_resources)

```yaml
# Example: Pub/Sub
- pubsub:
    api: google/pubsub/v1
    dotnet:
      renamed_services:
        Subscriber: SubscriberServiceApi
        Publisher: PublisherServiceApi

# Example: AI Platform
- aiplatform:
    apis: [google/cloud/aiplatform/v1, google/cloud/aiplatform/v1beta1]
    dotnet:
      renamed_resources:
        datalabeling.googleapis.com/Dataset: DataLabelingDataset
        automl.googleapis.com/Dataset: AutoMLDataset
```

#### Go (2 services with renamed_services)

```yaml
# Example: Pub/Sub
- pubsub:
    api: google/pubsub/v1
    go:
      renamed_services:
        Publisher: TopicAdmin
        Subscriber: SubscriptionAdmin
```

#### Node.js (2 services with selective generation)

```yaml
# Example: Storage
- storage:
    api: google/storage/v2
    node:
      selective_methods:
        - google.storage.v2.Storage.GetBucket
        - google.storage.v2.Storage.CreateBucket
        - google.storage.v2.Storage.DeleteBucket
```

### Exception: Destinations

Only 1 service needs non-default destinations:

```yaml
# Redis Cluster (only service with GITHUB destinations)
- redis:
    apis: [google/cloud/redis/cluster/v1, google/cloud/redis/cluster/v1beta1]
    destinations: [PACKAGE_MANAGER, GITHUB]
```

**Question**: What do empty destinations `[]` mean? (44 service versions have this)
- If it means "don't publish", we need a `publish: false` field
- If it means "handwritten", they shouldn't be in wildcard discovery anyway
- Need to investigate these 27 services

### Exception: Launch Stage

Only services where heuristic fails need explicit launch_stage:

```yaml
# Example: Service at GA but version looks like beta
- some-service:
    api: google/some/service/v1beta1
    launch_stage: GA  # Override heuristic (v1beta* → BETA)
```

**Most services don't need this** - the heuristic works 70% of the time, and the remaining 30% can use defaults.

## What Gets Ignored or Derived

### Ignored Fields

These fields in `library_settings` will be completely ignored:

1. **version**: Always derivable from `apis.name` field
2. **destinations**: Default to `[PACKAGE_MANAGER]` (except 1 service)
3. **common.destinations**: Same as above (ignored entirely)

### Derived Fields

These fields will be auto-derived:

1. **version**: From API path or `apis.name`
   ```
   google/cloud/secretmanager/v1 → google.cloud.secretmanager.v1
   ```

2. **launch_stage**: From version naming (70% accuracy is acceptable)
   ```
   v1 → GA
   v1beta1 → BETA
   v1alpha → ALPHA
   ```

3. **destinations**: Default `[PACKAGE_MANAGER]` for all languages

## Proposed librarian.yaml Structure

### Extending Existing LibraryConfig

```go
type LibraryConfig struct {
    // ... existing fields ...

    // Language-specific settings
    Java   *JavaLibrary   `yaml:"java,omitempty"`
    Python *PythonLibrary `yaml:"python,omitempty"`
    Node   *NodeLibrary   `yaml:"node,omitempty"`
    Dotnet *DotnetLibrary `yaml:"dotnet,omitempty"`
    Go     *GoLibrary     `yaml:"go,omitempty"`

    // Override derivation
    LaunchStage  string   `yaml:"launch_stage,omitempty"`
    Destinations []string `yaml:"destinations,omitempty"`
}

type JavaLibrary struct {
    Package           string            `yaml:"package,omitempty"`
    ServiceClassNames map[string]string `yaml:"service_class_names,omitempty"`
}

type PythonLibrary struct {
    RestAsyncIOEnabled        bool `yaml:"rest_async_io_enabled,omitempty"`
    UnversionedPackageDisabled bool `yaml:"unversioned_package_disabled,omitempty"`
}

type NodeLibrary struct {
    SelectiveMethods []string `yaml:"selective_methods,omitempty"`
}

type DotnetLibrary struct {
    RenamedServices  map[string]string `yaml:"renamed_services,omitempty"`
    RenamedResources map[string]string `yaml:"renamed_resources,omitempty"`
}

type GoLibrary struct {
    RenamedServices map[string]string `yaml:"renamed_services,omitempty"`
}
```

### Example: Complete Migration

**Before (service config library_settings):**

```yaml
publishing:
  library_settings:
  - version: google.logging.v2
    launch_stage: GA
    java_settings:
      library_package: com.google.cloud.logging.v2
      service_class_names:
        google.logging.v2.ConfigServiceV2: Config
        google.logging.v2.LoggingServiceV2: Logging
      common:
        destinations: [PACKAGE_MANAGER]
    python_settings:
      common:
        destinations: [PACKAGE_MANAGER]
    go_settings:
      common:
        destinations: [PACKAGE_MANAGER]
    # ... 5 more languages with identical destinations
```

**After (librarian.yaml):**

```yaml
libraries:
  - '*'

  - logging:
      api: google/logging/v2
      java:
        package: com.google.cloud.logging.v2
        service_class_names:
          google.logging.v2.ConfigServiceV2: Config
          google.logging.v2.LoggingServiceV2: Logging
```

**What's removed:**
- `version` (derived from api path)
- `launch_stage: GA` (derived from v2 → GA heuristic)
- All 8 `destinations: [PACKAGE_MANAGER]` (default)

**What's kept:**
- Java-specific package and class names

## Migration Strategy

### Phase 1: Identify All Language-Specific Settings

Extract from all 425 service configs:
- Java: library_package, service_class_names
- Python: experimental_features
- .NET: renamed_services, renamed_resources
- Go: renamed_services
- Node: selective_gapic_generation

Create migration data file listing all exceptions.

### Phase 2: Add Fields to librarian.yaml

Update structs in `internal/config/config.go`:
- Add JavaLibrary, NodeLibrary
- Extend PythonLibrary, GoLibrary, DotnetLibrary
- Add LaunchStage, Destinations to LibraryConfig

### Phase 3: Migrate Existing Configurations

For each language's `librarian.yaml`:
1. Parse current `library_settings` from googleapis
2. Extract language-specific settings
3. Add to appropriate library entries in `librarian.yaml`
4. Verify no data loss

### Phase 4: Update Generation Code

Modify generation code to:
1. Read language settings from `librarian.yaml` instead of service config
2. Derive version, launch_stage, destinations automatically
3. Use defaults when not specified

### Phase 5: Ignore library_settings in Service Configs

Update `internal/sidekick/parser/service_config.go`:
- Keep `DiscardUnknown: true` (already ignores library_settings)
- Document that library_settings is not used
- Remove any code that reads library_settings

## Examples by Language

### Go Example

**Services with Go-specific settings:**
- pubsub (renamed_services)
- logging (empty destinations - investigate)

```yaml
# data/go/librarian.yaml
libraries:
  - '*'

  # Pub/Sub: Renamed services
  - pubsub:
      go:
        renamed_services:
          Publisher: TopicAdmin
          Subscriber: SubscriptionAdmin
```

### Python Example

**Services with Python-specific settings:**
- aiplatform (rest_async_io_enabled)
- googleads v19-v22 (unversioned_package_disabled)

```yaml
# data/python/librarian.yaml
libraries:
  - '*'

  # AI Platform: Async REST I/O
  - google-cloud-aiplatform:
      apis: [google/cloud/aiplatform/v1, google/cloud/aiplatform/v1beta1]
      python:
        rest_async_io_enabled: true

  # Google Ads: Disable unversioned package
  - google-ads-googleads:
      apis:
        - google/ads/googleads/v19
        - google/ads/googleads/v20
        - google/ads/googleads/v21
        - google/ads/googleads/v22
      python:
        unversioned_package_disabled: true
```

### Java Example

**Many services with Java-specific settings (31 with package, 14 with service names):**

```yaml
# data/java/librarian.yaml (excerpt)
libraries:
  - '*'

  - logging:
      api: google/logging/v2
      java:
        package: com.google.cloud.logging.v2
        service_class_names:
          google.logging.v2.ConfigServiceV2: Config
          google.logging.v2.LoggingServiceV2: Logging
          google.logging.v2.MetricsServiceV2: Metrics

  - bigquerystorage:
      api: google/cloud/bigquery/storage/v1
      java:
        package: com.google.cloud.bigquery.storage.v1
        service_class_names:
          google.cloud.bigquery.storage.v1.BigQueryRead: BaseBigQueryRead

  # ... ~29 more services with Java-specific config
```

## Open Questions

### 1. Empty Destinations []

27 services have empty destinations for specific languages:
- accessapproval, aiplatform, bigtable, cloudbuild, datastore, firestore, etc.

What does this mean?
- **Option A**: Don't publish to package manager (need `publish: false` field)
- **Option B**: Handwritten library (shouldn't be in wildcard anyway)
- **Option C**: Legacy config that should be removed

**Action**: Investigate these 27 services to understand intent.

### 2. Launch Stage Exceptions

70% of launch_stage can be derived from version naming. For the remaining 30%:
- **Option A**: Store explicit launch_stage in librarian.yaml
- **Option B**: Accept heuristic errors and fix manually when reported
- **Option C**: Don't migrate launch_stage at all (what uses it?)

**Action**: Determine if launch_stage is actually used by any tools.

### 3. Destinations for Redis Cluster

Only redis cluster needs `[PACKAGE_MANAGER, GITHUB]`. Why?
- **Option A**: Special case in librarian.yaml (`destinations: [PACKAGE_MANAGER, GITHUB]`)
- **Option B**: Global config for redis cluster publishing to GitHub
- **Option C**: Remove GitHub destination and publish only to package manager

**Action**: Understand why redis cluster publishes to GitHub.

## Benefits

After migration:

1. **No service team approval needed** for language-specific changes
2. **~80% reduction** in service config file size (remove 1,520 redundant lines)
3. **Centralized control** - Librarian team owns all language settings
4. **Language expertise** - Language teams can evolve settings without coordination
5. **Simpler service configs** - Service teams focus on service-level concerns

## Risks

1. **Data loss during migration** - Must verify all language-specific settings are preserved
2. **Breaking existing tools** - If other tools read library_settings, they will break
3. **Incomplete understanding** - Empty destinations `[]` meaning unclear

## What Would Actually End Up in librarian.yaml

### Java (data/java/librarian.yaml)

**36 services** need Java-specific configuration:

```yaml
libraries:
  - '*'

  # Services with library_package AND service_class_names (6 services)

  - bigquerystorage:
      api: google/cloud/bigquery/storage/v1
      java:
        package: com.google.cloud.bigquery.storage.v1
        service_class_names:
          google.cloud.bigquery.storage.v1.BigQueryRead: BaseBigQueryRead

  - bigtable:
      api: google/bigtable/v2
      java:
        package: com.google.cloud.bigtable.data.v2
        service_class_names:
          google.bigtable.v2.Bigtable: BaseBigtableData

  - logging:
      api: google/logging/v2
      java:
        package: com.google.cloud.logging.v2
        service_class_names:
          google.logging.v2.ConfigServiceV2: Config
          google.logging.v2.LoggingServiceV2: Logging
          google.logging.v2.MetricsServiceV2: Metrics

  - accessapproval:
      api: google/cloud/accessapproval/v1
      java:
        package: com.google.cloud.accessapproval.v1
        service_class_names:
          google.cloud.accessapproval.v1.AccessApproval: AccessApprovalAdmin

  # Services with only library_package (30 services)

  - spanner:
      api: google/spanner/v1
      java:
        package: com.google.cloud.spanner.v1

  - dlp:
      api: google/privacy/dlp/v2
      java:
        package: com.google.cloud.dlp.v2

  - monitoring:
      api: google/monitoring/v3
      java:
        package: com.google.cloud.monitoring.v3

  - firestore:
      api: google/firestore/v1
      java:
        package: com.google.cloud.firestore.v1

  - datastore:
      api: google/datastore/v1
      java:
        package: com.google.cloud.datastore.v1

  # ... 25 more services with java.package
```

### Python (data/python/librarian.yaml)

**6 services** need Python-specific configuration:

```yaml
libraries:
  - '*'

  # Async REST I/O (2 services)

  - google-cloud-aiplatform:
      apis:
        - google/cloud/aiplatform/v1
        - google/cloud/aiplatform/v1beta1
      python:
        rest_async_io_enabled: true

  - google-cloud-documentai:
      apis:
        - google/cloud/documentai/v1
        - google/cloud/documentai/v1beta3
      python:
        rest_async_io_enabled: true

  # Unversioned package disabled (4 services)

  - google-ads-googleads:
      apis:
        - google/ads/googleads/v19
        - google/ads/googleads/v20
        - google/ads/googleads/v21
        - google/ads/googleads/v22
      python:
        unversioned_package_disabled: true

  - google-cloud-compute:
      api: google/cloud/compute/v1
      python:
        unversioned_package_disabled: true

  # ... 2 more with unversioned_package_disabled
```

### .NET (data/dotnet/librarian.yaml)

**11 services** need .NET-specific configuration:

```yaml
libraries:
  - '*'

  # Renamed services (8 services)

  - pubsub:
      api: google/pubsub/v1
      dotnet:
        renamed_services:
          Subscriber: SubscriberServiceApi
          Publisher: PublisherServiceApi

  - logging:
      api: google/logging/v2
      dotnet:
        renamed_services:
          LoggingServiceV2: LoggingServiceV2Client

  - dialogflow:
      api: google/cloud/dialogflow/v2beta1
      dotnet:
        renamed_services:
          Contexts: ContextsClient
          Agents: AgentsClient

  # ... 5 more with renamed_services

  # Renamed resources (3 services)

  - aiplatform:
      apis:
        - google/cloud/aiplatform/v1
        - google/cloud/aiplatform/v1beta1
      dotnet:
        renamed_resources:
          datalabeling.googleapis.com/Dataset: DataLabelingDataset
          automl.googleapis.com/Dataset: AutoMLDataset
          automl.googleapis.com/Model: AutoMLModel

  - documentai:
      apis:
        - google/cloud/documentai/v1
        - google/cloud/documentai/v1beta3
      dotnet:
        renamed_resources:
          documentai.googleapis.com/ProcessorVersion: DocumentAIProcessorVersion

  - language:
      api: google/cloud/language/v1
      dotnet:
        renamed_resources:
          language.googleapis.com/Model: LanguageModel
```

### Go (data/go/librarian.yaml)

**2 services** need Go-specific configuration:

```yaml
libraries:
  - '*'

  - pubsub:
      api: google/pubsub/v1
      go:
        renamed_services:
          Publisher: TopicAdmin
          Subscriber: SubscriptionAdmin

  - logging:
      api: google/logging/v2
      go:
        renamed_services:
          LoggingServiceV2: Client
```

### Node.js (data/node/librarian.yaml)

**2 services** need Node.js-specific configuration:

```yaml
libraries:
  - '*'

  - storage:
      api: google/storage/v2
      node:
        selective_methods:
          - google.storage.v2.Storage.GetBucket
          - google.storage.v2.Storage.ListBuckets
          - google.storage.v2.Storage.DeleteBucket
          - google.storage.v2.Storage.CreateBucket
          - google.storage.v2.Storage.LockBucketRetentionPolicy
          - google.storage.v2.Storage.UpdateBucket

  - bigtable:
      api: google/bigtable/v2
      node:
        selective_methods:
          - google.bigtable.v2.Bigtable.ReadRows
          - google.bigtable.v2.Bigtable.SampleRowKeys
          - google.bigtable.v2.Bigtable.MutateRow
          - google.bigtable.v2.Bigtable.MutateRows
```

### Ruby, PHP, C++ (data/{ruby,php,cpp}/librarian.yaml)

**0 services** need language-specific configuration - all use defaults.

```yaml
libraries:
  - '*'
```

### Special Cases

**Redis Cluster** (only service with non-default destinations):

```yaml
# Appears in ALL language librarian.yaml files
- redis:
    apis:
      - google/cloud/redis/cluster/v1
      - google/cloud/redis/cluster/v1beta1
    destinations: [PACKAGE_MANAGER, GITHUB]
```

## Summary Statistics

| Language | Services with Config | % of Total |
|----------|---------------------|------------|
| Java     | 36                  | 19%        |
| .NET     | 11                  | 6%         |
| Python   | 6                   | 3%         |
| Node.js  | 2                   | 1%         |
| Go       | 2                   | 1%         |
| Ruby     | 0                   | 0%         |
| PHP      | 0                   | 0%         |
| C++      | 0                   | 0%         |

Out of ~190 services with `library_settings`:
- **~10%** (57 services) need language-specific configuration
- **~90%** (133 services) use all defaults

## Next Steps

1. Investigate empty destinations (27 services)
2. Investigate launch_stage usage (is it needed?)
3. Understand redis cluster GitHub destinations
4. Extract all language-specific settings to migration file
5. Design migration script to update librarian.yaml files
6. Test migration on small subset (1 language, 5 services)
7. Roll out to all languages
