#!/bin/bash

# Online Shop Backup Script
# This script performs database and log file backups

set -e  # Exit on any error

# Configuration
BACKUP_DIR="/var/backups/online-shop"
LOG_DIR="/var/log/online-shop"
DB_NAME="online_shop"
DB_USER="postgres"
DB_HOST="localhost"
DB_PORT="5432"
RETENTION_DAYS=30
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR/database"
mkdir -p "$BACKUP_DIR/logs"
mkdir -p "$BACKUP_DIR/temp"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$BACKUP_DIR/backup.log"
}

# Error handling
error_exit() {
    log "ERROR: $1"
    exit 1
}

# Check if required tools are available
check_dependencies() {
    log "Checking dependencies..."
    
    if ! command -v pg_dump &> /dev/null; then
        error_exit "pg_dump is not installed or not in PATH"
    fi
    
    if ! command -v gzip &> /dev/null; then
        error_exit "gzip is not installed or not in PATH"
    fi
    
    if ! command -v tar &> /dev/null; then
        error_exit "tar is not installed or not in PATH"
    fi
    
    log "All dependencies are available"
}

# Database backup function
backup_database() {
    log "Starting database backup..."
    
    local backup_file="$BACKUP_DIR/database/db_backup_$DATE.sql"
    local compressed_file="$backup_file.gz"
    
    # Set password for pg_dump (if needed)
    export PGPASSWORD="$DB_PASSWORD"
    
    # Perform database backup
    if pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        --verbose --clean --if-exists --create > "$backup_file"; then
        
        # Compress the backup
        if gzip "$backup_file"; then
            log "Database backup completed successfully: $compressed_file"
            
            # Verify backup integrity
            if gunzip -t "$compressed_file"; then
                log "Database backup integrity verified"
            else
                error_exit "Database backup integrity check failed"
            fi
        else
            error_exit "Failed to compress database backup"
        fi
    else
        error_exit "Database backup failed"
    fi
    
    # Unset password
    unset PGPASSWORD
}

# Log files backup function
backup_logs() {
    log "Starting log files backup..."
    
    local log_backup_file="$BACKUP_DIR/logs/logs_backup_$DATE.tar.gz"
    
    # Check if log directory exists
    if [ ! -d "$LOG_DIR" ]; then
        log "Warning: Log directory $LOG_DIR does not exist, skipping log backup"
        return 0
    fi
    
    # Create tar archive of log files
    if tar -czf "$log_backup_file" -C "$(dirname "$LOG_DIR")" "$(basename "$LOG_DIR")" 2>/dev/null; then
        log "Log files backup completed successfully: $log_backup_file"
    else
        log "Warning: Log files backup failed or no log files found"
    fi
}

# Application files backup function (optional)
backup_application() {
    log "Starting application files backup..."
    
    local app_backup_file="$BACKUP_DIR/application/app_backup_$DATE.tar.gz"
    local app_dir="/opt/online-shop"  # Adjust this path as needed
    
    # Create application backup directory
    mkdir -p "$BACKUP_DIR/application"
    
    # Check if application directory exists
    if [ ! -d "$app_dir" ]; then
        log "Warning: Application directory $app_dir does not exist, skipping application backup"
        return 0
    fi
    
    # Exclude certain directories and files
    local exclude_patterns=(
        "--exclude=*.log"
        "--exclude=tmp/*"
        "--exclude=cache/*"
        "--exclude=.git/*"
        "--exclude=node_modules/*"
        "--exclude=vendor/*"
    )
    
    # Create tar archive of application files
    if tar -czf "$app_backup_file" "${exclude_patterns[@]}" -C "$(dirname "$app_dir")" "$(basename "$app_dir")"; then
        log "Application files backup completed successfully: $app_backup_file"
    else
        log "Warning: Application files backup failed"
    fi
}

# Cleanup old backups
cleanup_old_backups() {
    log "Cleaning up old backups (older than $RETENTION_DAYS days)..."
    
    # Clean database backups
    find "$BACKUP_DIR/database" -name "db_backup_*.sql.gz" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
    
    # Clean log backups
    find "$BACKUP_DIR/logs" -name "logs_backup_*.tar.gz" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
    
    # Clean application backups
    find "$BACKUP_DIR/application" -name "app_backup_*.tar.gz" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
    
    # Clean backup logs older than 90 days
    find "$BACKUP_DIR" -name "backup.log.*" -mtime +90 -delete 2>/dev/null || true
    
    log "Cleanup completed"
}

# Upload to cloud storage (optional)
upload_to_cloud() {
    log "Uploading backups to cloud storage..."
    
    # This is a placeholder for cloud upload functionality
    # You can implement upload to AWS S3, Google Cloud Storage, etc.
    
    # Example for AWS S3:
    # aws s3 sync "$BACKUP_DIR" s3://your-backup-bucket/online-shop/ --delete
    
    # Example for Google Cloud Storage:
    # gsutil -m rsync -r -d "$BACKUP_DIR" gs://your-backup-bucket/online-shop/
    
    log "Cloud upload completed (placeholder)"
}

# Send notification
send_notification() {
    local status="$1"
    local message="$2"
    
    log "Sending backup notification: $status"
    
    # This is a placeholder for notification functionality
    # You can implement email, Slack, Discord, etc. notifications
    
    # Example email notification:
    # echo "$message" | mail -s "Online Shop Backup $status" admin@example.com
    
    # Example Slack notification:
    # curl -X POST -H 'Content-type: application/json' \
    #   --data "{\"text\":\"Online Shop Backup $status: $message\"}" \
    #   YOUR_SLACK_WEBHOOK_URL
    
    log "Notification sent (placeholder)"
}

# Health check function
health_check() {
    log "Performing health check..."
    
    # Check disk space
    local available_space=$(df "$BACKUP_DIR" | awk 'NR==2 {print $4}')
    local required_space=1048576  # 1GB in KB
    
    if [ "$available_space" -lt "$required_space" ]; then
        error_exit "Insufficient disk space for backup (available: ${available_space}KB, required: ${required_space}KB)"
    fi
    
    # Check database connectivity
    export PGPASSWORD="$DB_PASSWORD"
    if ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; then
        error_exit "Cannot connect to database"
    fi
    unset PGPASSWORD
    
    log "Health check passed"
}

# Rotate backup logs
rotate_logs() {
    local log_file="$BACKUP_DIR/backup.log"
    local max_size=10485760  # 10MB in bytes
    
    if [ -f "$log_file" ] && [ $(stat -f%z "$log_file" 2>/dev/null || stat -c%s "$log_file" 2>/dev/null || echo 0) -gt $max_size ]; then
        mv "$log_file" "$log_file.$(date +%Y%m%d_%H%M%S)"
        log "Backup log rotated"
    fi
}

# Main backup function
main() {
    log "=== Starting Online Shop Backup Process ==="
    
    # Rotate logs first
    rotate_logs
    
    local start_time=$(date +%s)
    local success=true
    local error_message=""
    
    # Perform health check
    if ! health_check; then
        success=false
        error_message="Health check failed"
    fi
    
    # Check dependencies
    if [ "$success" = true ]; then
        if ! check_dependencies; then
            success=false
            error_message="Dependency check failed"
        fi
    fi
    
    # Perform backups
    if [ "$success" = true ]; then
        if ! backup_database; then
            success=false
            error_message="Database backup failed"
        fi
    fi
    
    if [ "$success" = true ]; then
        backup_logs  # Non-critical, don't fail on error
        backup_application  # Non-critical, don't fail on error
    fi
    
    # Cleanup old backups
    if [ "$success" = true ]; then
        cleanup_old_backups
    fi
    
    # Upload to cloud (if configured)
    if [ "$success" = true ] && [ "${ENABLE_CLOUD_UPLOAD:-false}" = "true" ]; then
        upload_to_cloud
    fi
    
    # Calculate duration
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Send notification
    if [ "$success" = true ]; then
        local message="Backup completed successfully in ${duration} seconds"
        log "$message"
        send_notification "SUCCESS" "$message"
    else
        local message="Backup failed: $error_message"
        log "$message"
        send_notification "FAILED" "$message"
        exit 1
    fi
    
    log "=== Backup Process Completed ==="
}

# Load environment variables if config file exists
if [ -f "/etc/online-shop/backup.conf" ]; then
    source "/etc/online-shop/backup.conf"
fi

# Run main function
main "$@"