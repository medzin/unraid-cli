use anyhow::{Context, Result};
use graphql_client::{GraphQLQuery, Response};
use reqwest::Client;

pub struct UnraidClient {
    client: Client,
    url: String,
    api_key: String,
}

impl UnraidClient {
    pub fn new(url: String, api_key: String) -> Result<Self> {
        let client = Client::builder()
            .danger_accept_invalid_certs(true) // Unraid often uses self-signed certs
            .build()
            .context("Failed to create HTTP client")?;

        Ok(Self {
            client,
            url,
            api_key,
        })
    }

    pub async fn execute<Q: GraphQLQuery>(
        &self,
        variables: Q::Variables,
    ) -> Result<Q::ResponseData> {
        let body = Q::build_query(variables);

        let response = self
            .client
            .post(&self.url)
            .header("Content-Type", "application/json")
            .header("x-api-key", &self.api_key)
            .json(&body)
            .send()
            .await
            .context("Failed to send GraphQL request")?;

        let response: Response<Q::ResponseData> = response
            .json()
            .await
            .context("Failed to parse GraphQL response")?;

        if let Some(errors) = response.errors
            && !errors.is_empty()
        {
            let error_messages: Vec<String> = errors.iter().map(|e| e.message.clone()).collect();
            anyhow::bail!("GraphQL errors: {}", error_messages.join(", "));
        }

        response.data.context("No data returned from GraphQL query")
    }
}
