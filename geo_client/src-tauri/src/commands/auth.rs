use crate::storage::database;
use anyhow::Result;
use serde::{Deserialize, Serialize};
use tauri::command;

#[derive(Debug, Serialize, Deserialize)]
pub struct LoginRequest {
    pub username: String,
    pub password: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct LoginResponse {
    pub success: bool,
    pub token: String,
    pub expires_at: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct TokenInfo {
    pub token: String,
    pub expires_at: String,
    pub is_valid: bool,
}

/// 登录命令
#[command]
pub async fn login(
    username: String,
    password: String,
    api_base_url: String,
) -> Result<LoginResponse, String> {
    // 调用服务端API登录
    let client = reqwest::Client::new();
    let login_url = format!("{}/client/auth/login", api_base_url);
    
    let request_body = serde_json::json!({
        "username": username,
        "password": password,
    });
    
    match client
        .post(&login_url)
        .json(&request_body)
        .send()
        .await
    {
        Ok(response) => {
            if response.status().is_success() {
                match response.json::<LoginResponse>().await {
                    Ok(login_response) => {
                        if login_response.success {
                            // 保存token到数据库
                            if let Err(e) = database::save_auth_token(
                                &login_response.token,
                                &login_response.expires_at,
                            ) {
                                return Err(format!("保存token失败: {}", e));
                            }
                            Ok(login_response)
                        } else {
                            Err("登录失败：用户名或密码错误".to_string())
                        }
                    }
                    Err(e) => Err(format!("解析响应失败: {}", e)),
                }
            } else if response.status() == 401 {
                Err("登录失败：用户名或密码错误".to_string())
            } else {
                Err(format!("登录失败: HTTP {}", response.status()))
            }
        }
        Err(e) => Err(format!("网络请求失败: {}", e)),
    }
}

/// 获取当前token
#[command]
pub fn get_token() -> Result<TokenInfo, String> {
    match database::get_auth_token() {
        Ok(Some((token, expires_at))) => {
            let is_valid = !database::is_token_expired().unwrap_or(true);
            Ok(TokenInfo {
                token,
                expires_at,
                is_valid,
            })
        }
        Ok(None) => Err("未找到token".to_string()),
        Err(e) => Err(format!("获取token失败: {}", e)),
    }
}

/// 删除token（退出登录）
#[command]
pub fn logout() -> Result<(), String> {
    database::delete_auth_token().map_err(|e| format!("删除token失败: {}", e))
}

/// 检查token是否有效
#[command]
pub fn check_token_valid() -> Result<bool, String> {
    database::is_token_expired()
        .map(|expired| !expired)
        .map_err(|e| format!("检查token失败: {}", e))
}
