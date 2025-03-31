# Wasm Plugin

This plugin allow you to  customize delivery requests including URL, method, headers, and payload.

The Application Binary Interface (ABI) is defined in [versions](./versions).

For more examples, please see [examples/wasm](/examples/wasm).


### Configuration

| Name       | Type   | Description                                                  |
|-------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------|
| `file`</br> *required\** | string                                                                                                                                   | The filename of wasm module. |
| `envs`</br> *optional*  | map                                                                                                                             | The environment variables that are exposed to the wasm module. |



### Configuration examples

```yaml
name: wasm
enabled: true
config:
  file: /path/to/your.wasm
  envs:
  foo: bar
```

