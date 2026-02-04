use anyhow::Result;
use clap::Subcommand;

use crate::config::Config;

#[derive(Debug, Subcommand)]
pub enum ConfigCommands {
    /// Add a new server configuration
    Add {
        /// Name for this server configuration
        name: String,
        /// Server URL (e.g., `https://192.168.1.100`)
        #[arg(long)]
        url: String,
        /// API key for authentication
        #[arg(long)]
        api_key: String,
    },
    /// Remove a server configuration
    Remove {
        /// Name of the server to remove
        name: String,
    },
    /// Set the default server
    Default {
        /// Name of the server to set as default
        name: String,
    },
    /// List all configured servers
    List,
}

pub fn handle_config_command(cmd: ConfigCommands) -> Result<()> {
    match cmd {
        ConfigCommands::Add { name, url, api_key } => add_server(&name, url, api_key),
        ConfigCommands::Remove { name } => remove_server(&name),
        ConfigCommands::Default { name } => set_default(&name),
        ConfigCommands::List => list_servers(),
    }
}

fn add_server(name: &str, url: String, api_key: String) -> Result<()> {
    let mut config = Config::load()?;

    let is_first = config.servers.is_empty();
    config.add_server(name.to_string(), url, api_key);

    // Set as default if it's the first server
    if is_first {
        config.default = Some(name.to_string());
    }

    config.save()?;
    println!("Server '{name}' added successfully.");

    if is_first {
        println!("Set as default server.");
    }

    Ok(())
}

fn remove_server(name: &str) -> Result<()> {
    let mut config = Config::load()?;

    if config.remove_server(name) {
        config.save()?;
        println!("Server '{name}' removed successfully.");
    } else {
        println!("Server '{name}' not found.");
    }

    Ok(())
}

fn set_default(name: &str) -> Result<()> {
    let mut config = Config::load()?;
    config.set_default(name)?;
    config.save()?;
    println!("Default server set to '{name}'.");
    Ok(())
}

fn list_servers() -> Result<()> {
    let config = Config::load()?;

    if config.servers.is_empty() {
        println!("No servers configured.");
        println!("Use 'unraid config add <name> --url <url> --api-key <key>' to add a server.");
        return Ok(());
    }

    println!("Configured servers:");
    println!();

    for (name, server) in &config.servers {
        let default_marker = if config.default.as_deref() == Some(name) {
            " (default)"
        } else {
            ""
        };

        println!("  {name}{default_marker}");
        println!("    URL: {}", server.url);
        println!(
            "    API Key: {}...",
            &server.api_key[..8.min(server.api_key.len())]
        );
        println!();
    }

    Ok(())
}
