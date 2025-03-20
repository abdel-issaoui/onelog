#!/bin/bash
# AI-Optimized Project Structure Scanner for Golang
# Purpose: Generate a structured text file of project structure for AI analysis
# Usage: ./scan_project.sh [target_directory] [output_file]

# ===== CONFIGURATION =====
TARGET_DIR=${1:-"."}                          # Default to current directory if no argument is given
OUTPUT_FILE=${2:-"onelog.txt"}              # Allow custom output filename
MAX_FILE_SIZE=1048576                         # 1MB max file size

# ===== INITIALIZATION =====
if [[ ! -d "$TARGET_DIR" ]]; then
    echo "ERROR: Directory '$TARGET_DIR' does not exist."
    exit 1
fi

> "$OUTPUT_FILE"

echo "SCAN_START: Directory='$TARGET_DIR', Output='$OUTPUT_FILE'"

# ===== DIRECTORY STRUCTURE (ONLY *.go FILES) =====
echo "=== Directory Structure (Golang Files) ===" >> "$OUTPUT_FILE"
if command -v tree >/dev/null 2>&1; then
    tree --noreport -P '*.go' "$TARGET_DIR" >> "$OUTPUT_FILE" 2>/dev/null
else
    find "$TARGET_DIR" -type f -name "*.go" | sed "s|^$TARGET_DIR/||" >> "$OUTPUT_FILE"
fi

# ===== FILE PROCESSING FUNCTION =====
process_file() {
    local file="$1"
    local filename=$(basename "$file")
    
    local filesize=$(stat -c%s "$file" 2>/dev/null || echo "0")
    if [[ $filesize -gt $MAX_FILE_SIZE ]]; then
        echo "SKIP_REASON: File too large ($filesize bytes)" >> "$OUTPUT_FILE"
        return
    fi

    echo -e "\n\n=== File: $filename ===" >> "$OUTPUT_FILE"
    echo "FILE_PATH: $(realpath --relative-to="$TARGET_DIR" "$file")" >> "$OUTPUT_FILE"
    echo "FILE_SIZE: $filesize bytes" >> "$OUTPUT_FILE"
    echo "CONTENT_BEGIN" >> "$OUTPUT_FILE"
    if cat "$file" >> "$OUTPUT_FILE" 2>/dev/null; then
        echo "CONTENT_END" >> "$OUTPUT_FILE"
    else
        echo "ERROR: Could not read file" >> "$OUTPUT_FILE"
    fi
}

# ===== FILE CONTENT SCAN (ONLY *.go FILES) =====
echo -e "\n\n=== File Contents ===" >> "$OUTPUT_FILE"
find "$TARGET_DIR" -type f -name "*.go" -print0 | while IFS= read -r -d '' file; do
    process_file "$file"
done

echo "Scan complete. Output saved to $OUTPUT_FILE"
