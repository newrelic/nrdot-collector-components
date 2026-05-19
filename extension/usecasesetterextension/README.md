# Use Case Setter Extension

| Status        |                                                                                                               |
| ------------- |---------------------------------------------------------------------------------------------------------------|
| Stability     | [beta](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#beta) |
| Distributions | [nrdot](https://github.com/newrelic/nrdot-collector-releases/releases)                                        |
| [Code Owners](https://github.com/newrelic/nrdot-collector-releases/blob/main/CONTRIBUTING.md) | [@emiliaFer](https://www.github.com/emiliaFer)                                                                |

The `usecase_setter` extension resolves a use case identifier from one of three sources: a static configuration value, request context metadata, or a request attribute. The resolved value can optionally fall back to a default if the primary source yields no result.

## Configuration

The following settings are required:

- `usecase`: A single use case configuration object with the following properties:
    - `value`: A static string. The use case identifier is taken directly from this value.
    - `from_context`: The use case identifier is looked up from request context metadata using this string as the key (e.g. a metadata header name passed by a receiver).
    - `from_attribute`: The use case identifier is taken from a request attribute using this string as the key.
    - `default_value` (optional): Fallback value used when `from_context` or `from_attribute` yields no result.

Exactly one of `value`, `from_context`, or `from_attribute` must be set. They are mutually exclusive.

In order for `from_context` to work, other components in the pipeline must be configured appropriately:

- Receivers must have `include_metadata: true` so that metadata keys are forwarded through the pipeline.
- If a [batch processor][batch-processor] is present, configure it to [preserve client metadata][batch-processor-preserve-metadata] by adding the relevant key to `metadata_keys`.

## Configuration Examples

**Static value:**

```yaml
extensions:
  usecase_setter:
    usecase:
      value: my-use-case
```

**From request context with a fallback:**

```yaml
extensions:
  usecase_setter:
    usecase:
      from_context: tenant_id
      default_value: default-tenant

receivers:
  otlp:
    protocols:
      http:
        include_metadata: true

processors:
  batch:
    metadata_keys:
      - tenant_id

service:
  extensions: [usecase_setter]
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp]
```

**From a request attribute:**

```yaml
extensions:
  usecase_setter:
    usecase:
      from_attribute: use_case_attr
      default_value: default-use-case
```

[batch-processor]: https://github.com/open-telemetry/opentelemetry-collector/tree/main/processor/batchprocessor/README.md
[batch-processor-preserve-metadata]: https://github.com/open-telemetry/opentelemetry-collector/tree/main/processor/batchprocessor/README.md#batching-and-client-metadata