#!/bin/bash

# Database backup script for online shop application
# This script creates backups of the PostgreSQL database and application logs

set -euo pipefail

# Configuration
BACKUP_DIR="/var/backups/online-shop"
LOG_DIR="/var/log/online-shop"
DB_NAME="${DB_NAME:-online_shop}"
DB_USER="${DB_USER:-postgres}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"
mkdir -p "$BACKUP_DIR/database"
mkdir -p "$BACKUP_DIR/logs"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$BACKUP_DIR/backup.log"
}

# Function to cleanup old backups
cleanup_old_backups() {
    log "Cleaning up backups older than $RETENTION_DAYS days..."
    
    # Remove old database backups
    find "$BACKUP_DIR/database" -name "*.sql.gz" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
    
    # Remove old log backups
    find "$BACKUP_DIR/logs" -name "*.tar.gz" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
    
    log "Cleanup completed"
}

# Function to backup database
backup_database() {
    log "Starting database backup..."
    
    local backup_file="$BACKUP_DIR/database/db_backup_$TIMESTAMP.sql"
    local compressed_file="$backup_file.gz"
    
    # Create database dump
    if PGPASSWORD="$PGPASSWORD" pg_dump \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        --verbose \
        --no-password \
        --format=plain \
        --no-owner \
        --no-privileges \
        > "$backup_file"; then
        
        # Compress the backup
        gzip "$backup_file"
        
        # Verify backup integrity
        if gunzip -t "$compressed_file" 2>/dev/null; then
            local size=$(du -h "$compressed_file" | cut -f1)
            log "Database backup completed successfully: $compressed_file ($size)"
        else
            log "ERROR: Database backup verification failed"
            rm -f "$compressed_file"
            return 1
        fi
    else
        log "ERROR: Database backup failed"
        rm -f "$backup_file"
        return 1
    fi
}

# Function to backup application logs
backup_logs() {
    log "Starting log backup..."
    
    if [ -d "$LOG_DIR" ]; then
        local log_backup_file="$BACKUP_DIR/logs/logs_backup_$TIMESTAMP.tar.gz"
        
        # Create compressed archive of logs
        if tar -czf "$log_backup_file" -C "$(dirname "$LOG_DIR")" "$(basename "$LOG_DIR")" 2>/dev/null; then
            local size=$(du -h "$log_backup_file" | cut -f1)
            log "Log backup completed successfully: $log_backup_file ($size)"
        else
            log "ERROR: Log backup failed"
            return 1
        fi
    else
        log "WARNING: Log directory $LOG_DIR not found, skipping log backup"
    fi
}

# Function to rotate application logs
rotate_logs() {
    log "Starting log rotation..."
    
    # Rotate application logs (keep last 5 files)
    for log_file in "$LOG_DIR"/*.log; do
        if [ -f "$log_file" ]; then
            # Create rotated versions
            for i in {4..1}; do
                if [ -f "$log_file.$i" ]; then
                    mv "$log_file.$i" "$log_file.$((i+1))"
                fi
            done
            
            # Move current log to .1
            if [ -s "$log_file" ]; then
                cp "$log_file" "$log_file.1"
                > "$log_file"  # Truncate current log
            fi
        fi
    done
    
    # Remove old rotated logs (older than 5 rotations)
    find "$LOG_DIR" -name "*.log.[6-9]" -delete 2>/dev/null || true
    find "$LOG_DIR" -name "*.log.1[0-9]" -delete 2>/dev/null || true
    
    log "Log rotation completed"
}

# Function to send backup notification
send_notification() {
    local status="$1"
    local message="$2"
    
    # Send notification via webhook or email (if configured)
    if [ -n "${BACKUP_WEBHOOK_URL:-}" ]; then
        curl -s -X POST "$BACKUP_WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "{\"status\":\"$status\",\"message\":\"$message\",\"timestamp\":\"$(date -Iseconds)\"}" \
            >/dev/null 2>&1 || true
    fi
    
    # Log notification
    log "Notification sent: $status - $message"
}

# Function to check disk space
check_disk_space() {
    local backup_dir_usage=$(df "$BACKUP_DIR" | awk 'NR==2 {print $5}' | sed 's/%//')
    local threshold=90
    
    if [ "$backup_dir_usage" -gt "$threshold" ]; then
        log "WARNING: Backup directory is $backup_dir_usage% full (threshold: $threshold%)"
        send_notification "warning" "Backup directory is $backup_dir_usage% full"
    fi
}

# Main execution
main() {
    log "Starting backup process..."
    
    # Check if required environment variables are set
    if [ -z "${PGPASSWORD:-}" ]; then
        log "ERROR: PGPASSWORD environment variable is not set"
        exit 1
    fi
    
    # Check disk space
    check_disk_space
    
    local backup_success=true
    
    # Perform database backup
    if ! backup_database; then
        backup_success=false
    fi
    
    # Perform log backup
    if ! backup_logs; then
        backup_success=false
    fi
    
    # Rotate logs
    rotate_logs
    
    # Cleanup old backups
    cleanup_old_backups
    
    # Send notification
    if [ "$backup_success" = true ]; then
        log "Backup process completed successfully"
        send_notification "success" "Backup completed successfully"
    else
        log "Backup process completed with errors"
        send_notification "error" "Backup completed with errors"
        exit 1
    fi
}

# Handle script interruption
trap 'log "Backup process interrupted"; exit 1' INT TERM

# Run main function
main "$@"