pub mod database;
pub mod browser;

/// 初始化存储模块
pub fn init() -> anyhow::Result<()> {
    database::init_db()?;
    Ok(())
}
