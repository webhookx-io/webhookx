// asc index.ts --target release

import { __pin } from "rt/itcms";

const OK = 0
enum LogLevel { debug, info, warn, error}

export function allocate(size: i32): usize {
    let buffer = new ArrayBuffer(size);
    let ptr = changetype<usize>(buffer);
    return __pin(ptr);
}

// @ts-ignore: decorator
@global
export function abort_proc_exit(message: string | null, file_name: string | null, line_number: u32, column_number: u32): void {
    let logMessage = "abort: ";
    if (message !== null) {
        logMessage += message.toString();
    }
    if (file_name !== null) {
        logMessage += " at: " + file_name.toString() + "(" + line_number.toString() + ":" + column_number.toString() + ")";
    }
    log_string(LogLevel.error, logMessage);
}

// @ts-ignore: decorator
@external("env", "get_request_json")
declare function get_request_json(return_value_data: usize, return_value_size: usize): i32;

// @ts-ignore: decorator
@external("env", "set_request_json")
declare function set_request_json(value_data: usize, value_size: usize): i32;

// @ts-ignore: decorator
@external("env", "log")
declare function log(log_level: i32, str_value: usize, str_size: i32): i32;


function log_string(level: LogLevel, str: string): void {
    let encoded = String.UTF8.encode(str)
    log(level, changetype<usize>(encoded), encoded.byteLength)
}

export function transform(): i32 {
    let request_json_ptr = heap.alloc(4); // var to store json pointer
    let request_json_size = heap.alloc(4); // var to store json size

    let status = get_request_json(changetype<usize>(request_json_ptr), changetype<usize>(request_json_size));
    if (status != OK) {
        return 0
    }

    let request_json = String.UTF8.decodeUnsafe(load<usize>(request_json_ptr), load<usize>(request_json_size));
    log_string(LogLevel.debug, request_json)

    let json = '{"url":"http://localhost:9999/anything?debug=true","method":"PUT","headers":{"foo":"bar"},"payload":"{\\"key\\": \\"value\\", \\"other\\": \\"other-value\\"}"}'
    let encoded = String.UTF8.encode(json)
    status = set_request_json(changetype<usize>(encoded), encoded.byteLength);
    if (status != OK) {
        return 0
    }

    return 1
}
