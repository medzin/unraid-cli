use anyhow::{Result, bail};
use clap::Subcommand;

use crate::client::UnraidClient;
use crate::graphql::get_docker_containers::GetDockerContainersDockerContainers as Container;
use crate::graphql::{
    ContainerState, GetDockerContainers, StartDockerContainer, StopDockerContainer,
    UpdateDockerContainer,
};

#[derive(Debug, Subcommand)]
pub enum DockerCommands {
    /// List Docker containers
    #[command(visible_alias = "ls")]
    ListContainers {
        /// Show all containers (default: only running)
        #[arg(short, long)]
        all: bool,
    },
    /// Start a Docker container
    Start {
        /// Container name
        name: String,
    },
    /// Stop a Docker container
    Stop {
        /// Container name
        name: String,
    },
    /// Restart a Docker container (stop then start)
    Restart {
        /// Container name
        name: String,
    },
    /// Update a Docker container to the latest image
    Update {
        /// Container name
        name: String,
    },
}

pub async fn handle_docker_command(cmd: DockerCommands, client: &UnraidClient) -> Result<()> {
    match cmd {
        DockerCommands::ListContainers { all } => list_containers(client, all).await,
        DockerCommands::Start { name } => start_container(client, &name).await,
        DockerCommands::Stop { name } => stop_container(client, &name).await,
        DockerCommands::Restart { name } => restart_container(client, &name).await,
        DockerCommands::Update { name } => update_container(client, &name).await,
    }
}

/// Resolve a container name to its prefixed ID by querying all containers.
async fn resolve_container_id(client: &UnraidClient, name: &str) -> Result<String> {
    let response = client
        .execute::<GetDockerContainers>(crate::graphql::get_docker_containers::Variables {})
        .await?;

    find_container_id(&response.docker.containers, name)
}

fn find_container_id(containers: &[Container], name: &str) -> Result<String> {
    let name_lower = name.to_lowercase();

    for container in containers {
        for container_name in &container.names {
            let clean = container_name.trim_start_matches('/').to_lowercase();
            if clean == name_lower {
                return Ok(container.id.clone());
            }
        }
    }

    bail!(
        "Container '{name}' not found. Use 'docker list-containers --all' to see available containers."
    );
}

async fn start_container(client: &UnraidClient, name: &str) -> Result<()> {
    let id = resolve_container_id(client, name).await?;

    println!("Starting container '{name}'...");
    let response = client
        .execute::<StartDockerContainer>(crate::graphql::start_docker_container::Variables { id })
        .await?;

    let container = response.docker.start;
    let state = format!("{:?}", container.state).to_lowercase();
    println!("Container '{name}' is now {state}.");

    Ok(())
}

async fn stop_container(client: &UnraidClient, name: &str) -> Result<()> {
    let id = resolve_container_id(client, name).await?;

    println!("Stopping container '{name}'...");
    let response = client
        .execute::<StopDockerContainer>(crate::graphql::stop_docker_container::Variables { id })
        .await?;

    let container = response.docker.stop;
    let state = format!("{:?}", container.state).to_lowercase();
    println!("Container '{name}' is now {state}.");

    Ok(())
}

async fn update_container(client: &UnraidClient, name: &str) -> Result<()> {
    let id = resolve_container_id(client, name).await?;

    println!("Updating container '{name}'...");
    let response = client
        .execute::<UpdateDockerContainer>(crate::graphql::update_docker_container::Variables { id })
        .await?;

    let container = response.docker.update_container;
    let state = format!("{:?}", container.state).to_lowercase();
    println!("Container '{name}' updated successfully (state: {state}).");

    Ok(())
}

async fn restart_container(client: &UnraidClient, name: &str) -> Result<()> {
    let id = resolve_container_id(client, name).await?;

    println!("Restarting container '{name}'...");
    client
        .execute::<StopDockerContainer>(crate::graphql::stop_docker_container::Variables {
            id: id.clone(),
        })
        .await?;

    let response = client
        .execute::<StartDockerContainer>(crate::graphql::start_docker_container::Variables { id })
        .await?;

    let container = response.docker.start;
    let state = format!("{:?}", container.state).to_lowercase();
    println!("Container '{name}' is now {state}.");

    Ok(())
}

async fn list_containers(client: &UnraidClient, show_all: bool) -> Result<()> {
    let response = client
        .execute::<GetDockerContainers>(crate::graphql::get_docker_containers::Variables {})
        .await?;

    let containers = response.docker.containers;
    let filtered = filter_by_state(containers, show_all);

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

fn filter_by_state(containers: Vec<Container>, show_all: bool) -> Vec<Container> {
    if show_all {
        containers
    } else {
        containers
            .into_iter()
            .filter(|c| c.state == ContainerState::RUNNING)
            .collect()
    }
}

fn truncate(s: &str, max_len: usize) -> String {
    if s.len() <= max_len {
        s.to_string()
    } else {
        format!("{}...", &s[..max_len - 3])
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    fn sample_container(id: &str, name: &str, state: ContainerState) -> Container {
        Container {
            id: id.to_string(),
            names: vec![name.to_string()],
            image: "some-image:latest".to_string(),
            state,
            status: "Up 2 hours".to_string(),
            ports: vec![],
        }
    }

    fn sample_containers() -> Vec<Container> {
        vec![
            sample_container("id-1", "plex", ContainerState::RUNNING),
            sample_container("id-2", "sonarr", ContainerState::RUNNING),
            sample_container("id-3", "radarr", ContainerState::EXITED),
            sample_container("id-4", "nginx", ContainerState::PAUSED),
        ]
    }

    #[test]
    fn truncate_truncates_correctly() {
        let cases = [
            ("hello", 10, "hello"),
            ("hello", 5, "hello"),
            ("hello world", 8, "hello..."),
            ("", 5, ""),
        ];

        for (input, max_len, expected) in cases {
            assert_eq!(
                truncate(input, max_len),
                expected,
                "truncate({input:?}, {max_len})"
            );
        }
    }

    // find_container_id tests

    #[test]
    fn find_container_id_returns_id_for_matching_name() {
        let containers = sample_containers();
        let result = find_container_id(&containers, "plex").unwrap();
        assert_eq!(result, "id-1");
    }

    #[test]
    fn find_container_id_is_case_insensitive() {
        let containers = sample_containers();
        let result = find_container_id(&containers, "PLEX").unwrap();
        assert_eq!(result, "id-1");
    }

    #[test]
    fn find_container_id_strips_leading_slash() {
        let containers = vec![Container {
            id: "id-slash".to_string(),
            names: vec!["/mycontainer".to_string()],
            image: "img".to_string(),
            state: ContainerState::RUNNING,
            status: "Up".to_string(),
            ports: vec![],
        }];

        let result = find_container_id(&containers, "mycontainer").unwrap();
        assert_eq!(result, "id-slash");
    }

    #[test]
    fn find_container_id_returns_error_for_unknown_name() {
        let containers = sample_containers();
        let err = find_container_id(&containers, "nonexistent").unwrap_err();
        assert!(err.to_string().contains("not found"));
    }

    #[test]
    fn find_container_id_returns_error_for_empty_list() {
        let err = find_container_id(&[], "anything").unwrap_err();
        assert!(err.to_string().contains("not found"));
    }

    #[test]
    fn find_container_id_matches_second_name_in_list() {
        let containers = vec![Container {
            id: "id-multi".to_string(),
            names: vec!["primary".to_string(), "alias".to_string()],
            image: "img".to_string(),
            state: ContainerState::RUNNING,
            status: "Up".to_string(),
            ports: vec![],
        }];

        let result = find_container_id(&containers, "alias").unwrap();
        assert_eq!(result, "id-multi");
    }

    // filter_by_state tests

    #[test]
    fn filter_by_state_returns_all_containers_when_show_all_is_true() {
        let containers = sample_containers();
        let filtered = filter_by_state(containers, true);
        assert_eq!(filtered.len(), 4);
    }

    #[test]
    fn filter_by_state_returns_only_running_when_show_all_is_false() {
        let containers = sample_containers();
        let filtered = filter_by_state(containers, false);
        assert_eq!(filtered.len(), 2);
        assert!(filtered.iter().all(|c| c.state == ContainerState::RUNNING));
    }

    #[test]
    fn filter_by_state_returns_empty_when_no_running_containers() {
        let containers = vec![
            sample_container("id-1", "a", ContainerState::EXITED),
            sample_container("id-2", "b", ContainerState::PAUSED),
        ];
        let filtered = filter_by_state(containers, false);
        assert!(filtered.is_empty());
    }
}
