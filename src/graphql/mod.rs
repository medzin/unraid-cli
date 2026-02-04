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

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "src/graphql/schema.graphql",
    query_path = "src/graphql/queries/containers.graphql",
    response_derives = "Debug, Clone, PartialEq, Eq"
)]
pub struct GetDockerContainers;

pub use get_docker_containers::*;
