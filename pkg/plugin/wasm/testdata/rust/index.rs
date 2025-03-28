// cargo build --release --target wasm32-unknown-unknown

extern crate alloc;
extern crate core;
extern crate wee_alloc;

use std::mem::MaybeUninit;
use core::slice;
use core::str;

#[link(wasm_import_module = "env")]
extern "C" {
    fn get_request_json(return_value_data: *mut usize, return_value_size: *mut usize) -> i32;
    fn set_request_json(value_data: *const u8, value_size: usize) -> i32;
    fn log(log_level: i32, str_value: *const u8, str_size: i32) -> i32;
}

const OK: i32 = 0;
#[repr(i32)]
enum LogLevel {
    Debug = 0,
    Info = 1,
    Warn = 2,
    Error = 3,
}


/// Set the global allocator to the WebAssembly optimized one.
#[global_allocator]
static ALLOC: wee_alloc::WeeAlloc = wee_alloc::WeeAlloc::INIT;

#[cfg_attr(all(target_arch = "wasm32"), export_name = "allocate")]
pub extern "C" fn allocate(size: i32) -> *mut u8 {
    let vec: Vec<MaybeUninit<u8>> = vec![MaybeUninit::uninit(); size as usize];
    Box::into_raw(vec.into_boxed_slice()) as *mut u8
}


fn log_string(level: LogLevel, str: &str) {
    unsafe {
        log(level as i32, str.as_ptr(), str.len() as i32);
    }
}

#[no_mangle]
pub extern "C" fn transform() -> i32 {
    let mut request_json_ptr: usize = 0;
    let mut request_json_size: usize = 0;

    let status = unsafe { get_request_json(&mut request_json_ptr, &mut request_json_size) };
    if status != OK {
        return 0;
    }

    let request_json = unsafe {
        let slice = slice::from_raw_parts(request_json_ptr as *const u8, request_json_size);
        str::from_utf8_unchecked(slice)
    };

    log_string(LogLevel::Debug, request_json);
    log_string(LogLevel::Info, "a info message");
    log_string(LogLevel::Warn, "a warn message");
    log_string(LogLevel::Error, "a error message");

    let json = r#"{"url":"https://httpbin.org/anything","method":"POST","headers":{"foo":"bar"},"payload":"{}"}"#;

    let status = unsafe { set_request_json(json.as_ptr(), json.len()) };
    if status != OK {
        return 0;
    }

    1
}
