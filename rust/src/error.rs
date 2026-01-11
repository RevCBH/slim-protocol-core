//! Error types for SLIM protocol

use std::fmt;

/// Result type for SLIM operations
pub type Result<T> = std::result::Result<T, SlimError>;

/// Errors that can occur during SLIM encoding/decoding
#[derive(Debug, Clone, PartialEq)]
pub enum SlimError {
    /// Invalid input data
    InvalidInput(String),

    /// Parsing error
    ParseError(String),

    /// Maximum depth exceeded
    MaxDepthExceeded,

    /// Schema validation error
    ValidationError(String),

    /// Unsupported type
    UnsupportedType(String),
}

impl fmt::Display for SlimError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            SlimError::InvalidInput(msg) => write!(f, "Invalid input: {}", msg),
            SlimError::ParseError(msg) => write!(f, "Parse error: {}", msg),
            SlimError::MaxDepthExceeded => write!(f, "Maximum depth exceeded"),
            SlimError::ValidationError(msg) => write!(f, "Validation error: {}", msg),
            SlimError::UnsupportedType(msg) => write!(f, "Unsupported type: {}", msg),
        }
    }
}

impl std::error::Error for SlimError {}
