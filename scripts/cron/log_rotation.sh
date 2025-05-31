#!/bin/bash

# Log rotation script for online shop application
# This script rotates application logs and manages log retention

set -euo pipefail

# Configuration
APP_NAME="online-shop"
LOG_DIR="/var/log/online-shop"
ARCHIVE_DIR="/var/log/online-shop/archive"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
MAX_LOG_SIZE="${MAX_LOG_SIZE:-100M}"
COMPRESS_LOGS="${COMPRESS_LOGS:-true}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Create directories if they don't exist
mkdir -p "$LOG_DIR"
mkdir -p "$ARCHIVE_DIR"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_DIR/log_rotation.log"
}

# Function to get file size in bytes
get_file_size() {
    local file="$1"
    if [ -f "$file" ]; then
        stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo 0
    else
        echo 0
    fi
}

# Function to convert size string to bytes
size_to_bytes() {
    local size="$1"
    local number=$(echo "$size" | sed 's/[^0-9]*//g')
    local unit=$(echo "$size" | sed 's/[0-9]*//g' | tr '[:lower:]' '[:upper:]')
    
    case "$unit" in
        "K"|"KB") echo $((number * 1024)) ;;
        "M"|"MB") echo $((number * 1024 * 1024)) ;;
        "G"|"GB") echo $((number * 1024 * 1024 * 1024)) ;;
        *) echo "$number" ;;
    esac
}

# Function to rotate a single log file
rotate_log_file() {
    local log_file="$1"
    local max_size_bytes=$(size_to_bytes "$MAX_LOG_SIZE")
    local file_size=$(get_file_size "$log_file")
    
    if [ ! -f "$log_file" ]; then
        return 0
    fi
    
    log "Processing log file: $log_file"
    
    # Check if rotation is needed
    local needs_rotation=false
    
    # Size-based rotation
    if [ "$file_size" -gt "$max_size_bytes" ]; then
        log "File size ($file_size bytes) exceeds limit ($max_size_bytes bytes)"
        needs_rotation=true
    fi
    
    # Time-based rotation (daily)
    local file_date=$(date -r "$log_file" +%Y%m%d 2>/dev/null || echo "")
    local current_date=$(date +%Y%m%d)
    if [ -n "$file_date" ] && [ "$file_date" != "$current_date" ]; then
        log "File is from previous day ($file_date)"
        needs_rotation=true
    fi
    
    if [ "$needs_rotation" = true ]; then
        rotate_file "$log_file"
    else
        log "No rotation needed for $log_file"
    fi
}

# Function to perform the actual rotation
rotate_file() {
    local log_file="$1"
    local base_name=$(basename "$log_file" .log)
    local rotated_file="$ARCHIVE_DIR/${base_name}_${TIMESTAMP}.log"
    
    log "Rotating $log_file to $rotated_file"
    
    # Copy the log file to archive
    if cp "$log_file" "$rotated_file"; then
        # Truncate the original log file
        > "$log_file"
        
        # Compress the rotated file if enabled
        if [ "$COMPRESS_LOGS" = true ]; then
            log "Compressing $rotated_file"
            if gzip "$rotated_file"; then
                log "Compressed to ${rotated_file}.gz"
            else
                log "WARNING: Failed to compress $rotated_file"
            fi
        fi
        
        log "Successfully rotated $log_file"
    else
        log "ERROR: Failed to rotate $log_file"
        return 1
    fi
}

# Function to clean up old log files
cleanup_old_logs() {
    log "Cleaning up logs older than $RETENTION_DAYS days..."
    
    local deleted_count=0
    
    # Clean up archived logs
    if [ -d "$ARCHIVE_DIR" ]; then
        while IFS= read -r -d '' old_file; do
            log "Deleting old log file: $old_file"
            rm -f "$old_file"
            ((deleted_count++))
        done < <(find "$ARCHIVE_DIR" -type f \( -name "*.log" -o -name "*.log.gz" \) -mtime +$RETENTION_DAYS -print0 2>/dev/null)
    fi
    
    # Clean up old rotated logs in main directory
    while IFS= read -r -d '' old_file; do
        log "Deleting old rotated log: $old_file"
        rm -f "$old_file"
        ((deleted_count++))
    done < <(find "$LOG_DIR" -type f -name "*.log.[0-9]*" -mtime +$RETENTION_DAYS -print0 2>/dev/null)
    
    log "Cleanup completed: $deleted_count files deleted"
}

# Function to manage numbered log rotations (like logrotate)
manage_numbered_rotations() {
    local log_file="$1"
    local max_rotations="${2:-5}"
    
    if [ ! -f "$log_file" ]; then
        return 0
    fi
    
    local base_file="${log_file%.*}"
    local extension="${log_file##*.}"
    
    # Shift existing rotations
    for i in $(seq $((max_rotations - 1)) -1 1); do
        local current_file="$base_file.$i.$extension"
        local next_file="$base_file.$((i + 1)).$extension"
        
        if [ -f "$current_file" ]; then
            if [ $i -eq $((max_rotations - 1)) ]; then
                # Delete the oldest rotation
                rm -f "$current_file"
                log "Deleted oldest rotation: $current_file"
            else
                # Move to next number
                mv "$current_file" "$next_file"
                log "Moved $current_file to $next_file"
            fi
        fi
    done
    
    # Move current log to .1
    if [ -s "$log_file" ]; then
        local first_rotation="$base_file.1.$extension"
        cp "$log_file" "$first_rotation"
        > "$log_file"  # Truncate current log
        
        # Compress if enabled
        if [ "$COMPRESS_LOGS" = true ]; then
            gzip "$first_rotation"
            log "Created and compressed rotation: ${first_rotation}.gz"
        else
            log "Created rotation: $first_rotation"
        fi
    fi
}

# Function to check log directory permissions
check_permissions() {
    log "Checking log directory permissions..."
    
    if [ ! -w "$LOG_DIR" ]; then
        log "ERROR: No write permission to log directory: $LOG_DIR"
        return 1
    fi
    
    if [ ! -w "$ARCHIVE_DIR" ]; then
        log "ERROR: No write permission to archive directory: $ARCHIVE_DIR"
        return 1
    fi
    
    log "Permissions OK"
    return 0
}

# Function to generate rotation report
generate_rotation_report() {
    local rotated_files="$1"
    local deleted_files="$2"
    local errors="$3"
    
    cat > "$LOG_DIR/rotation_report_$TIMESTAMP.json" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "log_directory": "$LOG_DIR",
    "archive_directory": "$ARCHIVE_DIR",
    "retention_days": $RETENTION_DAYS,
    "max_log_size": "$MAX_LOG_SIZE",
    "compress_logs": $COMPRESS_LOGS,
    "rotated_files": $rotated_files,
    "deleted_files": $deleted_files,
    "errors": $errors,
    "disk_usage": {
        "log_dir_mb": $(du -sm "$LOG_DIR" 2>/dev/null | cut -f1 || echo 0),
        "archive_dir_mb": $(du -sm "$ARCHIVE_DIR" 2>/dev/null | cut -f1 || echo 0)
    }
}
EOF
    
    log "Rotation report generated: rotation_report_$TIMESTAMP.json"
}

# Function to send rotation notification
send_notification() {
    local status="$1"
    local message="$2"
    
    # Send notification via webhook if configured
    if [ -n "${LOG_ROTATION_WEBHOOK:-}" ]; then
        curl -s -X POST "$LOG_ROTATION_WEBHOOK" \
            -H "Content-Type: application/json" \
            -d "{\"status\":\"$status\",\"message\":\"$message\",\"timestamp\":\"$(date -Iseconds)\"}" \
            >/dev/null 2>&1 || true
    fi
}

# Main execution
main() {
    log "Starting log rotation for $APP_NAME..."
    
    # Check permissions
    if ! check_permissions; then
        log "Permission check failed, exiting"
        exit 1
    fi
    
    local rotated_count=0
    local error_count=0
    
    # Find and rotate log files
    while IFS= read -r -d '' log_file; do
        if rotate_log_file "$log_file"; then
            ((rotated_count++))
        else
            ((error_count++))
        fi
    done < <(find "$LOG_DIR" -maxdepth 1 -name "*.log" -type f -print0 2>/dev/null)
    
    # Handle numbered rotations for specific files
    local numbered_rotation_files=(
        "$LOG_DIR/application.log"
        "$LOG_DIR/error.log"
        "$LOG_DIR/access.log"
        "$LOG_DIR/worker.log"
    )
    
    for file in "${numbered_rotation_files[@]}"; do
        if [ -f "$file" ]; then
            manage_numbered_rotations "$file" 5
        fi
    done
    
    # Clean up old logs
    local deleted_count=0
    cleanup_old_logs
    
    # Generate report
    generate_rotation_report "$rotated_count" "$deleted_count" "$error_count"
    
    # Send notification
    if [ "$error_count" -eq 0 ]; then
        log "Log rotation completed successfully: $rotated_count files rotated"
        send_notification "success" "Log rotation completed: $rotated_count files rotated"
    else
        log "Log rotation completed with errors: $error_count errors"
        send_notification "error" "Log rotation completed with $error_count errors"
        exit 1
    fi
}

# Handle script interruption
trap 'log "Log rotation interrupted"; exit 1' INT TERM

# Run main function
main "$@"