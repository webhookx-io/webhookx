# WebhookX

[![release](https://img.shields.io/github/v/release/webhookx-io/webhookx?color=green)](https://github.com/webhookx-io/webhookx/releases) [![test-workflow](https://github.com/webhookx-io/webhookx/actions/workflows/test.yml/badge.svg)](https://github.com/webhookx-io/webhookx/actions/workflows/test.yml) [![lint-workflow](https://github.com/webhookx-io/webhookx/actions/workflows/lint.yml/badge.svg)](https://github.com/webhookx-io/webhookx/actions/workflows/lint.yml) [![codecov](https://codecov.io/gh/webhookx-io/webhookx/graph/badge.svg?token=O4AQNRBJRF)](https://codecov.io/gh/webhookx-io/webhookx) [![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go) 

[![Join Slack](https://img.shields.io/badge/Join_WebhookX_👋-green?logo=slack&label=Slack)](https://join.slack.com/t/webhookx/shared_invite/zt-3ekjrpig4-1T~kOq8aUh7XXA5xiSGtXw) [![Twitter](https://img.shields.io/twitter/follow/WebhookX.svg?style=social)](https://twitter.com/intent/follow?screen_name=WebhookX)


WebhookX is an open-source webhooks gateway for message receiving, processing, and delivering.


## Architecture

<p align="center">
<img src="./architecture.png" alt="webhookx architecture" width="100%" />
</p>

## Posts

- [Introducing WebAssembly Plugin](https://docs.webhookx.io/blog/engineering/introducing-webassembly-plugin)
- [Function: A Way of Handling Webhook Verification Chaos](https://docs.webhookx.io/blog/engineering/function-a-way-of-handling-webhook-verification-chaos)



## Features

- **Admin API:** Expose a RESTful API on port `:8080` for managing WebhookX entities.
- **Retries:** Automatically retry unsuccessful deliveries with configurable delays.
- **Fan out:** Route events to multiple endpoints based on the event type.
- **Rate Limiting:** Protect the gateway ingestion and delivery endpoints from overload.
- **Declarative configuration:** Manage WebhookX through declarative configuration files to achieve GitOps/DevOps workflows.
- **Multi tenancy:**  Multiple workspaces. Each workspace provides the isolation of configuration entities.
- **Plugins:** Extend functionality via inbound and outbound plugins.
  - `webhookx-signature`: Sign outbound requests with HMAC(SHA-256) by adding `Webhookx-Signature` and `Webhookx-Timestamp` headers.
  - `wasm`: Transform outbound requests using high-level languages such as AssemblyScript, Rust or TinyGo. See [plugin/wasm](plugins/wasm).
  - `function`: Customize inbound behavior with JavaScript, e.g. signature verification or request body transformation.
- **Observability:** OpenTelemetry metrics and tracing for monitoring and troubleshooting.



## Installation

WebhookX binary provides you both Server and CLI command.

### macOS

```shell
brew tap webhookx-io/webhookx && brew install webhookx
```

### Linux | Windows

Download the binary distribution on [releases](https://github.com/webhookx-io/webhookx/releases).

## Get started

> [!TIP]
> This is only for evaluating features and is not suitable for production. For production deployment, please refer [documentation](https://docs.webhookx.io/docs/).

### 1. Start WebhookX using Docker Compose

The  [`docker-compose.yml`](https://github.com/webhookx-io/webhookx/blob/main/docker-compose.yml) provides a simple all-in-one (standalone) deployment for quick start.

```
curl -O https://raw.githubusercontent.com/webhookx-io/webhookx/main/docker-compose.yml && docker compose -f docker-compose.yml up
```

Once it's running, you will see HTTP 200 response

```
curl http://localhost:8080
```

```http
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Server: WebhookX/0.9.0

{
    "version": "0.9.0",
    "message": "Welcome to WebhookX",
  	"configuration": {}
}
```



### 2. Setup entities configuration

Let's use a sample declarative configuration [webhookx.sample.yml](https://github.com/webhookx-io/webhookx/blob/main/webhookx.sample.yml) in this quick start.

> You may need to install CLI `webhookx` first, see [Installation](#Installation).

```
webhookx admin sync webhookx.sample.yml
```

This command sends the configuration file to WebhookX via Admin API.

Once it is set up, you're ready to send events to WebhookX.



### 3. Send events to WebhookX

> The Ingestion is exposed on port  `:8081`

```
curl -X POST http://localhost:8081 \
--header 'Content-Type: application/json' \
--data '{
    "event_type": "charge.succeeded",
    "data": {
        "key": "value"
    }
}'
```

We sent a `charge.succeeded` event including `data` to WebhookX, and it will be routed and delivered to endpoints that are defined in the `webhookx.sample.yml`file.



### 4. Inspect delivery result

> [!TIP]
>
> Attempt object represents the delivery result of an event, and contains inspection information. 

> The Admin is exposed on port  `:8080`

Let's make a request to retrieve the attempt list

```
curl http://localhost:8080/workspaces/default/attempts
```

<details>
<summary>HTTP Response</summary>

```json
{
  "total": 1,
  "data": [
    {
      "id": "338lax8Xe774EhimzBukip37Zne",
      "event_id": "338las8UmbKJZl3aikM44ZWh71P",
      "endpoint_id": "338lQch7qdBqKvlVxXHcPTjLDTn",
      "status": "SUCCESSFUL",
      "attempt_number": 1,
      "scheduled_at": 1758706646768,
      "attempted_at": 1758706646819,
      "trigger_mode": "INITIAL",
      "exhausted": false,
      "error_code": null,
      "request": {
        "method": "POST",
        "url": "https://httpbin.org/anything",
        "headers": null,
        "body": null
      },
      "response": {
        "status": 200,
        "latency": 2402,
        "headers": null,
        "body": null
      },
      "created_at": 1758706646769,
      "updated_at": 1758706649223
    }
  ]
}
```
</details>

To inspect the data such as `request.headers`, `request.body`, `response.headers`, and  `response.body`, try

```
http://localhost:8080/workspaces/default/attempts/338lax8Xe774EhimzBukip37Zne
```

<details>
<summary>HTTP Response</summary>

```json
{
  "id": "338lax8Xe774EhimzBukip37Zne",
  "event_id": "338las8UmbKJZl3aikM44ZWh71P",
  "endpoint_id": "338lQch7qdBqKvlVxXHcPTjLDTn",
  "status": "SUCCESSFUL",
  "attempt_number": 1,
  "scheduled_at": 1758706646768,
  "attempted_at": 1758706646819,
  "trigger_mode": "INITIAL",
  "exhausted": false,
  "error_code": null,
  "request": {
    "method": "POST",
    "url": "https://httpbin.org/anything",
    "headers": {
      "Content-Type": "application/json; charset=utf-8",
      "User-Agent": "WebhookX/0.9.0",
      "Webhookx-Delivery-Id": "338lax8Xe774EhimzBukip37Zne",
      "Webhookx-Event-Id": "338las8UmbKJZl3aikM44ZWh71P",
      "Webhookx-Signature": "v1=37e25342d983c26d783eafe50fe170eaac731383439568e3354315c2e84c5173",
      "Webhookx-Timestamp": "1758706646",
      "X-Apikey": "secret"
    },
    "body": "{\n        \"key\": \"value\"\n    }"
  },
  "response": {
    "status": 200,
    "latency": 2402,
    "headers": {
      "Access-Control-Allow-Credentials": "true",
      "Access-Control-Allow-Origin": "*",
      "Content-Length": "778",
      "Content-Type": "application/json",
      "Date": "Wed, 24 Sep 2025 09:37:29 GMT",
      "Server": "gunicorn/19.9.0"
    },
    "body": "{\n  \"args\": {}, \n  \"data\": \"{\\n        \\\"key\\\": \\\"value\\\"\\n    }\", \n  \"files\": {}, \n  \"form\": {}, \n  \"headers\": {\n    \"Accept-Encoding\": \"gzip\", \n    \"Content-Length\": \"30\", \n    \"Content-Type\": \"application/json; charset=utf-8\", \n    \"Host\": \"httpbin.org\", \n    \"User-Agent\": \"WebhookX/0.9.0\", \n    \"Webhookx-Delivery-Id\": \"338lax8Xe774EhimzBukip37Zne\", \n    \"Webhookx-Event-Id\": \"338las8UmbKJZl3aikM44ZWh71P\", \n    \"Webhookx-Signature\": \"v1=37e25342d983c26d783eafe50fe170eaac731383439568e3354315c2e84c5173\", \n    \"Webhookx-Timestamp\": \"1758706646\", \n    \"X-Amzn-Trace-Id\": \"Root=1-68d3bbd7-195b190632c27c547358dbee\", \n    \"X-Apikey\": \"secret\"\n  }, \n  \"json\": {\n    \"key\": \"value\"\n  }, \n  \"method\": \"POST\", \n  \"origin\": \"0.0.0.0\", \n  \"url\": \"https://httpbin.org/anything\"\n}\n"
  },
  "created_at": 1758706646769,
  "updated_at": 1758706649223
}

```

</details>




To explore all the API, see [openapi.webhookx.io](https://openapi.webhookx.io).



## CLI

```
Usage:
  webhookx [command]

Available Commands:
  admin       Admin commands
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  migrations  
  start       Start server
  version     Print the version

Flags:
  -h, --help      help for webhookx
      --verbose   Verbose logging.

Use "webhookx [command] --help" for more information about a command.
```

- [CLI Reference](https://docs.webhookx.io/docs/cli)



## Runtime dependencies

Requires the following runtime dependencies to work:

- PostgreSQL(>=13): Lower versions of PostgreSQL may work, but have not been fully tested.
- Redis(>=6.2): Requires minimum version 6.2.



## Status and Compatibility

The project is currently under active development, hence breaking changes may be introduced in minor releases.

The public API will strictly follow semantic versioning after `v1.0.0`.



## Contributing

We ❤️ pull requests, and we’re continually working hard to make it as easy as possible for developers to contribute.

Thank you for your contribution to WebhookX!



## Stay Ahead

Star WebhookX on GitHub for instant updates on new releases.

<img src="./star.gif" alt="star" width="100%"/>



## License

WebhookX is under the Apache 2.0 license. See the [LICENSE](https://github.com/webhookx-io/webhookx/blob/main/LICENSE) file for details.
