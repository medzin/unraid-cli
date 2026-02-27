use anyhow::{Context, Result};
use directories::ProjectDirs;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerConfig {
    pub url: String,
    pub api_key: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct Config {
    #[serde(default)]
    pub default: Option<String>,
    #[serde(default)]
    pub servers: HashMap<String, ServerConfig>,
}

impl Config {
    pub fn load() -> Result<Self> {
        let path = Self::config_path()?;

        if !path.exists() {
            return Ok(Self::default());
        }

        let content = fs::read_to_string(&path)
            .with_context(|| format!("Failed to read config file: {}", path.display()))?;

        toml::from_str(&content)
            .with_context(|| format!("Failed to parse config file: {}", path.display()))
    }

    pub fn save(&self) -> Result<()> {
        let path = Self::config_path()?;

        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).with_context(|| {
                format!("Failed to create config directory: {}", parent.display())
            })?;
        }

        let content = toml::to_string_pretty(self).context("Failed to serialize config")?;

        fs::write(&path, content)
            .with_context(|| format!("Failed to write config file: {}", path.display()))
    }

    pub fn config_path() -> Result<PathBuf> {
        let proj_dirs =
            ProjectDirs::from("", "", "unraid").context("Failed to determine config directory")?;

        Ok(proj_dirs.config_dir().join("config.toml"))
    }

    pub fn get_server(&self, name: Option<&str>) -> Option<&ServerConfig> {
        let server_name = name.or(self.default.as_deref())?;
        self.servers.get(server_name)
    }

    pub fn add_server(&mut self, name: String, url: String, api_key: String) {
        self.servers.insert(name, ServerConfig { url, api_key });
    }

    pub fn remove_server(&mut self, name: &str) -> bool {
        let removed = self.servers.remove(name).is_some();

        // Clear default if we removed the default server
        if self.default.as_deref() == Some(name) {
            self.default = None;
        }

        removed
    }

    pub fn set_default(&mut self, name: &str) -> Result<()> {
        if !self.servers.contains_key(name) {
            anyhow::bail!("Server '{name}' not found in configuration");
        }
        self.default = Some(name.to_string());
        Ok(())
    }
}

/// Resolved server configuration from all sources
#[derive(Debug, Clone)]
pub struct ResolvedConfig {
    pub url: String,
    pub api_key: String,
}

impl ResolvedConfig {
    /// Resolve configuration from CLI args, environment variables, and config file.
    /// Priority: CLI args > env vars > config file
    pub fn resolve(
        cli_server: Option<&str>,
        cli_url: Option<&str>,
        cli_api_key: Option<&str>,
    ) -> Result<Self> {
        // If URL and API key are provided directly, use them
        if let (Some(url), Some(api_key)) = (cli_url, cli_api_key) {
            return Ok(Self {
                url: url.to_string(),
                api_key: api_key.to_string(),
            });
        }

        // Check environment variables
        let env_url = std::env::var("UNRAID_URL").ok();
        let env_api_key = std::env::var("UNRAID_API_KEY").ok();
        let env_server = std::env::var("UNRAID_SERVER").ok();

        // If both env vars are set, use them
        if let (Some(url), Some(api_key)) = (&env_url, &env_api_key) {
            return Ok(Self {
                url: cli_url.unwrap_or(url).to_string(),
                api_key: cli_api_key.unwrap_or(api_key).to_string(),
            });
        }

        // Load config file
        let config = Config::load()?;

        // Determine which server to use
        let server_name = cli_server.or(env_server.as_deref());

        if let Some(server) = config.get_server(server_name) {
            return Ok(Self {
                url: cli_url.unwrap_or(&server.url).to_string(),
                api_key: cli_api_key.unwrap_or(&server.api_key).to_string(),
            });
        }

        anyhow::bail!(
            "No server configured. Use 'unraid config add <name>' to add a server, \
            or set UNRAID_URL and UNRAID_API_KEY environment variables."
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn sample_config() -> Config {
        let mut config = Config::default();
        config.add_server(
            "tower".to_string(),
            "https://192.168.1.100".to_string(),
            "key-tower".to_string(),
        );
        config.add_server(
            "backup".to_string(),
            "https://192.168.1.101".to_string(),
            "key-backup".to_string(),
        );
        config.default = Some("tower".to_string());
        config
    }

    #[test]
    fn default_config_has_no_servers_and_no_default() {
        let config = Config::default();
        assert!(config.default.is_none());
        assert!(config.servers.is_empty());
    }

    #[test]
    fn add_server_stores_url_and_api_key() {
        let mut config = Config::default();
        config.add_server(
            "test".to_string(),
            "https://example.com".to_string(),
            "api-key".to_string(),
        );

        assert_eq!(config.servers.len(), 1);
        let server = config.servers.get("test").unwrap();
        assert_eq!(server.url, "https://example.com");
        assert_eq!(server.api_key, "api-key");
    }

    #[test]
    fn add_server_with_existing_name_overwrites_previous_config() {
        let mut config = sample_config();
        config.add_server(
            "tower".to_string(),
            "https://new-url.com".to_string(),
            "new-key".to_string(),
        );

        let server = config.servers.get("tower").unwrap();
        assert_eq!(server.url, "https://new-url.com");
        assert_eq!(server.api_key, "new-key");
    }

    #[test]
    fn get_server_with_name_returns_matching_server() {
        let config = sample_config();

        let server = config.get_server(Some("backup")).unwrap();
        assert_eq!(server.url, "https://192.168.1.101");
    }

    #[test]
    fn get_server_without_name_returns_default_server() {
        let config = sample_config();

        let server = config.get_server(None).unwrap();
        assert_eq!(server.url, "https://192.168.1.100"); // tower is default
    }

    #[test]
    fn get_server_without_name_returns_none_when_no_default_set() {
        let mut config = sample_config();
        config.default = None;

        assert!(config.get_server(None).is_none());
    }

    #[test]
    fn get_server_with_unknown_name_returns_none() {
        let config = sample_config();
        assert!(config.get_server(Some("nonexistent")).is_none());
    }

    #[test]
    fn remove_server_deletes_server_and_returns_true() {
        let mut config = sample_config();

        assert!(config.remove_server("backup"));
        assert_eq!(config.servers.len(), 1);
        assert!(config.servers.get("backup").is_none());
    }

    #[test]
    fn remove_server_preserves_default_when_removing_non_default_server() {
        let mut config = sample_config();

        config.remove_server("backup");

        assert_eq!(config.default, Some("tower".to_string()));
    }

    #[test]
    fn remove_server_returns_false_for_unknown_server() {
        let mut config = sample_config();

        assert!(!config.remove_server("nonexistent"));
        assert_eq!(config.servers.len(), 2);
    }

    #[test]
    fn remove_server_clears_default_when_removing_default_server() {
        let mut config = sample_config();
        assert_eq!(config.default, Some("tower".to_string()));

        config.remove_server("tower");

        assert!(config.default.is_none());
    }

    #[test]
    fn set_default_updates_default_to_existing_server() {
        let mut config = sample_config();

        config.set_default("backup").unwrap();
        assert_eq!(config.default, Some("backup".to_string()));
    }

    #[test]
    fn set_default_returns_error_for_unknown_server() {
        let mut config = sample_config();

        let result = config.set_default("nonexistent");
        assert!(result.is_err());
        assert!(
            result
                .unwrap_err()
                .to_string()
                .contains("not found in configuration")
        );
    }

    #[test]
    fn config_survives_toml_roundtrip_serialization() {
        let config = sample_config();
        let toml_str = toml::to_string(&config).unwrap();
        let parsed: Config = toml::from_str(&toml_str).unwrap();

        assert_eq!(parsed.default, config.default);
        assert_eq!(parsed.servers.len(), config.servers.len());
        assert_eq!(
            parsed.servers.get("tower").unwrap().url,
            config.servers.get("tower").unwrap().url
        );
    }

    #[test]
    fn deserialize_empty_toml_creates_default_config() {
        let toml_str = "";
        let config: Config = toml::from_str(toml_str).unwrap();

        assert!(config.default.is_none());
        assert!(config.servers.is_empty());
    }

    #[test]
    fn deserialize_toml_with_only_default_preserves_default_with_empty_servers() {
        let toml_str = r#"
default = "myserver"
"#;
        let config: Config = toml::from_str(toml_str).unwrap();

        assert_eq!(config.default, Some("myserver".to_string()));
        assert!(config.servers.is_empty());
    }

    #[test]
    fn resolve_config_uses_cli_args_when_provided() {
        let resolved =
            ResolvedConfig::resolve(None, Some("https://cli-url.com"), Some("cli-key")).unwrap();

        assert_eq!(resolved.url, "https://cli-url.com");
        assert_eq!(resolved.api_key, "cli-key");
    }
}
