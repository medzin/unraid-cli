use anyhow::{Result, bail};
use clap::Subcommand;

use crate::client::UnraidClient;
use crate::graphql::get_vms::GetVmsVmsDomains as VmDomain;
use crate::graphql::get_vms::VmState;
use crate::graphql::{ForceStopVm, GetVms, PauseVm, RebootVm, ResetVm, ResumeVm, StartVm, StopVm};

#[derive(Debug, Subcommand)]
pub enum VmCommands {
    /// List virtual machines
    #[command(visible_alias = "ls")]
    List {
        /// Show all VMs (default: only running)
        #[arg(short, long)]
        all: bool,
    },
    /// Start a virtual machine
    Start {
        /// VM name
        name: String,
    },
    /// Stop a virtual machine (graceful shutdown)
    Stop {
        /// VM name
        name: String,
    },
    /// Force stop a virtual machine
    ForceStop {
        /// VM name
        name: String,
    },
    /// Pause a virtual machine
    Pause {
        /// VM name
        name: String,
    },
    /// Resume a paused virtual machine
    Resume {
        /// VM name
        name: String,
    },
    /// Reboot a virtual machine
    Reboot {
        /// VM name
        name: String,
    },
    /// Reset a virtual machine (hard reboot)
    Reset {
        /// VM name
        name: String,
    },
}

pub async fn handle_vm_command(cmd: VmCommands, client: &UnraidClient) -> Result<()> {
    match cmd {
        VmCommands::List { all } => list_vms(client, all).await,
        VmCommands::Start { name } => start_vm(client, &name).await,
        VmCommands::Stop { name } => stop_vm(client, &name).await,
        VmCommands::ForceStop { name } => force_stop_vm(client, &name).await,
        VmCommands::Pause { name } => pause_vm(client, &name).await,
        VmCommands::Resume { name } => resume_vm(client, &name).await,
        VmCommands::Reboot { name } => reboot_vm(client, &name).await,
        VmCommands::Reset { name } => reset_vm(client, &name).await,
    }
}

async fn resolve_vm_id(client: &UnraidClient, name: &str) -> Result<String> {
    let response = client
        .execute::<GetVms>(crate::graphql::get_vms::Variables {})
        .await
        .map_err(map_vms_unavailable)?;

    let domains = response.vms.domains.unwrap_or_default();
    find_vm_id(&domains, name)
}

fn find_vm_id(domains: &[VmDomain], name: &str) -> Result<String> {
    let name_lower = name.to_lowercase();

    for domain in domains {
        if let Some(ref vm_name) = domain.name {
            if vm_name.to_lowercase() == name_lower {
                return Ok(domain.id.clone());
            }
        }
    }

    bail!("VM '{name}' not found. Use 'vm list --all' to see available VMs.");
}

fn map_vms_unavailable(err: anyhow::Error) -> anyhow::Error {
    let msg = err.to_string();
    if msg.contains("VMs are not available") {
        anyhow::anyhow!("VMs are not available on this server. Is the VM service enabled?")
    } else {
        err
    }
}

async fn start_vm(client: &UnraidClient, name: &str) -> Result<()> {
    let id = resolve_vm_id(client, name).await?;

    println!("Starting VM '{name}'...");
    client
        .execute::<StartVm>(crate::graphql::start_vm::Variables { id })
        .await?;

    println!("VM '{name}' start command sent.");
    Ok(())
}

async fn stop_vm(client: &UnraidClient, name: &str) -> Result<()> {
    let id = resolve_vm_id(client, name).await?;

    println!("Stopping VM '{name}'...");
    client
        .execute::<StopVm>(crate::graphql::stop_vm::Variables { id })
        .await?;

    println!("VM '{name}' stop command sent.");
    Ok(())
}

async fn force_stop_vm(client: &UnraidClient, name: &str) -> Result<()> {
    let id = resolve_vm_id(client, name).await?;

    println!("Force stopping VM '{name}'...");
    client
        .execute::<ForceStopVm>(crate::graphql::force_stop_vm::Variables { id })
        .await?;

    println!("VM '{name}' force stop command sent.");
    Ok(())
}

async fn pause_vm(client: &UnraidClient, name: &str) -> Result<()> {
    let id = resolve_vm_id(client, name).await?;

    println!("Pausing VM '{name}'...");
    client
        .execute::<PauseVm>(crate::graphql::pause_vm::Variables { id })
        .await?;

    println!("VM '{name}' pause command sent.");
    Ok(())
}

async fn resume_vm(client: &UnraidClient, name: &str) -> Result<()> {
    let id = resolve_vm_id(client, name).await?;

    println!("Resuming VM '{name}'...");
    client
        .execute::<ResumeVm>(crate::graphql::resume_vm::Variables { id })
        .await?;

    println!("VM '{name}' resume command sent.");
    Ok(())
}

async fn reboot_vm(client: &UnraidClient, name: &str) -> Result<()> {
    let id = resolve_vm_id(client, name).await?;

    println!("Rebooting VM '{name}'...");
    client
        .execute::<RebootVm>(crate::graphql::reboot_vm::Variables { id })
        .await?;

    println!("VM '{name}' reboot command sent.");
    Ok(())
}

async fn reset_vm(client: &UnraidClient, name: &str) -> Result<()> {
    let id = resolve_vm_id(client, name).await?;

    println!("Resetting VM '{name}'...");
    client
        .execute::<ResetVm>(crate::graphql::reset_vm::Variables { id })
        .await?;

    println!("VM '{name}' reset command sent.");
    Ok(())
}

async fn list_vms(client: &UnraidClient, show_all: bool) -> Result<()> {
    let response = client
        .execute::<GetVms>(crate::graphql::get_vms::Variables {})
        .await
        .map_err(map_vms_unavailable)?;

    let domains = response.vms.domains.unwrap_or_default();
    let filtered = filter_by_state(domains, show_all);

    if filtered.is_empty() {
        if show_all {
            println!("No VMs found.");
        } else {
            println!("No running VMs found. Use --all to show all VMs.");
        }
        return Ok(());
    }

    println!("{:<30} {:<12}", "NAME", "STATE");
    println!("{}", "-".repeat(42));

    for vm in filtered {
        let name = vm.name.as_deref().unwrap_or("unnamed");
        let state = format_vm_state(&vm.state);

        println!("{:<30} {:<12}", truncate(name, 29), state);
    }

    Ok(())
}

const fn format_vm_state(state: &VmState) -> &str {
    match state {
        VmState::RUNNING => "running",
        VmState::PAUSED => "paused",
        VmState::SHUTDOWN => "shutdown",
        VmState::SHUTOFF => "shutoff",
        VmState::IDLE => "idle",
        VmState::CRASHED => "crashed",
        VmState::PMSUSPENDED => "suspended",
        VmState::NOSTATE => "no state",
        VmState::Other(_) => "unknown",
    }
}

fn filter_by_state(domains: Vec<VmDomain>, show_all: bool) -> Vec<VmDomain> {
    if show_all {
        domains
    } else {
        domains
            .into_iter()
            .filter(|d| d.state == VmState::RUNNING)
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

    fn sample_vm(id: &str, name: &str, state: VmState) -> VmDomain {
        VmDomain {
            id: id.to_string(),
            name: Some(name.to_string()),
            state,
        }
    }

    fn sample_vms() -> Vec<VmDomain> {
        vec![
            sample_vm("vm-1", "Windows 11", VmState::RUNNING),
            sample_vm("vm-2", "Ubuntu Server", VmState::RUNNING),
            sample_vm("vm-3", "macOS", VmState::SHUTOFF),
            sample_vm("vm-4", "Debian", VmState::PAUSED),
        ]
    }

    // find_vm_id tests

    #[test]
    fn find_vm_id_resolves_matching_names() {
        let vms = sample_vms();
        let cases = [
            ("Windows 11", Ok("vm-1")),
            ("windows 11", Ok("vm-1")),    // case-insensitive
            ("UBUNTU SERVER", Ok("vm-2")), // case-insensitive
            ("nonexistent", Err("not found")),
        ];

        for (input, expected) in cases {
            let result = find_vm_id(&vms, input);
            match expected {
                Ok(id) => assert_eq!(result.unwrap(), id, "find_vm_id({input:?})"),
                Err(msg) => assert!(
                    result.unwrap_err().to_string().contains(msg),
                    "find_vm_id({input:?}) should contain '{msg}'"
                ),
            }
        }
    }

    #[test]
    fn find_vm_id_returns_error_for_empty_list() {
        let err = find_vm_id(&[], "anything").unwrap_err();
        assert!(err.to_string().contains("not found"));
    }

    #[test]
    fn find_vm_id_skips_vms_with_no_name() {
        let vms = vec![VmDomain {
            id: "vm-noname".to_string(),
            name: None,
            state: VmState::RUNNING,
        }];
        let err = find_vm_id(&vms, "something").unwrap_err();
        assert!(err.to_string().contains("not found"));
    }

    // filter_by_state tests

    #[test]
    fn filter_by_state_returns_all_when_show_all_is_true() {
        let vms = sample_vms();
        let filtered = filter_by_state(vms, true);
        assert_eq!(filtered.len(), 4);
    }

    #[test]
    fn filter_by_state_returns_only_running_when_show_all_is_false() {
        let vms = sample_vms();
        let filtered = filter_by_state(vms, false);
        assert_eq!(filtered.len(), 2);
        assert!(filtered.iter().all(|d| d.state == VmState::RUNNING));
    }

    #[test]
    fn filter_by_state_returns_empty_when_no_running_vms() {
        let vms = vec![
            sample_vm("vm-1", "a", VmState::SHUTOFF),
            sample_vm("vm-2", "b", VmState::PAUSED),
        ];
        let filtered = filter_by_state(vms, false);
        assert!(filtered.is_empty());
    }

    // format_vm_state tests

    #[test]
    fn format_vm_state_maps_all_known_states() {
        let cases: Vec<(VmState, &str)> = vec![
            (VmState::RUNNING, "running"),
            (VmState::PAUSED, "paused"),
            (VmState::SHUTDOWN, "shutdown"),
            (VmState::SHUTOFF, "shutoff"),
            (VmState::IDLE, "idle"),
            (VmState::CRASHED, "crashed"),
            (VmState::PMSUSPENDED, "suspended"),
            (VmState::NOSTATE, "no state"),
            (VmState::Other("CUSTOM".to_string()), "unknown"),
        ];

        for (state, expected) in &cases {
            assert_eq!(
                format_vm_state(state),
                *expected,
                "format_vm_state({state:?})"
            );
        }
    }

    // map_vms_unavailable tests

    #[test]
    fn map_vms_unavailable_rewrites_known_error() {
        let cases = [
            (
                "GraphQL errors: Failed to retrieve VM domains: VMs are not available",
                "VM service enabled",
            ),
            ("connection refused", "connection refused"),
        ];

        for (input, expected_substring) in cases {
            let mapped = map_vms_unavailable(anyhow::anyhow!(input.to_string()));
            assert!(
                mapped.to_string().contains(expected_substring),
                "map_vms_unavailable({input:?}) should contain '{expected_substring}'"
            );
        }
    }

    // truncate tests

    #[test]
    fn truncate_handles_various_lengths() {
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
}
