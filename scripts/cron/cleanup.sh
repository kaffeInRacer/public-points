#!/bin/bash

# Cleanup script for online shop application
# This script performs various cleanup tasks to maintain system health

set -euo pipefail

# Configuration
APP_NAME="online-shop"
TEMP_DIR="/tmp/online-shop"
CACHE_DIR="/var/cache/online-shop"
LOG_DIR="/var/log/online-shop"
UPLOAD_DIR="/var/uploads/online-shop"
SESSION_DIR="/var/sessions/online-shop"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
CLEANUP_LOG="$LOG_DIR/cleanup.log"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Retention periods (in days)
TEMP_FILE_RETENTION=1
CACHE_RETENTION=7
OLD_LOG_RETENTION=30
SESSION_RETENTION=1
UPLOAD_RETENTION=90

# Create log directory if it doesn't exist
mkdir -p "$(dirname "$CLEANUP_LOG")"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$CLEANUP_LOG"
}

# Function to cleanup temporary files
cleanup_temp_files() {
    log "Cleaning up temporary files..."
    
    local deleted_count=0
    local freed_space=0
    
    if [ -d "$TEMP_DIR" ]; then
        # Calculate space before cleanup
        local space_before=$(du -sb "$TEMP_DIR" 2>/dev/null | cut -f1 || echo 0)
        
        # Remove files older than retention period
        while IFS= read -r -d '' temp_file; do
            local file_size=$(stat -f%z "$temp_file" 2>/dev/null || stat -c%s "$temp_file" 2>/dev/null || echo 0)
            if rm -f "$temp_file" 2>/dev/null; then
                ((deleted_count++))
                ((freed_space += file_size))
            fi
        done < <(find "$TEMP_DIR" -type f -mtime +$TEMP_FILE_RETENTION -print0 2>/dev/null)
        
        # Remove empty directories
        find "$TEMP_DIR" -type d -empty -delete 2>/dev/null || true
        
        log "Temporary files cleanup: $deleted_count files deleted, $(($freed_space / 1024 / 1024))MB freed"
    else
        log "Temporary directory $TEMP_DIR not found"
    fi
}

# Function to cleanup cache files
cleanup_cache() {
    log "Cleaning up cache files..."
    
    local deleted_count=0
    local freed_space=0
    
    if [ -d "$CACHE_DIR" ]; then
        # Calculate space before cleanup
        local space_before=$(du -sb "$CACHE_DIR" 2>/dev/null | cut -f1 || echo 0)
        
        # Remove cache files older than retention period
        while IFS= read -r -d '' cache_file; do
            local file_size=$(stat -f%z "$cache_file" 2>/dev/null || stat -c%s "$cache_file" 2>/dev/null || echo 0)
            if rm -f "$cache_file" 2>/dev/null; then
                ((deleted_count++))
                ((freed_space += file_size))
            fi
        done < <(find "$CACHE_DIR" -type f -mtime +$CACHE_RETENTION -print0 2>/dev/null)
        
        # Remove empty directories
        find "$CACHE_DIR" -type d -empty -delete 2>/dev/null || true
        
        log "Cache cleanup: $deleted_count files deleted, $(($freed_space / 1024 / 1024))MB freed"
    else
        log "Cache directory $CACHE_DIR not found"
    fi
}

# Function to cleanup old log files
cleanup_old_logs() {
    log "Cleaning up old log files..."
    
    local deleted_count=0
    local freed_space=0
    
    if [ -d "$LOG_DIR" ]; then
        # Remove old log files (excluding current cleanup log)
        while IFS= read -r -d '' log_file; do
            if [[ "$log_file" != "$CLEANUP_LOG" ]]; then
                local file_size=$(stat -f%z "$log_file" 2>/dev/null || stat -c%s "$log_file" 2>/dev/null || echo 0)
                if rm -f "$log_file" 2>/dev/null; then
                    ((deleted_count++))
                    ((freed_space += file_size))
                fi
            fi
        done < <(find "$LOG_DIR" -type f \( -name "*.log" -o -name "*.log.gz" \) -mtime +$OLD_LOG_RETENTION -print0 2>/dev/null)
        
        log "Old logs cleanup: $deleted_count files deleted, $(($freed_space / 1024 / 1024))MB freed"
    else
        log "Log directory $LOG_DIR not found"
    fi
}

# Function to cleanup expired sessions
cleanup_sessions() {
    log "Cleaning up expired sessions..."
    
    local deleted_count=0
    
    if [ -d "$SESSION_DIR" ]; then
        # Remove session files older than retention period
        while IFS= read -r -d '' session_file; do
            if rm -f "$session_file" 2>/dev/null; then
                ((deleted_count++))
            fi
        done < <(find "$SESSION_DIR" -type f -mtime +$SESSION_RETENTION -print0 2>/dev/null)
        
        log "Session cleanup: $deleted_count session files deleted"
    else
        log "Session directory $SESSION_DIR not found"
    fi
}

# Function to cleanup old uploads
cleanup_old_uploads() {
    log "Cleaning up old uploads..."
    
    local deleted_count=0
    local freed_space=0
    
    if [ -d "$UPLOAD_DIR" ]; then
        # Remove upload files older than retention period
        # Be careful with this - only remove files that are confirmed to be temporary or unused
        while IFS= read -r -d '' upload_file; do
            # Only remove files in temp subdirectories
            if [[ "$upload_file" == *"/temp/"* ]] || [[ "$upload_file" == *"/tmp/"* ]]; then
                local file_size=$(stat -f%z "$upload_file" 2>/dev/null || stat -c%s "$upload_file" 2>/dev/null || echo 0)
                if rm -f "$upload_file" 2>/dev/null; then
                    ((deleted_count++))
                    ((freed_space += file_size))
                fi
            fi
        done < <(find "$UPLOAD_DIR" -type f -mtime +$UPLOAD_RETENTION -print0 2>/dev/null)
        
        log "Upload cleanup: $deleted_count files deleted, $(($freed_space / 1024 / 1024))MB freed"
    else
        log "Upload directory $UPLOAD_DIR not found"
    fi
}

# Function to cleanup Redis cache
cleanup_redis_cache() {
    log "Cleaning up Redis cache..."
    
    if command -v redis-cli >/dev/null 2>&1; then
        # Check if Redis is accessible
        if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping >/dev/null 2>&1; then
            # Get memory usage before cleanup
            local memory_before=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" info memory | grep used_memory_human | cut -d: -f2 | tr -d '\r')
            
            # Remove expired keys (Redis should do this automatically, but we can force it)
            local expired_keys=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" eval "
                local keys = redis.call('keys', 'cache:*')
                local expired = 0
                for i=1,#keys do
                    local ttl = redis.call('ttl', keys[i])
                    if ttl == -1 then
                        redis.call('del', keys[i])
                        expired = expired + 1
                    end
                end
                return expired
            " 0 2>/dev/null || echo 0)
            
            # Clean up session keys older than 24 hours
            local session_keys=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" eval "
                local keys = redis.call('keys', 'session:*')
                local deleted = 0
                for i=1,#keys do
                    local ttl = redis.call('ttl', keys[i])
                    if ttl == -1 or ttl > 86400 then
                        redis.call('del', keys[i])
                        deleted = deleted + 1
                    end
                end
                return deleted
            " 0 2>/dev/null || echo 0)
            
            # Get memory usage after cleanup
            local memory_after=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" info memory | grep used_memory_human | cut -d: -f2 | tr -d '\r')
            
            log "Redis cleanup: $expired_keys expired keys, $session_keys old sessions removed"
            log "Redis memory: $memory_before -> $memory_after"
        else
            log "Redis not accessible, skipping Redis cleanup"
        fi
    else
        log "redis-cli not available, skipping Redis cleanup"
    fi
}

# Function to cleanup system resources
cleanup_system_resources() {
    log "Cleaning up system resources..."
    
    # Clear system caches (if running as root)
    if [ "$(id -u)" -eq 0 ]; then
        # Drop caches
        sync
        echo 3 > /proc/sys/vm/drop_caches 2>/dev/null || true
        log "System caches dropped"
    else
        log "Not running as root, skipping system cache cleanup"
    fi
    
    # Clean up core dumps
    local core_dumps=$(find /tmp -name "core.*" -type f -mtime +1 2>/dev/null | wc -l)
    if [ "$core_dumps" -gt 0 ]; then
        find /tmp -name "core.*" -type f -mtime +1 -delete 2>/dev/null || true
        log "Removed $core_dumps core dump files"
    fi
    
    # Clean up old package manager caches (if available)
    if command -v apt-get >/dev/null 2>&1; then
        apt-get clean >/dev/null 2>&1 || true
        log "APT cache cleaned"
    fi
    
    if command -v yum >/dev/null 2>&1; then
        yum clean all >/dev/null 2>&1 || true
        log "YUM cache cleaned"
    fi
}

# Function to check disk space and warn if low
check_disk_space() {
    log "Checking disk space..."
    
    local threshold=90
    local warning_threshold=80
    
    # Check root filesystem
    local usage=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
    
    if [ "$usage" -gt "$threshold" ]; then
        log "CRITICAL: Disk usage is $usage% (threshold: $threshold%)"
        return 1
    elif [ "$usage" -gt "$warning_threshold" ]; then
        log "WARNING: Disk usage is $usage% (warning threshold: $warning_threshold%)"
        return 1
    else
        log "Disk usage OK: $usage%"
        return 0
    fi
}

# Function to generate cleanup report
generate_cleanup_report() {
    local total_freed="$1"
    local errors="$2"
    
    cat > "$LOG_DIR/cleanup_report_$TIMESTAMP.json" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "hostname": "$(hostname)",
    "total_space_freed_mb": $total_freed,
    "errors": $errors,
    "disk_usage_after": "$(df / | awk 'NR==2 {print $5}')",
    "directories_cleaned": [
        "$TEMP_DIR",
        "$CACHE_DIR",
        "$LOG_DIR",
        "$SESSION_DIR",
        "$UPLOAD_DIR"
    ],
    "retention_policies": {
        "temp_files_days": $TEMP_FILE_RETENTION,
        "cache_days": $CACHE_RETENTION,
        "logs_days": $OLD_LOG_RETENTION,
        "sessions_days": $SESSION_RETENTION,
        "uploads_days": $UPLOAD_RETENTION
    }
}
EOF
    
    log "Cleanup report generated: cleanup_report_$TIMESTAMP.json"
}

# Function to send cleanup notification
send_notification() {
    local status="$1"
    local message="$2"
    
    # Send notification via webhook if configured
    if [ -n "${CLEANUP_WEBHOOK:-}" ]; then
        curl -s -X POST "$CLEANUP_WEBHOOK" \
            -H "Content-Type: application/json" \
            -d "{\"status\":\"$status\",\"message\":\"$message\",\"timestamp\":\"$(date -Iseconds)\"}" \
            >/dev/null 2>&1 || true
    fi
}

# Main execution
main() {
    log "Starting cleanup process for $APP_NAME..."
    
    local error_count=0
    local total_freed=0
    
    # Get initial disk usage
    local disk_usage_before=$(df / | awk 'NR==2 {print $5}')
    log "Initial disk usage: $disk_usage_before"
    
    # Run cleanup tasks
    local cleanup_tasks=(
        "cleanup_temp_files"
        "cleanup_cache"
        "cleanup_old_logs"
        "cleanup_sessions"
        "cleanup_old_uploads"
        "cleanup_redis_cache"
        "cleanup_system_resources"
    )
    
    for task in "${cleanup_tasks[@]}"; do
        log "Running $task..."
        if ! eval "$task"; then
            log "ERROR: $task failed"
            ((error_count++))
        fi
    done
    
    # Check final disk space
    local disk_usage_after=$(df / | awk 'NR==2 {print $5}')
    log "Final disk usage: $disk_usage_after"
    
    # Calculate space freed (approximate)
    local usage_before=$(echo "$disk_usage_before" | sed 's/%//')
    local usage_after=$(echo "$disk_usage_after" | sed 's/%//')
    local space_freed=$((usage_before - usage_after))
    
    # Generate report
    generate_cleanup_report "$space_freed" "$error_count"
    
    # Check if disk space is still critical
    if ! check_disk_space; then
        log "WARNING: Disk space is still critical after cleanup"
        send_notification "warning" "Cleanup completed but disk space is still critical"
    fi
    
    # Send final notification
    if [ "$error_count" -eq 0 ]; then
        log "Cleanup completed successfully"
        send_notification "success" "Cleanup completed successfully, freed ${space_freed}% disk space"
    else
        log "Cleanup completed with $error_count errors"
        send_notification "error" "Cleanup completed with $error_count errors"
        exit 1
    fi
}

# Handle script interruption
trap 'log "Cleanup process interrupted"; exit 1' INT TERM

# Run main function
main "$@"