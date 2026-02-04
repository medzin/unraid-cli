use anyhow::Result;
use clap::Subcommand;

use crate::client::UnraidClient;
use crate::graphql::{ContainerState, GetDockerContainers};

#[derive(Debug, Subcommand)]
pub enum DockerCommands {
    /// List Docker containers
    #[command(visible_alias = "ls")]
    ListContainers {
        /// Show all containers (default: only running)
        #[arg(short, long)]
        all: bool,
    },
}

pub async fn handle_docker_command(cmd: DockerCommands, client: &UnraidClient) -> Result<()> {
    match cmd {
        DockerCommands::ListContainers { all } => list_containers(client, all).await,
    }
}

async fn list_containers(client: &UnraidClient, show_all: bool) -> Result<()> {
    let response = client
        .execute::<GetDockerContainers>(crate::graphql::get_docker_containers::Variables {})
        .await?;

    let containers = response.docker.containers;

    // Filter containers based on state
    let filtered: Vec<_> = if show_all {
        containers
    } else {
        containers
            .into_iter()
            .filter(|c| c.state == ContainerState::RUNNING)
            .collect()
    };

    if filtered.is_empty() {
        if show_all {
            println!("No containers found.");
        } else {
            println!("No running containers found. Use --all to show all containers.");
        }
        return Ok(());
    }

    // Print header
    println!(
        "{:<30} {:<40} {:<10} {:<20}",
        "NAME", "IMAGE", "STATE", "STATUS"
    );
    println!("{}", "-".repeat(100));

    // Print containers
    for container in filtered {
        let name = container
            .names
            .first()
            .map_or("unnamed", |s| s.trim_start_matches('/'));

        let state = match container.state {
            ContainerState::RUNNING => "running",
            ContainerState::PAUSED => "paused",
            ContainerState::EXITED => "exited",
            ContainerState::Other(ref s) => s.as_str(),
        };

        println!(
            "{:<30} {:<40} {:<10} {:<20}",
            truncate(name, 29),
            truncate(&container.image, 39),
            state,
            truncate(&container.status, 19)
        );
    }

    Ok(())
}

fn truncate(s: &str, max_len: usize) -> String {
    if s.len() <= max_len {
        s.to_string()
    } else {
        format!("{}...", &s[..max_len - 3])
    }
}
