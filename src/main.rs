//! CLI client for interacting with the Unraid API.

mod client;
mod commands;
mod config;
mod graphql;

use anyhow::Result;
use clap::{Parser, Subcommand};

use crate::client::UnraidClient;
use crate::commands::config::{ConfigCommands, handle_config_command};
use crate::commands::docker::{DockerCommands, handle_docker_command};
use crate::config::ResolvedConfig;

#[derive(Parser)]
#[command(name = "unraid")]
#[command(about = "CLI client for Unraid API", long_about = None)]
#[command(version)]
struct Cli {
    /// Server name from config to use
    #[arg(long, global = true, env = "UNRAID_SERVER")]
    server: Option<String>,

    /// Server URL (overrides config and env)
    #[arg(long, global = true, env = "UNRAID_URL")]
    url: Option<String>,

    /// API key (overrides config and env)
    #[arg(long, global = true, env = "UNRAID_API_KEY")]
    api_key: Option<String>,

    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Manage server configurations
    Config {
        #[command(subcommand)]
        command: ConfigCommands,
    },
    /// Docker container management
    Docker {
        #[command(subcommand)]
        command: DockerCommands,
    },
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();

    match cli.command {
        Commands::Config { command } => {
            handle_config_command(command)?;
        }
        Commands::Docker { command } => {
            let resolved = ResolvedConfig::resolve(
                cli.server.as_deref(),
                cli.url.as_deref(),
                cli.api_key.as_deref(),
            )?;

            let client = UnraidClient::new(resolved.url, resolved.api_key)?;
            handle_docker_command(command, &client).await?;
        }
    }

    Ok(())
}
