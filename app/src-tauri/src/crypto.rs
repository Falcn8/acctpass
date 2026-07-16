use hmac::{Hmac, Mac};
use serde::{Deserialize, Serialize};
use sha2::Sha256;

pub const SEED_SIZE: usize = 32;
pub const SALT_SIZE: usize = 32;
pub const KEY_SIZE: usize = 32;
pub const VAULT_AAD: &str = "acctpass:vault:v1";
pub const DEFAULT_MEMORY_KIB: u32 = 64 * 1024;
pub const DEFAULT_TIME: u32 = 3;
pub const DEFAULT_THREADS: u32 = 1;

const LOWER: &str = "abcdefghijklmnopqrstuvwxyz";
const UPPER: &str = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";
const DIGITS: &str = "0123456789";
pub const SYMBOLS: &str = "!@#$%^&*()-_=+[]{}?";
const NO_SYMBOLS_ALPHABET: &str = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PasswordOptions {
    pub platform: String,
    pub email: String,
    pub counter: u32,
    pub length: usize,
    pub symbols: bool,
    #[serde(default, alias = "allowed_symbols")]
    pub allowed_symbols: String,
}

#[derive(Debug, Clone, Copy)]
pub struct ArgonParams {
    pub memory_kib: u32,
    pub time: u32,
    pub threads: u32,
}

pub fn derive_key(
    password: &[u8],
    salt: &[u8],
    params: ArgonParams,
) -> Result<[u8; KEY_SIZE], String> {
    let params = argon2::Params::new(
        params.memory_kib,
        params.time,
        params.threads,
        Some(KEY_SIZE),
    )
    .map_err(|_| "Invalid Argon2id parameters in vault.".to_string())?;
    let argon = argon2::Argon2::new(argon2::Algorithm::Argon2id, argon2::Version::V0x13, params);
    let mut key = [0_u8; KEY_SIZE];
    argon
        .hash_password_into(password, salt, &mut key)
        .map_err(|_| "Could not derive the vault key.".to_string())?;
    Ok(key)
}

pub fn generate_password(seed: &[u8], options: &PasswordOptions) -> Result<String, String> {
    if seed.len() != SEED_SIZE {
        return Err(format!("Seed must be {SEED_SIZE} bytes."));
    }
    if options.counter < 1 {
        return Err("Counter must be at least 1.".into());
    }
    if options.length < 1 || options.length > 256 {
        return Err("Length must be between 1 and 256.".into());
    }

    let platform = normalize_identity(&options.platform);
    let email = normalize_identity(&options.email);
    if platform.is_empty() {
        return Err("Enter a service name.".into());
    }
    if email.is_empty() {
        return Err("Enter an email or username.".into());
    }

    let allowed_symbols = resolve_allowed_symbols(options.symbols, &options.allowed_symbols)?;
    let alphabet = format!("{NO_SYMBOLS_ALPHABET}{allowed_symbols}");
    let mut base_context = format!(
        "acctpass:v1|platform={platform}|email={email}|counter={}|length={}|symbols={}",
        options.counter, options.length, options.symbols
    );
    if options.symbols && allowed_symbols != SYMBOLS {
        base_context.push_str("|allowed-symbols=");
        base_context.push_str(&allowed_symbols);
    }

    let required_classes = if options.symbols { 4 } else { 3 };
    let enforce_classes = options.length >= required_classes;
    for attempt in 0..10_000 {
        let context = if attempt == 0 {
            base_context.clone()
        } else {
            format!("{base_context}|attempt={attempt}")
        };
        let password = password_from_context(seed, &context, &alphabet, options.length)?;
        if !enforce_classes || satisfies_rules(&password, &allowed_symbols) {
            return Ok(password);
        }
    }
    Err("Could not generate a password satisfying the selected rules.".into())
}

fn normalize_identity(value: &str) -> String {
    value.trim().to_lowercase()
}

fn resolve_allowed_symbols(symbols: bool, custom: &str) -> Result<String, String> {
    if !symbols {
        if !custom.is_empty() {
            return Err("Allowed symbols cannot be set when symbols are disabled.".into());
        }
        return Ok(String::new());
    }
    if custom.is_empty() {
        return Ok(SYMBOLS.to_string());
    }
    normalize_symbol_subset(custom)
}

fn normalize_symbol_subset(value: &str) -> Result<String, String> {
    if let Some(unsupported) = value
        .chars()
        .find(|character| !SYMBOLS.contains(*character))
    {
        return Err(format!("Unsupported symbol {unsupported:?}."));
    }
    let normalized: String = SYMBOLS
        .chars()
        .filter(|character| value.contains(*character))
        .collect();
    if normalized.is_empty() {
        return Err("Choose at least one symbol.".into());
    }
    Ok(normalized)
}

fn password_from_context(
    seed: &[u8],
    context: &str,
    alphabet: &str,
    length: usize,
) -> Result<String, String> {
    let alphabet = alphabet.as_bytes();
    let mut output = Vec::with_capacity(length);
    let mut block = 0_u32;
    while output.len() < length {
        let input = if block == 0 {
            context.to_string()
        } else {
            format!("{context}|block={block}")
        };
        let mut mac = <Hmac<Sha256> as Mac>::new_from_slice(seed)
            .map_err(|_| "Could not initialize password derivation.".to_string())?;
        mac.update(input.as_bytes());
        let digest = mac.finalize().into_bytes();
        rejection_sample(&digest, alphabet, length - output.len(), &mut output);
        block += 1;
    }
    String::from_utf8(output).map_err(|_| "Password output was invalid.".into())
}

fn rejection_sample(random: &[u8], alphabet: &[u8], needed: usize, output: &mut Vec<u8>) {
    let limit = 256 - (256 % alphabet.len());
    for byte in random {
        if output.len() >= output.capacity() || needed == 0 {
            break;
        }
        if usize::from(*byte) >= limit {
            continue;
        }
        output.push(alphabet[usize::from(*byte) % alphabet.len()]);
        if output.len() == output.capacity() {
            break;
        }
    }
}

fn satisfies_rules(password: &str, allowed_symbols: &str) -> bool {
    password.chars().any(|c| LOWER.contains(c))
        && password.chars().any(|c| UPPER.contains(c))
        && password.chars().any(|c| DIGITS.contains(c))
        && (allowed_symbols.is_empty() || password.chars().any(|c| allowed_symbols.contains(c)))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::vault::Vault;
    use base64::{Engine, engine::general_purpose::STANDARD};

    #[derive(Deserialize)]
    struct Fixture {
        seed_b64: String,
        master_password: String,
        vault: Vault,
        passwords: Vec<PasswordVector>,
    }

    #[derive(Deserialize)]
    struct PasswordVector {
        #[serde(flatten)]
        options: PasswordOptions,
        expected: String,
    }

    #[test]
    fn matches_go_compatibility_vectors() {
        let fixture: Fixture =
            serde_json::from_str(include_str!("../../../compatibility/vectors.json"))
                .expect("valid fixture");
        let seed = STANDARD.decode(&fixture.seed_b64).expect("valid seed");
        let decrypted = fixture
            .vault
            .decrypt_seed(fixture.master_password.as_bytes())
            .expect("fixture vault decrypts");
        assert_eq!(decrypted.as_slice(), seed.as_slice());

        for vector in fixture.passwords {
            let password = generate_password(&seed, &vector.options).expect("password generated");
            assert_eq!(password, vector.expected);
        }
    }
}
