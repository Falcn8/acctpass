use std::fs::{self, File};
use std::io::{Read, Write};
use std::path::{Path, PathBuf};

use base64::{Engine, engine::general_purpose::STANDARD};
use chacha20poly1305::aead::{Aead, KeyInit, Payload};
use chacha20poly1305::{XChaCha20Poly1305, XNonce};
use serde::{Deserialize, Serialize};
use zeroize::Zeroize;

use crate::crypto::{
    ArgonParams, DEFAULT_MEMORY_KIB, DEFAULT_THREADS, DEFAULT_TIME, SALT_SIZE, SEED_SIZE,
    VAULT_AAD, derive_key,
};

const VAULT_VERSION: u32 = 1;
const KDF_NAME: &str = "argon2id";
const CIPHER_NAME: &str = "xchacha20poly1305";
const NONCE_SIZE: usize = 24;
const MAX_VAULT_BYTES: u64 = 64 * 1024;
const MAX_MEMORY_KIB: u32 = 1024 * 1024;
const MAX_TIME: u32 = 10;
const MAX_THREADS: u32 = 16;

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Vault {
    version: u32,
    created_at: String,
    kdf: KdfConfig,
    cipher: CipherData,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
struct KdfConfig {
    name: String,
    memory_kib: u32,
    time: u32,
    threads: u32,
    salt_b64: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
struct CipherData {
    name: String,
    nonce_b64: String,
    ciphertext_b64: String,
}

impl Vault {
    pub fn new(master_password: &[u8]) -> Result<Self, String> {
        let mut seed = [0_u8; SEED_SIZE];
        let mut salt = [0_u8; SALT_SIZE];
        let mut nonce = [0_u8; NONCE_SIZE];
        getrandom::fill(&mut seed)
            .map_err(|_| "Could not create a secure random seed.".to_string())?;
        getrandom::fill(&mut salt)
            .map_err(|_| "Could not create a secure random salt.".to_string())?;
        getrandom::fill(&mut nonce)
            .map_err(|_| "Could not create a secure random nonce.".to_string())?;

        let params = ArgonParams {
            memory_kib: DEFAULT_MEMORY_KIB,
            time: DEFAULT_TIME,
            threads: DEFAULT_THREADS,
        };
        let mut key = derive_key(master_password, &salt, params)?;
        let cipher = XChaCha20Poly1305::new((&key).into());
        let encrypted = cipher.encrypt(
            XNonce::from_slice(&nonce),
            Payload {
                msg: &seed,
                aad: VAULT_AAD.as_bytes(),
            },
        );
        key.zeroize();
        seed.zeroize();
        let ciphertext = encrypted.map_err(|_| "Could not encrypt the vault seed.".to_string())?;

        Ok(Self {
            version: VAULT_VERSION,
            created_at: chrono::Utc::now().to_rfc3339_opts(chrono::SecondsFormat::Secs, true),
            kdf: KdfConfig {
                name: KDF_NAME.into(),
                memory_kib: params.memory_kib,
                time: params.time,
                threads: params.threads,
                salt_b64: STANDARD.encode(salt),
            },
            cipher: CipherData {
                name: CIPHER_NAME.into(),
                nonce_b64: STANDARD.encode(nonce),
                ciphertext_b64: STANDARD.encode(ciphertext),
            },
        })
    }

    pub fn created_at(&self) -> &str {
        &self.created_at
    }

    pub fn validate(&self) -> Result<(), String> {
        if self.version != VAULT_VERSION {
            return Err(format!("Vault version {} is not supported.", self.version));
        }
        if self.kdf.name != KDF_NAME {
            return Err("The vault uses an unsupported key derivation method.".into());
        }
        if self.cipher.name != CIPHER_NAME {
            return Err("The vault uses an unsupported cipher.".into());
        }
        if self.kdf.memory_kib == 0
            || self.kdf.memory_kib > MAX_MEMORY_KIB
            || self.kdf.time == 0
            || self.kdf.time > MAX_TIME
            || self.kdf.threads == 0
            || self.kdf.threads > MAX_THREADS
        {
            return Err("The vault contains unsafe Argon2id parameters.".into());
        }
        decode_sized("salt", &self.kdf.salt_b64, SALT_SIZE)?;
        decode_sized("nonce", &self.cipher.nonce_b64, NONCE_SIZE)?;
        let ciphertext = STANDARD
            .decode(&self.cipher.ciphertext_b64)
            .map_err(|_| "The vault ciphertext is not valid base64.".to_string())?;
        if ciphertext.len() != SEED_SIZE + 16 {
            return Err("The vault ciphertext has an invalid length.".into());
        }
        Ok(())
    }

    pub fn decrypt_seed(&self, master_password: &[u8]) -> Result<[u8; SEED_SIZE], String> {
        self.validate()?;
        let salt = STANDARD
            .decode(&self.kdf.salt_b64)
            .map_err(|_| "Invalid vault salt.".to_string())?;
        let nonce = STANDARD
            .decode(&self.cipher.nonce_b64)
            .map_err(|_| "Invalid vault nonce.".to_string())?;
        let ciphertext = STANDARD
            .decode(&self.cipher.ciphertext_b64)
            .map_err(|_| "Invalid vault ciphertext.".to_string())?;
        let mut key = derive_key(
            master_password,
            &salt,
            ArgonParams {
                memory_kib: self.kdf.memory_kib,
                time: self.kdf.time,
                threads: self.kdf.threads,
            },
        )?;
        let cipher = XChaCha20Poly1305::new((&key).into());
        let decrypted = cipher.decrypt(
            XNonce::from_slice(&nonce),
            Payload {
                msg: &ciphertext,
                aad: VAULT_AAD.as_bytes(),
            },
        );
        key.zeroize();
        let mut plaintext = decrypted.map_err(|_| {
            "The master password is incorrect, or the vault is damaged.".to_string()
        })?;
        if plaintext.len() != SEED_SIZE {
            plaintext.zeroize();
            return Err("The decrypted vault seed has an invalid length.".into());
        }
        let mut seed = [0_u8; SEED_SIZE];
        seed.copy_from_slice(&plaintext);
        plaintext.zeroize();
        Ok(seed)
    }
}

pub fn load(path: &Path) -> Result<Vault, String> {
    let metadata =
        fs::metadata(path).map_err(|_| "No vault was found on this device.".to_string())?;
    if metadata.len() > MAX_VAULT_BYTES {
        return Err("The vault file is unexpectedly large.".into());
    }
    let mut file = File::open(path).map_err(|_| "Could not open the vault.".to_string())?;
    let mut data = String::new();
    file.read_to_string(&mut data)
        .map_err(|_| "Could not read the vault.".to_string())?;
    parse(&data)
}

pub fn parse(data: &str) -> Result<Vault, String> {
    if data.len() as u64 > MAX_VAULT_BYTES {
        return Err("The vault file is unexpectedly large.".into());
    }
    let vault: Vault = serde_json::from_str(data)
        .map_err(|_| "This is not a valid acctpass vault.".to_string())?;
    vault.validate()?;
    Ok(vault)
}

pub fn save(path: &Path, vault: &Vault) -> Result<(), String> {
    vault.validate()?;
    let mut data =
        serde_json::to_vec_pretty(vault).map_err(|_| "Could not encode the vault.".to_string())?;
    data.push(b'\n');
    write_atomic(path, &data, true)
}

pub fn export(path: &Path, vault: &Vault) -> Result<(), String> {
    vault.validate()?;
    let mut data =
        serde_json::to_vec_pretty(vault).map_err(|_| "Could not encode the vault.".to_string())?;
    data.push(b'\n');
    write_atomic(path, &data, false)
}

pub fn app_vault_path(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    #[cfg(any(target_os = "android", target_os = "ios"))]
    {
        use tauri::Manager;
        return app
            .path()
            .app_config_dir()
            .map(|directory| directory.join("vault.json"))
            .map_err(|_| "Could not locate the app storage directory.".to_string());
    }
    #[cfg(not(any(target_os = "android", target_os = "ios")))]
    {
        let _ = app;
        dirs::config_dir()
            .map(|directory| directory.join("acctpass").join("vault.json"))
            .ok_or_else(|| "Could not locate the system configuration directory.".to_string())
    }
}

fn decode_sized(label: &str, value: &str, expected: usize) -> Result<Vec<u8>, String> {
    let decoded = STANDARD
        .decode(value)
        .map_err(|_| format!("The vault {label} is not valid base64."))?;
    if decoded.len() != expected {
        return Err(format!("The vault {label} has an invalid length."));
    }
    Ok(decoded)
}

fn write_atomic(path: &Path, data: &[u8], private: bool) -> Result<(), String> {
    let directory = path
        .parent()
        .ok_or_else(|| "The selected vault path is invalid.".to_string())?;
    fs::create_dir_all(directory)
        .map_err(|_| "Could not create the vault directory.".to_string())?;
    if fs::symlink_metadata(directory)
        .map(|metadata| metadata.file_type().is_symlink())
        .unwrap_or(false)
    {
        return Err("Refusing to use a symbolic-link vault directory.".into());
    }
    if fs::symlink_metadata(path)
        .map(|metadata| metadata.file_type().is_symlink())
        .unwrap_or(false)
    {
        return Err("Refusing to overwrite a symbolic-link vault file.".into());
    }

    #[cfg(unix)]
    if private {
        use std::os::unix::fs::PermissionsExt;
        fs::set_permissions(directory, fs::Permissions::from_mode(0o700))
            .map_err(|_| "Could not secure the vault directory.".to_string())?;
    }

    let mut temporary = tempfile::NamedTempFile::new_in(directory)
        .map_err(|_| "Could not create a temporary vault file.".to_string())?;
    temporary
        .write_all(data)
        .and_then(|_| temporary.as_file().sync_all())
        .map_err(|_| "Could not write the vault safely.".to_string())?;
    temporary
        .persist(path)
        .map_err(|_| "Could not replace the vault safely.".to_string())?;

    #[cfg(unix)]
    if private {
        use std::os::unix::fs::PermissionsExt;
        fs::set_permissions(path, fs::Permissions::from_mode(0o600))
            .map_err(|_| "Could not secure the vault file.".to_string())?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn new_vault_round_trips_through_disk() {
        let password = b"correct horse battery staple";
        let vault = Vault::new(password).expect("vault created");
        let expected_seed = vault.decrypt_seed(password).expect("vault unlocked");
        assert!(vault.decrypt_seed(b"wrong password").is_err());

        let directory = tempfile::tempdir().expect("temporary directory");
        let path = directory.path().join("acctpass").join("vault.json");
        save(&path, &vault).expect("vault saved");
        let loaded = load(&path).expect("vault loaded");
        let loaded_seed = loaded
            .decrypt_seed(password)
            .expect("loaded vault unlocked");
        assert_eq!(loaded_seed, expected_seed);

        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            assert_eq!(
                fs::metadata(&path)
                    .expect("vault metadata")
                    .permissions()
                    .mode()
                    & 0o777,
                0o600
            );
            assert_eq!(
                fs::metadata(path.parent().expect("vault directory"))
                    .expect("directory metadata")
                    .permissions()
                    .mode()
                    & 0o777,
                0o700
            );
        }
    }
}
