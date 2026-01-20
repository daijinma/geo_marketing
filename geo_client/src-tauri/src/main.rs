// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod commands;
mod models;
mod storage;

use commands::{auth, greet};

fn main() {
    // 初始化数据库
    if let Err(e) = storage::init() {
        eprintln!("初始化数据库失败: {}", e);
    }
    
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![
            greet,
            auth::login,
            auth::get_token,
            auth::logout,
            auth::check_token_valid,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
