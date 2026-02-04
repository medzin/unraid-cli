//! Build script to fetch GraphQL schema from Apollo Studio.

#![allow(clippy::expect_used, clippy::panic, missing_docs)]

use std::env;
use std::fs;
use std::path::Path;
use std::process::Command;

fn main() {
    let manifest_dir = env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR not set");
    let schema_path = Path::new(&manifest_dir).join("src/graphql/schema.graphql");

    // Run rover to fetch the schema from Apollo GraphQL
    let output = Command::new("rover")
        .args(["graph", "fetch", "Unraid-API@current"])
        .output()
        .expect("Failed to execute rover. Is it installed? (https://www.apollographql.com/docs/rover/getting-started)");

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        panic!(
            "rover graph fetch failed:\n{stderr}\n\
            Make sure APOLLO_KEY is set and you have access to the Unraid-API graph."
        );
    }

    let schema = String::from_utf8(output.stdout).expect("Invalid UTF-8 in schema output");

    // Write the schema to the source directory
    fs::write(&schema_path, schema).expect("Failed to write schema.graphql");

    // Tell Cargo to rerun this script if the schema file is deleted
    println!("cargo::rerun-if-changed={}", schema_path.display());
    // Also rerun if APOLLO_KEY changes (allows refreshing schema)
    println!("cargo::rerun-if-env-changed=APOLLO_KEY");
}
