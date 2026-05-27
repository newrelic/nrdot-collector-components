# Use Case Extension

| Status        |                                                                                                               |
| ------------- |---------------------------------------------------------------------------------------------------------------|
| Stability     | [beta](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#beta) |
| Distributions | [nrdot](https://github.com/newrelic/nrdot-collector-releases/releases)                                        |
| [Code Owners](https://github.com/newrelic/nrdot-collector-releases/blob/main/CONTRIBUTING.md) | [@emiliaFer](https://www.github.com/emiliaFer)                                                                |

The `usecase` extension appends a use case identifier to the `User-Agent` header of outgoing HTTP requests.

## Configuration

The following settings are required:

- `usecase`: A single use case configuration object with the following property:
    - `value`: A static string. The use case identifier is taken directly from this value.

## Configuration Example

```yaml
extensions:
  usecase:
    usecase:
      value: my-use-case
```