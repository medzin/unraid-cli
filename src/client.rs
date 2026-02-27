use anyhow::{Context, Result};
use graphql_client::{GraphQLQuery, Response};
use reqwest::Client;
use std::time::Duration;

pub struct UnraidClient {
    client: Client,
    url: String,
    api_key: String,
}

impl UnraidClient {
    pub fn new(url: String, api_key: String, timeout_secs: u64) -> Result<Self> {
        let client = Client::builder()
            .danger_accept_invalid_certs(true) // Unraid often uses self-signed certs
            .timeout(Duration::from_secs(timeout_secs))
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

        Self::process_response(response)
    }

    fn process_response<T>(response: Response<T>) -> Result<T> {
        if let Some(errors) = response.errors
            && !errors.is_empty()
        {
            let error_messages: Vec<String> = errors.iter().map(|e| e.message.clone()).collect();
            anyhow::bail!("GraphQL errors: {}", error_messages.join(", "));
        }

        response.data.context("No data returned from GraphQL query")
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use graphql_client::Error;

    #[test]
    fn new_creates_client_successfully() {
        let result = UnraidClient::new(
            "https://192.168.1.100/graphql".to_string(),
            "test-api-key".to_string(),
            5,
        );
        assert!(result.is_ok());
    }

    #[test]
    fn process_response_returns_data_when_no_errors() {
        let cases: Vec<(Response<String>, &str)> = vec![
            (
                Response {
                    data: Some("hello".to_string()),
                    errors: None,
                    extensions: None,
                },
                "errors=None",
            ),
            (
                Response {
                    data: Some("hello".to_string()),
                    errors: Some(vec![]),
                    extensions: None,
                },
                "errors=Some([])",
            ),
        ];

        for (response, label) in cases {
            let result = UnraidClient::process_response(response).unwrap();
            assert_eq!(result, "hello", "{label}");
        }
    }

    #[test]
    fn process_response_returns_error_when_data_is_none() {
        let response: Response<String> = Response {
            data: None,
            errors: None,
            extensions: None,
        };

        let err = UnraidClient::process_response(response).unwrap_err();
        assert!(err.to_string().contains("No data returned"));
    }

    #[test]
    fn process_response_returns_error_on_graphql_errors() {
        let response: Response<String> = Response {
            data: Some("ignored".to_string()),
            errors: Some(vec![Error {
                message: "field not found".to_string(),
                locations: None,
                path: None,
                extensions: None,
            }]),
            extensions: None,
        };

        let err = UnraidClient::process_response(response).unwrap_err();
        assert!(err.to_string().contains("GraphQL errors"));
        assert!(err.to_string().contains("field not found"));
    }

    #[test]
    fn process_response_joins_multiple_error_messages() {
        let response: Response<String> = Response {
            data: None,
            errors: Some(vec![
                Error {
                    message: "error one".to_string(),
                    locations: None,
                    path: None,
                    extensions: None,
                },
                Error {
                    message: "error two".to_string(),
                    locations: None,
                    path: None,
                    extensions: None,
                },
            ]),
            extensions: None,
        };

        let err = UnraidClient::process_response(response).unwrap_err();
        assert!(err.to_string().contains("error one"));
        assert!(err.to_string().contains("error two"));
    }
}
