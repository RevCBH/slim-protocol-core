//! SLIM CLI - A jq-like tool for the SLIM protocol
//!
//! Convert between JSON and SLIM, query data, and more.

use anyhow::{Context, Result};
use clap::{Parser, Subcommand};
use serde_json::Value;
use slim_protocol::{decode, encode, infer_schema, validate_schema};
use slim_protocol::{DecodeOptions, EncodeOptions};
use std::fs;
use std::io::{self, Read, Write};

#[derive(Parser)]
#[command(name = "slim")]
#[command(about = "A jq-like tool for SLIM protocol - convert, query, and manipulate data", long_about = None)]
#[command(version)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Encode JSON to SLIM format
    Encode {
        /// Input file (or stdin if not provided)
        #[arg(short, long)]
        input: Option<String>,

        /// Output file (or stdout if not provided)
        #[arg(short, long)]
        output: Option<String>,

        /// Maximum nesting depth
        #[arg(long, default_value = "15")]
        max_depth: usize,

        /// Minimum rows for table format
        #[arg(long, default_value = "1")]
        table_threshold: usize,

        /// Show token savings
        #[arg(short, long)]
        stats: bool,
    },

    /// Decode SLIM to JSON format
    Decode {
        /// Input file (or stdin if not provided)
        #[arg(short, long)]
        input: Option<String>,

        /// Output file (or stdout if not provided)
        #[arg(short, long)]
        output: Option<String>,

        /// Pretty print JSON output
        #[arg(short, long)]
        pretty: bool,

        /// Strict mode (fail on invalid input)
        #[arg(short, long)]
        strict: bool,
    },

    /// Infer schema from data
    InferSchema {
        /// Input file (or stdin if not provided)
        #[arg(short, long)]
        input: Option<String>,

        /// Input format (json or slim)
        #[arg(short, long, default_value = "json")]
        format: String,
    },

    /// Validate data against a schema
    Validate {
        /// Input file (or stdin if not provided)
        #[arg(short, long)]
        input: Option<String>,

        /// Schema string (e.g., "id#,name$,active?")
        #[arg(short, long)]
        schema: String,

        /// Input format (json or slim)
        #[arg(short, long, default_value = "json")]
        format: String,
    },

    /// Query data (jq-like filtering)
    Query {
        /// Input file (or stdin if not provided)
        #[arg(short, long)]
        input: Option<String>,

        /// JQ-like query expression
        query: String,

        /// Input format (json or slim)
        #[arg(short, long, default_value = "json")]
        format: String,

        /// Output format (json or slim)
        #[arg(short = 'O', long, default_value = "json")]
        output_format: String,
    },

    /// Show statistics about SLIM vs JSON
    Stats {
        /// Input file (or stdin if not provided)
        #[arg(short, long)]
        input: Option<String>,
    },

    /// Format/pretty-print SLIM or JSON
    Format {
        /// Input file (or stdin if not provided)
        #[arg(short, long)]
        input: Option<String>,

        /// Input format (json or slim)
        #[arg(short, long, default_value = "json")]
        format: String,
    },
}

fn main() -> Result<()> {
    let cli = Cli::parse();

    match cli.command {
        Commands::Encode {
            input,
            output,
            max_depth,
            table_threshold,
            stats,
        } => {
            let input_data = read_input(input)?;
            let json_value: Value = serde_json::from_str(&input_data)
                .context("Failed to parse JSON input")?;

            let options = EncodeOptions {
                max_depth,
                table_threshold,
                pretty: false,
            };

            let slim_output = encode(&json_value, options)
                .map_err(|e| anyhow::anyhow!("Encoding failed: {}", e))?;

            write_output(output, &slim_output)?;

            if stats {
                let json_len = input_data.len();
                let slim_len = slim_output.len();
                let savings = if json_len > 0 {
                    ((json_len - slim_len) as f64 / json_len as f64) * 100.0
                } else {
                    0.0
                };
                eprintln!("\nStatistics:");
                eprintln!("  JSON: {} chars", json_len);
                eprintln!("  SLIM: {} chars", slim_len);
                eprintln!("  Savings: {:.1}%", savings);
            }

            Ok(())
        }

        Commands::Decode {
            input,
            output,
            pretty,
            strict,
        } => {
            let input_data = read_input(input)?;
            let options = DecodeOptions { strict };

            let value = decode(&input_data, options)
                .map_err(|e| anyhow::anyhow!("Decoding failed: {}", e))?;

            let json_output = if pretty {
                serde_json::to_string_pretty(&value)?
            } else {
                serde_json::to_string(&value)?
            };

            write_output(output, &json_output)?;
            Ok(())
        }

        Commands::InferSchema { input, format } => {
            let input_data = read_input(input)?;
            let value = if format == "slim" {
                decode(&input_data, DecodeOptions::default())
                    .map_err(|e| anyhow::anyhow!("Decoding failed: {}", e))?
            } else {
                serde_json::from_str(&input_data)
                    .context("Failed to parse JSON input")?
            };

            // Convert to array if needed
            let array = if let Value::Array(arr) = value {
                arr
            } else {
                vec![value]
            };

            let schema = infer_schema(&array);
            println!("{}", schema);
            Ok(())
        }

        Commands::Validate {
            input,
            schema,
            format,
        } => {
            let input_data = read_input(input)?;
            let value = if format == "slim" {
                decode(&input_data, DecodeOptions::default())
                    .map_err(|e| anyhow::anyhow!("Decoding failed: {}", e))?
            } else {
                serde_json::from_str(&input_data)
                    .context("Failed to parse JSON input")?
            };

            let result = validate_schema(&value, &schema);

            if result.valid {
                println!("✓ Valid");
                Ok(())
            } else {
                eprintln!("✗ Validation failed:");
                for error in result.errors {
                    eprintln!("  - {}: {}", error.path, error.message);
                    if let Some(expected) = error.expected {
                        eprintln!("    Expected: {}", expected);
                    }
                    if let Some(actual) = error.actual {
                        eprintln!("    Actual: {}", actual);
                    }
                }
                std::process::exit(1);
            }
        }

        Commands::Query {
            input,
            query,
            format,
            output_format,
        } => {
            let input_data = read_input(input)?;
            let value = if format == "slim" {
                decode(&input_data, DecodeOptions::default())
                    .map_err(|e| anyhow::anyhow!("Decoding failed: {}", e))?
            } else {
                serde_json::from_str(&input_data)
                    .context("Failed to parse JSON input")?
            };

            // Simple query implementation (subset of jq)
            let result = apply_query(&value, &query)?;

            let output = if output_format == "slim" {
                encode(&result, EncodeOptions::default())
                    .map_err(|e| anyhow::anyhow!("Encoding failed: {}", e))?
            } else {
                serde_json::to_string_pretty(&result)?
            };

            println!("{}", output);
            Ok(())
        }

        Commands::Stats { input } => {
            let input_data = read_input(input)?;
            let json_value: Value = serde_json::from_str(&input_data)
                .context("Failed to parse JSON input")?;

            let slim_output = encode(&json_value, EncodeOptions::default())
                .map_err(|e| anyhow::anyhow!("Encoding failed: {}", e))?;

            let json_len = input_data.len();
            let slim_len = slim_output.len();
            let savings = if json_len > 0 {
                ((json_len - slim_len) as f64 / json_len as f64) * 100.0
            } else {
                0.0
            };

            println!("Format Comparison:");
            println!("  JSON: {} chars (~{} tokens)", json_len, json_len / 4);
            println!("  SLIM: {} chars (~{} tokens)", slim_len, slim_len / 4);
            println!("  Savings: {:.1}% ({} chars)", savings, json_len - slim_len);
            println!();
            println!("Data Structure:");
            if let Value::Array(arr) = &json_value {
                println!("  Type: Array");
                println!("  Length: {}", arr.len());
                if let Some(Value::Object(_)) = arr.first() {
                    println!("  Format: Table (array of objects)");
                }
            } else if let Value::Object(obj) = &json_value {
                println!("  Type: Object");
                println!("  Keys: {}", obj.len());
            } else {
                println!("  Type: {:?}", json_value);
            }

            Ok(())
        }

        Commands::Format { input, format } => {
            let input_data = read_input(input)?;

            if format == "slim" {
                // Decode SLIM and re-encode it (normalize)
                let value = decode(&input_data, DecodeOptions::default())
                    .map_err(|e| anyhow::anyhow!("Decoding failed: {}", e))?;
                let slim = encode(&value, EncodeOptions::default())
                    .map_err(|e| anyhow::anyhow!("Encoding failed: {}", e))?;
                println!("{}", slim);
            } else {
                // Pretty print JSON
                let value: Value = serde_json::from_str(&input_data)
                    .context("Failed to parse JSON input")?;
                println!("{}", serde_json::to_string_pretty(&value)?);
            }

            Ok(())
        }
    }
}

fn read_input(path: Option<String>) -> Result<String> {
    match path {
        Some(p) => fs::read_to_string(p).context("Failed to read input file"),
        None => {
            let mut buffer = String::new();
            io::stdin()
                .read_to_string(&mut buffer)
                .context("Failed to read from stdin")?;
            Ok(buffer)
        }
    }
}

fn write_output(path: Option<String>, content: &str) -> Result<()> {
    match path {
        Some(p) => fs::write(p, content).context("Failed to write output file"),
        None => {
            io::stdout()
                .write_all(content.as_bytes())
                .context("Failed to write to stdout")?;
            println!(); // Add newline
            Ok(())
        }
    }
}

fn apply_query(value: &Value, query: &str) -> Result<Value> {
    // Simple query implementation (subset of jq functionality)
    match query {
        "." => Ok(value.clone()),
        ".[]" => {
            if let Value::Array(arr) = value {
                Ok(Value::Array(arr.clone()))
            } else {
                anyhow::bail!("Cannot iterate over non-array")
            }
        }
        q if q.starts_with('.') && q.len() > 1 => {
            // Simple path access like .name or .[0]
            let path = &q[1..];

            // Handle array index like .[0]
            if path.starts_with('[') && path.ends_with(']') {
                let index_str = &path[1..path.len()-1];
                if let Ok(index) = index_str.parse::<usize>() {
                    if let Value::Array(arr) = value {
                        return arr.get(index).cloned()
                            .ok_or_else(|| anyhow::anyhow!("Index out of bounds"));
                    }
                }
            }

            // Handle object key access
            if let Value::Object(obj) = value {
                obj.get(path).cloned()
                    .ok_or_else(|| anyhow::anyhow!("Key '{}' not found", path))
            } else {
                anyhow::bail!("Cannot access property on non-object")
            }
        }
        "length" => {
            match value {
                Value::Array(arr) => Ok(Value::Number(arr.len().into())),
                Value::Object(obj) => Ok(Value::Number(obj.len().into())),
                Value::String(s) => Ok(Value::Number(s.len().into())),
                _ => anyhow::bail!("Cannot get length of this type"),
            }
        }
        "keys" => {
            if let Value::Object(obj) = value {
                let keys: Vec<Value> = obj.keys()
                    .map(|k| Value::String(k.clone()))
                    .collect();
                Ok(Value::Array(keys))
            } else {
                anyhow::bail!("Cannot get keys of non-object")
            }
        }
        "type" => {
            let type_str = match value {
                Value::Null => "null",
                Value::Bool(_) => "boolean",
                Value::Number(_) => "number",
                Value::String(_) => "string",
                Value::Array(_) => "array",
                Value::Object(_) => "object",
            };
            Ok(Value::String(type_str.to_string()))
        }
        _ => anyhow::bail!("Unsupported query: '{}'. Supported: ., .[], .key, .[n], length, keys, type", query),
    }
}
