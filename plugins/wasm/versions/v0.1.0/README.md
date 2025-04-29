# Wasm plugin ABI v0.1.0 specification



### Callbacks exposed by the Wasm module

- [transform](#transform)
- [allocate](#allocate)



### Functions exposed by the Host

- [get_request_json](#get_request_json)
- [set_request_json](#set_request_json)
- [log](#log)



## transform

#### Params

None

#### Returns

| type | desc   |
| ---- | ------ |
| i32  | status |

The entry point of wasm plugin.

Called to transform a request before sending it to the endpoint.

Returning 1 indicates success, otherwise failure.



##  allocate

#### Params

| name    | type | desc |
| -------  | ---------------------- | -------  |
| size | i32 | memory size |

#### Returns

| type | desc           |
| ---- | -------------- |
| i32  | memory address |

Called to allocate continuous memory buffer of `size` using the in-VM memory allocator.

Plugin must return memory address pointing to the start of the allocated memory.

Returning 0 indicates failure.



## get_request_json

#### Params

| name              | type | desc                                   |
| ----------------- | ---- | -------------------------------------- |
| return_value_data | i32  | memory address to store the value      |
| return_value_size | i32  | memory address to store the value size |

#### Returns

| type                    | desc   |
| ----------------------- | ------ |
| i32 ([status](#status)) | status |

Retrieves the [request_json](#request_json) string that stored in `return_value_data` and size stored in `return_value_size`.

Returned value is:

- `OK` on success.
- `INVALID_MEMORY_ACCESS` when `return_value_data` and/or `return_value_size` point to invalid memory address.


## set_request_json

#### Params

| name       | type | desc                    |
| ---------- | ---- | ----------------------- |
| value_data | i32  | memory address of value |
| value_size | i32  | value size              |

#### Returns

| type                    | desc   |
| ----------------------- | ------ |
| i32 ([status](#status)) | status |

Sets a new [request_json](#request_json).



Returned value is:

- `OK` on success.
- `INVALID_MEMORY_ACCESS` when `value_data` and/or `value_size` point to invalid memory address.
- `INVALID_JSON` when json is invalid.


## log

#### Params

| name      | type                        | desc                     |
|-----------| --------------------------- | ------------------------ |
| log_level | i32 [log_level](#log_level) | log level                |
| str_data  | i32                         | memory address of string |
| str_size  | i32                         | str size                 |

#### Returns

| type                    | desc   |
| ----------------------- | ------ |
| i32 ([status](#status)) | status |

Logs string (`str_data`, `str_size`) at the `log_level`.

Returned value is:

- `OK` on success.
- `BAD_ARGUMENT` for unknown log_level.
- `INVALID_MEMORY_ACCESS` when `str_data` and/or `str_size` point to invalid memory address.


## Types

#### `status`

- `OK` = `0`
- `INTERNAL_FAILURE` = `1`
- `BAD_ARGUMENT` = `2`
- `INVALID_MEMORY_ACCESS` = `3`
- `INVALID_JSON` = `11`

#### `log_level`

- `DEBUG` = `0`
- `INFO` = `1`
- `WARN` = `2`
- `ERROR` = `3`

#### `request_json`

- `url`: string
-  `method`: string
- `headers`: object
- `payload`: string

example:

```json
{
    "url": "https://example.com",
    "method": "POST",
    "headers": {
        "x-foo": "bar"
    },
    "payload": "{}"
}
```

