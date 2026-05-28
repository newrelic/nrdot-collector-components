# Use Case Extension

| Status        |                                                                                                               |
| ------------- |---------------------------------------------------------------------------------------------------------------|
| Stability     | [alpha](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#beta) |
| Distributions | [nrdot](https://github.com/newrelic/nrdot-collector-releases/releases)                                        |
| [Code Owners](https://github.com/newrelic/nrdot-collector-releases/blob/main/CONTRIBUTING.md) | newrelic/otelcomm                                                              |

The `usecase` extension appends a use case identifier to the `User-Agent` header of outgoing HTTP requests to improve analytics and troubleshooting support by New Relic.

## Configuration

The following settings are required:

- `id`: A static string. The use case identifier that will be appended to the User-Agent header.
  - Only alphanumeric characters, forward slash (`/`), underscore (`_`), hyphen (`-`), and period (`.`) are allowed.

## Configuration Example

```yaml
extensions:
  usecase:
    id: my-use-case
```