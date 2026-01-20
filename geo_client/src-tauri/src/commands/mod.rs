pub mod auth;
pub mod task;
pub mod provider;
pub mod storage;

use tauri::command;

#[command]
pub fn greet(name: &str) -> String {
    format!("Hello, {}! You've been greeted from Rust!", name)
}
