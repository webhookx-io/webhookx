# WebhookX [![Go Report Card](https://goreportcard.com/badge/github.com/webhookx-io/webhookx)](https://goreportcard.com/report/github.com/webhookx-io/webhookx) [![Build Status](https://github.com/webhookx-io/webhookx/actions/workflows/test.yml/badge.svg)](https://github.com/webhookx-io/webhookx/actions/workflows/test.yml) [![codecov](https://codecov.io/gh/webhookx-io/webhookx/graph/badge.svg?token=O4AQNRBJRF)](https://codecov.io/gh/webhookx-io/webhookx) [![Build Status](https://github.com/webhookx-io/webhookx/actions/workflows/lint.yml/badge.svg)](https://github.com/webhookx-io/webhookx/actions/workflows/lint.yml)

[![Join Slack](https://img.shields.io/badge/Slack-4285F4?logo=slack&logoColor=white)](https://join.slack.com/t/webhookx/shared_invite/zt-2o4b6hv45-mWm6_WUcQP9qEf1nOxhrrg)
[![Follow on Twitter](https://img.shields.io/badge/twitter-1DA1F2?logo=twitter&logoColor=white)](https://twitter.com/webhookx)

WebhookX is an open-source webhooks gateway for message receiving, processing, and delivering.


## Features

- **Admin API:** The admin API(:8080) provides a RESTful API for webhooks entities management.
- **Retries:** WebhookX automatically retries unsuccessful deliveries at configurable delays.
- **Fan out:** Events can be fan out to multiple destinations.
- **Declarative configuration(WIP):**  Managing your configuration through declarative configuration file, and be DevOps compliant.
- **Workspace:** Entities are isolated by workspace.

## Roadmap

- [ ] Data retention policy
- [ ] Insight admin APIs
- [ ] Observability(o11y) including tracing and metrics
- [ ] Declarative configuration management

#### Outbound

- [ ] Authentication
- [ ] Manually retry

#### Inbound

- [ ] Middlewares/Plugins
- [ ] Authentication
- [ ] Event Transformer

## Installation

```shell
$ docker compose up
```

```shell
$ curl http://localhost:8080
```


## Getting started

#### 1. Create an endpoint

```
$ curl -X POST http://localhost:8080/workspaces/default/endpoints \
  --header 'Content-Type: application/json' \
  --data '{
      "request": {
          "url": "https://httpbin.org/anything",
          "method": "POST",
          "headers": {
              "api-key": "secret"
          }
      },
      "events": [
          "charge.succeeded"
      ]
  }'
```

#### 2. Create a source

```
$ curl -X POST http://localhost:8080/workspaces/default/sources \
  --header 'accept: application/json' \
  --header 'Content-Type: application/json' \
  --data '{
    "path": "/",
    "methods": ["POST"]
  }'
```

#### 3. Send an event to proxy

```
$ curl -X POST http://localhost:8081 \
--header 'Content-Type: application/json' \
--data '{
    "event_type": "charge.succeeded",
    "data": {
        "key": "value"
    }
}'
```

#### 4. Retrieve delivery attempt

```
$ curl http://localhost:8080/workspaces/default/attempts
```

<details>
<summary>See response</summary>

```json
{
  "total": 1,
  "data": [
    {
      "id": "2lbkquwRPXEs6WFJqb8gPoiumgS",
      "event_id": "2lbkqvg8QBjyYuHO1V8f8TThLpv",
      "endpoint_id": "2lbkpcHXI7hpDoP22CP0fZ85zJY",
      "status": "SUCCESSFUL",
      "attempt_number": 1,
      "scheduled_at": 1725456357071,
      "attempted_at": 1725456357583,
      "error_code": null,
      "request": {
        "method": "POST",
        "url": "https://httpbin.org/anything",
        "headers": {
          "Api-Key": "secret",
          "Content-Type": "application/json; charset=utf-8",
          "User-Agent": "WebhookX/"
        },
        "body": "{\"key\": \"value\"}"
      },
      "response": {
        "status": 200,
        "headers": {
          "Access-Control-Allow-Credentials": "true",
          "Access-Control-Allow-Origin": "*",
          "Content-Length": "503",
          "Content-Type": "application/json",
          "Date": "Wed, 04 Sep 2024 13:26:02 GMT",
          "Server": "gunicorn/19.9.0"
        },
        "body": "{\n  \"args\": {}, \n  \"data\": \"{\\\"key\\\": \\\"value\\\"}\", \n  \"files\": {}, \n  \"form\": {}, \n  \"headers\": {\n    \"Accept-Encoding\": \"gzip\", \n    \"Api-Key\": \"secret\", \n    \"Content-Length\": \"16\", \n    \"Content-Type\": \"application/json; charset=utf-8\", \n    \"Host\": \"httpbin.org\", \n    \"User-Agent\": \"WebhookX/\", \n    \"X-Amzn-Trace-Id\": \"Root=1-66d85fe7-618479242937ff9d43b29e47\"\n  }, \n  \"json\": {\n    \"key\": \"value\"\n  }, \n  \"method\": \"POST\", \n  \"origin\": \"155.254.60.32\", \n  \"url\": \"https://httpbin.org/anything\"\n}\n"
      },
      "created_at": 1725456357071,
      "updated_at": 1725456357071
    }
  ]
}
```
</details>

Explore more API at [openapi.yml](/openapi.yml).

## Runtime dependencies

The gateway requires the following runtime dependencies to work:

- PostgreSQL(>=13): Lower versions of PostgreSQL may work, but have not been fully tested.
- Redis(>=4): Lower versions of Redis may work, but have not been fully tested.

## Sponsoring

## Contributing

We ❤️ pull requests

## Contributors

Thank you for your contribution to WebhookX!

[![Contributors](https://contrib.rocks/image?repo=webhookx-io/webhookx)](https://github.com/webhookx-io/webhookx/graphs/contributors)

[![Star History Chart](https://api.star-history.com/svg?repos=webhookx-io/webhookx&type=Date)](https://api.star-history.com/svg?repos=webhookx-io/webhookx&type=Date)

## License

```
Copyright 2024 WebhookX

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
