# Customize headers examples

The examples in this directory show you how to customize request headers in different languages.

- [AssemblyCcript](assemblyscript)
- [Rust](rust)
- [TinyGo](tinygo)


```yaml
# webhookx.yml
endpoints:
  - name: default-endpoint
    request:
      timeout: 10000
      url: https://httpbin.org/anything
      method: POST
    retry:
      strategy: fixed
      config:
        attempts: [0, 3600, 3600]
    events: [ "charge.succeeded" ]
    plugins:
      - name: wasm
        config:
          file: /path/to/your.wasm
          envs:
            foo: bar
            secret: secret-value
sources:
  - name: default-source
    path: /
    methods: [ "POST" ]
    response:
      code: 200
      content_type: application/json
      body: '{"message": "OK"}'
```
