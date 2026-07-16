mod crypto;
mod vault;

use std::path::PathBuf;

use serde::{Deserialize, Serialize};
use tauri::AppHandle;
use zeroize::Zeroize;

use crypto::PasswordOptions;

const MAX_MASTER_PASSWORD_BYTES: usize = 1024;

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct CreateVaultRequest {
    master_password: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct ImportVaultRequest {
    path: String,
    master_password: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct ExportVaultRequest {
    path: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct GenerateRequest {
    master_password: String,
    #[serde(flatten)]
    options: PasswordOptions,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct VaultStatus {
    exists: bool,
    created_at: Option<String>,
    location: String,
}

#[tauri::command]
fn vault_status(app: AppHandle) -> Result<VaultStatus, String> {
    let path = vault::app_vault_path(&app)?;
    status_for_path(path)
}

#[tauri::command]
async fn create_vault(app: AppHandle, request: CreateVaultRequest) -> Result<VaultStatus, String> {
    validate_master_password(&request.master_password)?;
    let path = vault::app_vault_path(&app)?;
    tauri::async_runtime::spawn_blocking(move || {
        if path.exists() {
            return Err("A vault already exists on this device. Import from Vault transfer if you intend to replace it.".to_string());
        }
        let mut master_password = request.master_password;
        let created = vault::Vault::new(master_password.as_bytes());
        master_password.zeroize();
        let created = created?;
        vault::save(&path, &created)?;
        status_for_path(path)
    })
    .await
    .map_err(|_| "Vault creation stopped unexpectedly.".to_string())?
}

#[tauri::command]
async fn import_vault(app: AppHandle, request: ImportVaultRequest) -> Result<VaultStatus, String> {
    validate_master_password(&request.master_password)?;
    let destination = vault::app_vault_path(&app)?;
    tauri::async_runtime::spawn_blocking(move || {
        let source = PathBuf::from(request.path);
        let imported = vault::load(&source)?;
        let mut master_password = request.master_password;
        let decrypted = imported.decrypt_seed(master_password.as_bytes());
        master_password.zeroize();
        let mut seed = decrypted?;
        seed.zeroize();
        vault::save(&destination, &imported)?;
        status_for_path(destination)
    })
    .await
    .map_err(|_| "Vault import stopped unexpectedly.".to_string())?
}

#[tauri::command]
async fn export_vault(app: AppHandle, request: ExportVaultRequest) -> Result<(), String> {
    let source = vault::app_vault_path(&app)?;
    tauri::async_runtime::spawn_blocking(move || {
        let existing = vault::load(&source)?;
        vault::export(&PathBuf::from(request.path), &existing)
    })
    .await
    .map_err(|_| "Vault export stopped unexpectedly.".to_string())?
}

#[tauri::command]
async fn generate_password(app: AppHandle, request: GenerateRequest) -> Result<String, String> {
    validate_master_password(&request.master_password)?;
    let path = vault::app_vault_path(&app)?;
    tauri::async_runtime::spawn_blocking(move || {
        let existing = vault::load(&path)?;
        let mut master_password = request.master_password;
        let decrypted = existing.decrypt_seed(master_password.as_bytes());
        master_password.zeroize();
        let mut seed = decrypted?;
        let password = crypto::generate_password(&seed, &request.options);
        seed.zeroize();
        password
    })
    .await
    .map_err(|_| "Password generation stopped unexpectedly.".to_string())?
}

fn validate_master_password(master_password: &str) -> Result<(), String> {
    if master_password.is_empty() {
        return Err("Enter your master password.".into());
    }
    if master_password.len() > MAX_MASTER_PASSWORD_BYTES {
        return Err("The master password is unexpectedly long.".into());
    }
    Ok(())
}

fn status_for_path(path: PathBuf) -> Result<VaultStatus, String> {
    let location = path.to_string_lossy().into_owned();
    if !path.exists() {
        return Ok(VaultStatus {
            exists: false,
            created_at: None,
            location,
        });
    }
    let existing = vault::load(&path)?;
    Ok(VaultStatus {
        exists: true,
        created_at: Some(existing.created_at().to_string()),
        location,
    })
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_clipboard_manager::init())
        .plugin(tauri_plugin_dialog::init())
        .invoke_handler(tauri::generate_handler![
            vault_status,
            create_vault,
            import_vault,
            export_vault,
            generate_password
        ])
        .run(tauri::generate_context!())
        .expect("error while running acctpass");
}
