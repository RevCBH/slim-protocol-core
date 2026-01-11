#!/bin/bash
# SLIM CLI Demo Script
# Demonstrates various CLI commands

set -e

echo "=== SLIM CLI Demo ==="
echo

# Build the CLI if needed
if [ ! -f "./target/release/slim" ]; then
    echo "Building CLI tool..."
    cargo build --release --bin slim
    echo
fi

SLIM="./target/release/slim"

echo "1. Basic encoding (JSON to SLIM)"
echo "Input:"
cat << 'EOF' | tee /tmp/demo.json
[
  {"id": 1, "name": "Mario", "score": 95.5, "active": true},
  {"id": 2, "name": "Luigi", "score": 87.2, "active": false},
  {"id": 3, "name": "Peach", "score": 92.8, "active": true}
]
EOF

echo
echo "Output (with stats):"
cat /tmp/demo.json | $SLIM encode --stats
echo

echo "2. Decoding (SLIM to JSON)"
echo "Input:"
cat << 'EOF' | tee /tmp/demo.slim
|2|id#,name$,active?|
1,Mario,T
2,Luigi,F
EOF

echo
echo "Output:"
cat /tmp/demo.slim | $SLIM decode --pretty
echo

echo "3. Query operations (jq-like)"
echo "Get first user:"
cat /tmp/demo.json | $SLIM query '.[0]'
echo

echo "Get array length:"
cat /tmp/demo.json | $SLIM query 'length'
echo

echo "Get type:"
cat /tmp/demo.json | $SLIM query 'type'
echo

echo "Query nested data:"
echo '{"users":[{"name":"Mario"}],"count":1}' | $SLIM query '.users'
echo

echo "4. Schema operations"
echo "Infer schema:"
cat /tmp/demo.json | $SLIM infer-schema
echo

echo "Validate (valid):"
cat /tmp/demo.json | $SLIM validate --schema 'id#,name$,score#,active?'
echo

echo "Validate (invalid - missing field):"
echo '[{"id": 1}]' | $SLIM validate --schema 'id#,name$' || echo "(Expected validation error)"
echo

echo "5. Statistics"
cat /tmp/demo.json | $SLIM stats
echo

echo "6. Format operations"
echo "Pretty-print JSON:"
echo '{"a":1,"b":2}' | $SLIM format
echo

echo "Normalize SLIM:"
echo '|1|id#,name$|1,Mario' | $SLIM format --format slim
echo

echo "7. Piping operations (JSON → SLIM → JSON)"
echo "Original:"
cat /tmp/demo.json
echo
echo "Via SLIM (encode → decode):"
cat /tmp/demo.json | $SLIM encode | $SLIM decode --pretty
echo

echo "8. File I/O"
echo "Write to file:"
cat /tmp/demo.json | $SLIM encode --output /tmp/output.slim
echo "Encoded to: /tmp/output.slim"
cat /tmp/output.slim
echo

echo "Read from file:"
$SLIM decode --input /tmp/output.slim --pretty
echo

# Cleanup
rm -f /tmp/demo.json /tmp/demo.slim /tmp/output.slim

echo "=== Demo Complete ==="
