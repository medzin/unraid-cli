use graphql_client::GraphQLQuery;

// Custom scalar types from the Unraid GraphQL schema
// These are required by the GraphQL schema even if not all are used in current queries
// Names must match the GraphQL schema exactly
#[allow(dead_code, clippy::upper_case_acronyms)]
pub type PrefixedID = String;
#[allow(dead_code)]
pub type Port = i64;
#[allow(dead_code)]
pub type DateTime = String;
#[allow(dead_code)]
pub type BigInt = i64;
#[allow(dead_code, clippy::upper_case_acronyms)]
pub type JSON = serde_json::Value;
#[allow(dead_code, clippy::upper_case_acronyms)]
pub type URL = String;

// Docker queries & mutations

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "src/graphql/schema.graphql",
    query_path = "src/graphql/queries/containers.graphql",
    response_derives = "Debug, Clone, PartialEq, Eq"
)]
pub struct GetDockerContainers;

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "src/graphql/schema.graphql",
    query_path = "src/graphql/mutations/docker/start_container.graphql",
    response_derives = "Debug, Clone"
)]
pub struct StartDockerContainer;

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "src/graphql/schema.graphql",
    query_path = "src/graphql/mutations/docker/stop_container.graphql",
    response_derives = "Debug, Clone"
)]
pub struct StopDockerContainer;

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "src/graphql/schema.graphql",
    query_path = "src/graphql/mutations/docker/update_container.graphql",
    response_derives = "Debug, Clone"
)]
pub struct UpdateDockerContainer;

// VM queries & mutations

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "src/graphql/schema.graphql",
    query_path = "src/graphql/queries/vms.graphql",
    response_derives = "Debug, Clone, PartialEq, Eq"
)]
pub struct GetVms;

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "src/graphql/schema.graphql",
    query_path = "src/graphql/mutations/vm/start_vm.graphql",
    response_derives = "Debug, Clone"
)]
pub struct StartVm;

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "src/graphql/schema.graphql",
    query_path = "src/graphql/mutations/vm/stop_vm.graphql",
    response_derives = "Debug, Clone"
)]
pub struct StopVm;

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "src/graphql/schema.graphql",
    query_path = "src/graphql/mutations/vm/pause_vm.graphql",
    response_derives = "Debug, Clone"
)]
pub struct PauseVm;

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "src/graphql/schema.graphql",
    query_path = "src/graphql/mutations/vm/resume_vm.graphql",
    response_derives = "Debug, Clone"
)]
pub struct ResumeVm;

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "src/graphql/schema.graphql",
    query_path = "src/graphql/mutations/vm/force_stop_vm.graphql",
    response_derives = "Debug, Clone"
)]
pub struct ForceStopVm;

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "src/graphql/schema.graphql",
    query_path = "src/graphql/mutations/vm/reboot_vm.graphql",
    response_derives = "Debug, Clone"
)]
pub struct RebootVm;

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "src/graphql/schema.graphql",
    query_path = "src/graphql/mutations/vm/reset_vm.graphql",
    response_derives = "Debug, Clone"
)]
pub struct ResetVm;

pub use get_docker_containers::*;
