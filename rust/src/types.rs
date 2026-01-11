//! Core type definitions for SLIM protocol

use serde_json::Value as JsonValue;

/// Alias for serde_json::Value, representing any SLIM-compatible value
pub type Value = JsonValue;

/// Options for encoding data to SLIM format
#[derive(Debug, Clone)]
pub struct EncodeOptions {
    /// Maximum nesting depth before returning !DEEP
    pub max_depth: usize,

    /// Minimum number of rows to use table format for arrays of objects
    pub table_threshold: usize,

    /// Pretty print with newlines (for debugging)
    pub pretty: bool,
}

impl Default for EncodeOptions {
    fn default() -> Self {
        Self {
            max_depth: 15,
            table_threshold: 1,
            pretty: false,
        }
    }
}

/// Options for decoding SLIM strings
#[derive(Debug, Clone)]
pub struct DecodeOptions {
    /// Throw error on invalid SLIM input
    pub strict: bool,
}

impl Default for DecodeOptions {
    fn default() -> Self {
        Self { strict: false }
    }
}

/// Column type in a SLIM table schema
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ColumnType {
    /// Number (#)
    Number,
    /// Boolean (?)
    Boolean,
    /// String ($)
    String,
    /// Array (@)
    Array,
    /// Object (~)
    Object,
    /// Nullable (!)
    Nullable,
}

impl ColumnType {
    /// Convert type marker to ColumnType
    pub fn from_char(c: char) -> Option<Self> {
        match c {
            '#' => Some(ColumnType::Number),
            '?' => Some(ColumnType::Boolean),
            '$' => Some(ColumnType::String),
            '@' => Some(ColumnType::Array),
            '~' => Some(ColumnType::Object),
            '!' => Some(ColumnType::Nullable),
            _ => None,
        }
    }

    /// Convert ColumnType to type marker
    pub fn to_char(self) -> char {
        match self {
            ColumnType::Number => '#',
            ColumnType::Boolean => '?',
            ColumnType::String => '$',
            ColumnType::Array => '@',
            ColumnType::Object => '~',
            ColumnType::Nullable => '!',
        }
    }

    /// Convert to human-readable string
    pub fn to_string_repr(self) -> &'static str {
        match self {
            ColumnType::Number => "number",
            ColumnType::Boolean => "boolean",
            ColumnType::String => "string",
            ColumnType::Array => "array",
            ColumnType::Object => "object",
            ColumnType::Nullable => "any (nullable)",
        }
    }
}

/// Column definition in a SLIM table schema
#[derive(Debug, Clone, PartialEq)]
pub struct ColumnDef {
    /// Column name
    pub name: String,
    /// Column type
    pub col_type: ColumnType,
    /// Whether the column is nullable
    pub nullable: bool,
}

/// Result of schema validation
#[derive(Debug, Clone, PartialEq)]
pub struct ValidationResult {
    /// Whether validation passed
    pub valid: bool,
    /// Validation errors if any
    pub errors: Vec<ValidationError>,
}

/// Schema validation error
#[derive(Debug, Clone, PartialEq)]
pub struct ValidationError {
    /// Path to the invalid field
    pub path: String,
    /// Error message
    pub message: String,
    /// Expected type
    pub expected: Option<String>,
    /// Actual type
    pub actual: Option<String>,
}

// SLIM special value constants
pub const SLIM_NULL: &str = "!null";
pub const SLIM_UNDEFINED: &str = "!undef";
pub const SLIM_DEEP: &str = "!DEEP";
pub const SLIM_TRUE: &str = "?T";
pub const SLIM_FALSE: &str = "?F";
pub const SLIM_NAN: &str = "#NaN";
pub const SLIM_INF: &str = "#Inf";
pub const SLIM_NEG_INF: &str = "#-Inf";
