#!/bin/bash
# Hearth File Header & Comment Fixer
# Usage: ./scripts/add-headers.sh [--dry-run]

DRY_RUN=""
if [[ "$1" == "--dry-run" ]]; then
    DRY_RUN="echo"
fi

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
REPO_NAME=$(basename "$REPO_ROOT")

add_header() {
    local file="$1"
    local ext="${file##*.}"
    
    # Skip binary files, generated files, node_modules, vendor
    [[ "$file" =~ (node_modules|vendor|_test\.go|\.git|generated\.go|\.pb\.go) ]] && return
    
    # Determine comment style
    local prefix=""
    case "$ext" in
        go)      prefix="//" ;;
        ts|tsx|js|jsx) prefix="//" ;;
        svelte)  prefix="<!-- -->" ;;
        css|scss) prefix="/*" ;;
        sh)      prefix="#" ;;
        yaml|yml) prefix="#" ;;
        json)    return ;; # Don't add headers to JSON
        md)      prefix="<!-- -->" ;;
        *)       return ;;
    esac
    
    # Check if header already exists
    if grep -q "Repository: $REPO_NAME" "$file" 2>/dev/null; then
        return
    fi
    
    # Create header
    local rel_path="${file#$REPO_ROOT/}"
    local header=""
    
    case "$ext" in
        go)
            header="// Repository: $REPO_NAME
// Path: $rel_path
// Location: $REPO_ROOT/$rel_path

"
            ;;
        ts|tsx|js|jsx)
            header="// Repository: $REPO_NAME
// Path: $rel_path

"
            ;;
        svelte)
            header="<!-- Repository: $REPO_NAME | Path: $rel_path -->

"
            ;;
        css|scss)
            header="/* Repository: $REPO_NAME | Path: $rel_path */

"
            ;;
        sh)
            header="# Repository: $REPO_NAME | Path: $rel_path

"
            ;;
        yaml|yml)
            header="# Repository: $REPO_NAME | Path: $rel_path

"
            ;;
        md)
            header="<!-- Repository: $REPO_NAME | Path: $rel_path -->

"
            ;;
    esac
    
    if [[ -n "$header" ]]; then
        $DRY_RUN sed -i "1s|^|$header|" "$file"
        echo "Added header to $rel_path"
    fi
}

strip_bs_comments() {
    local file="$1"
    local ext="${file##*.}"
    
    # Only process source files
    [[ ! "$ext" =~ ^(go|ts|tsx|js|jsx)$ ]] && return
    
    # Remove verbose synopsis blocks like:
    # // Synopsis:
    # // This function does X, Y, Z
    # //
    # // More details here
    $DRY_RUN sed -i '/^\/\/ Synopsis:/,/^\/\/$/d' "$file"
    $DRY_RUN sed -i '/^\/\/ Summary:/,/^\/\/$/d' "$file"
    $DRY_RUN sed -i '/^\/\/ Description:/,/^\/\/$/d' "$file"
    
    # Remove single-line BS comments that are obvious
    # e.g. "// this is a comment" where the code is obvious
    $DRY_RUN sed -i '/^\/\/ this file/Id' "$file"
    $DRY_RUN sed -i '/^\/\/ todo:/Id' "$file"
    $DRY_RUN sed -i '/^\/\/ fixme:/Id' "$file"
}

echo "Processing files in $REPO_ROOT..."
find "$REPO_ROOT" -type f \( -name "*.go" -o -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx" -o -name "*.svelte" -o -name "*.css" -o -name "*.sh" -o -name "*.yaml" -o -name "*.yml" -o -name "*.md" \) | while read file; do
    add_header "$file"
    # strip_bs_comments "$file"
done

echo "Done!"
