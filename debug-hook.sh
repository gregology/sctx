#!/bin/bash
# Debug wrapper for sctx hook - logs stdin/stdout/stderr to /tmp/sctx-debug.log
LOGFILE="/tmp/sctx-debug.log"
INPUT=$(cat)
echo "=== $(date -Iseconds) ===" >> "$LOGFILE"
echo "STDIN: $INPUT" >> "$LOGFILE"
OUTPUT=$(echo "$INPUT" | /Users/greg/go/bin/sctx hook 2>> "$LOGFILE")
EXIT_CODE=$?
echo "EXIT: $EXIT_CODE" >> "$LOGFILE"
echo "STDOUT: $OUTPUT" >> "$LOGFILE"
echo "" >> "$LOGFILE"
echo "$OUTPUT"
exit $EXIT_CODE
