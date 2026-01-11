package slim

// EncodeOptions contains options for encoding
type EncodeOptions struct {
	// Maximum nesting depth before returning !DEEP
	MaxDepth int
	// Minimum number of rows to use table format for arrays of objects
	TableThreshold int
	// Pretty print with newlines (for debugging)
	Pretty bool
}

// DefaultEncodeOptions returns default encoding options
func DefaultEncodeOptions() EncodeOptions {
	return EncodeOptions{
		MaxDepth:       15,
		TableThreshold: 1,
		Pretty:         false,
	}
}

// DecodeOptions contains options for decoding
type DecodeOptions struct {
	// Throw error on invalid SLIM input
	Strict bool
}

// DefaultDecodeOptions returns default decoding options
func DefaultDecodeOptions() DecodeOptions {
	return DecodeOptions{
		Strict: false,
	}
}

// ColumnType represents the type of a column in a schema
type ColumnType string

const (
	ColumnTypeNumber   ColumnType = "#"
	ColumnTypeBoolean  ColumnType = "?"
	ColumnTypeString   ColumnType = "$"
	ColumnTypeArray    ColumnType = "@"
	ColumnTypeObject   ColumnType = "~"
	ColumnTypeNullable ColumnType = "!"
)

// ColumnDef represents a column definition in a schema
type ColumnDef struct {
	Name     string
	Type     ColumnType
	Nullable bool
}

// ValidationError represents a schema validation error
type ValidationError struct {
	Path     string
	Message  string
	Expected string
	Actual   string
}

// ValidationResult represents the result of schema validation
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// SLIM special value constants
const (
	SlimNull      = "!null"
	SlimUndefined = "!undef"
	SlimDeep      = "!DEEP"
	SlimTrue      = "?T"
	SlimFalse     = "?F"
	SlimNaN       = "#NaN"
	SlimInf       = "#Inf"
	SlimNegInf    = "#-Inf"
)
