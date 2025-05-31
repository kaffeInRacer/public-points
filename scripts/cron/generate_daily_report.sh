#!/bin/bash

# Daily report generator for online shop application
# This script generates a comprehensive daily summary report

set -euo pipefail

# Configuration
APP_NAME="online-shop"
LOG_DIR="/var/log/online-shop"
REPORT_DIR="/var/reports/online-shop"
REPORT_FILE="$REPORT_DIR/daily_report_$(date +%Y%m%d).json"
HTML_REPORT="$REPORT_DIR/daily_report_$(date +%Y%m%d).html"
TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")
DATE=$(date +%Y-%m-%d)

# Create report directory if it doesn't exist
mkdir -p "$REPORT_DIR"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_DIR/daily_report.log"
}

# Function to get system metrics
get_system_metrics() {
    local cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | sed 's/%us,//')
    local memory_usage=$(free | awk 'NR==2{printf "%.1f", $3*100/$2}')
    local disk_usage=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
    local load_average=$(uptime | awk -F'load average:' '{print $2}' | sed 's/^ *//')
    local uptime_info=$(uptime | awk '{print $3,$4}' | sed 's/,//')
    
    cat << EOF
    "system_metrics": {
        "cpu_usage_percent": "$cpu_usage",
        "memory_usage_percent": "$memory_usage",
        "disk_usage_percent": "$disk_usage",
        "load_average": "$load_average",
        "uptime": "$uptime_info"
    }
EOF
}

# Function to get application metrics
get_application_metrics() {
    local app_processes=$(pgrep -f "$APP_NAME" | wc -l)
    local app_memory=0
    local app_cpu=0
    
    if [ "$app_processes" -gt 0 ]; then
        app_memory=$(ps aux | grep "$APP_NAME" | grep -v grep | awk '{sum+=$4} END {printf "%.1f", sum}')
        app_cpu=$(ps aux | grep "$APP_NAME" | grep -v grep | awk '{sum+=$3} END {printf "%.1f", sum}')
    fi
    
    cat << EOF
    "application_metrics": {
        "processes_running": $app_processes,
        "memory_usage_percent": "$app_memory",
        "cpu_usage_percent": "$app_cpu"
    }
EOF
}

# Function to analyze log files
analyze_logs() {
    local error_count=0
    local warning_count=0
    local info_count=0
    local total_requests=0
    
    if [ -f "$LOG_DIR/application.log" ]; then
        error_count=$(grep -c "ERROR" "$LOG_DIR/application.log" 2>/dev/null || echo 0)
        warning_count=$(grep -c "WARNING\|WARN" "$LOG_DIR/application.log" 2>/dev/null || echo 0)
        info_count=$(grep -c "INFO" "$LOG_DIR/application.log" 2>/dev/null || echo 0)
    fi
    
    if [ -f "$LOG_DIR/access.log" ]; then
        total_requests=$(wc -l < "$LOG_DIR/access.log" 2>/dev/null || echo 0)
    fi
    
    # Get top errors
    local top_errors=""
    if [ -f "$LOG_DIR/application.log" ]; then
        top_errors=$(grep "ERROR" "$LOG_DIR/application.log" 2>/dev/null | tail -5 | sed 's/"/\\"/g' | awk '{print "\"" $0 "\""}' | paste -sd, || echo '""')
    fi
    
    cat << EOF
    "log_analysis": {
        "error_count": $error_count,
        "warning_count": $warning_count,
        "info_count": $info_count,
        "total_requests": $total_requests,
        "top_errors": [$top_errors]
    }
EOF
}

# Function to get database metrics
get_database_metrics() {
    local db_status="unknown"
    local db_connections=0
    local db_size="0"
    
    if command -v pg_isready >/dev/null 2>&1; then
        if PGPASSWORD="$PGPASSWORD" pg_isready -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5432}" -d "${DB_NAME:-online_shop}" -U "${DB_USER:-postgres}" >/dev/null 2>&1; then
            db_status="connected"
            
            # Get connection count
            db_connections=$(PGPASSWORD="$PGPASSWORD" psql -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5432}" -d "${DB_NAME:-online_shop}" -U "${DB_USER:-postgres}" -t -c "SELECT count(*) FROM pg_stat_activity;" 2>/dev/null | tr -d ' ' || echo 0)
            
            # Get database size
            db_size=$(PGPASSWORD="$PGPASSWORD" psql -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5432}" -d "${DB_NAME:-online_shop}" -U "${DB_USER:-postgres}" -t -c "SELECT pg_size_pretty(pg_database_size('${DB_NAME:-online_shop}'));" 2>/dev/null | tr -d ' ' || echo "unknown")
        else
            db_status="disconnected"
        fi
    fi
    
    cat << EOF
    "database_metrics": {
        "status": "$db_status",
        "active_connections": $db_connections,
        "database_size": "$db_size"
    }
EOF
}

# Function to get Redis metrics
get_redis_metrics() {
    local redis_status="unknown"
    local redis_memory="0"
    local redis_keys=0
    
    if command -v redis-cli >/dev/null 2>&1; then
        if redis-cli -h "${REDIS_HOST:-localhost}" -p "${REDIS_PORT:-6379}" ping >/dev/null 2>&1; then
            redis_status="connected"
            redis_memory=$(redis-cli -h "${REDIS_HOST:-localhost}" -p "${REDIS_PORT:-6379}" info memory | grep used_memory_human | cut -d: -f2 | tr -d '\r' || echo "unknown")
            redis_keys=$(redis-cli -h "${REDIS_HOST:-localhost}" -p "${REDIS_PORT:-6379}" dbsize 2>/dev/null || echo 0)
        else
            redis_status="disconnected"
        fi
    fi
    
    cat << EOF
    "redis_metrics": {
        "status": "$redis_status",
        "memory_usage": "$redis_memory",
        "total_keys": $redis_keys
    }
EOF
}

# Function to get backup status
get_backup_status() {
    local last_backup="never"
    local backup_size="0"
    local backup_status="unknown"
    
    local backup_dir="/var/backups/online-shop/database"
    if [ -d "$backup_dir" ]; then
        local latest_backup=$(find "$backup_dir" -name "*.sql.gz" -type f -printf '%T@ %p\n' 2>/dev/null | sort -n | tail -1 | cut -d' ' -f2-)
        if [ -n "$latest_backup" ]; then
            last_backup=$(date -r "$latest_backup" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
            backup_size=$(du -h "$latest_backup" 2>/dev/null | cut -f1 || echo "unknown")
            backup_status="success"
        fi
    fi
    
    cat << EOF
    "backup_status": {
        "last_backup": "$last_backup",
        "backup_size": "$backup_size",
        "status": "$backup_status"
    }
EOF
}

# Function to get security metrics
get_security_metrics() {
    local failed_logins=0
    local suspicious_ips=""
    
    # Check for failed login attempts in auth logs
    if [ -f "/var/log/auth.log" ]; then
        failed_logins=$(grep "$(date +%b\ %d)" /var/log/auth.log | grep -c "Failed password" 2>/dev/null || echo 0)
        suspicious_ips=$(grep "$(date +%b\ %d)" /var/log/auth.log | grep "Failed password" | awk '{print $(NF-3)}' | sort | uniq -c | sort -nr | head -3 | awk '{print $2}' | paste -sd, 2>/dev/null || echo "")
    fi
    
    cat << EOF
    "security_metrics": {
        "failed_logins_today": $failed_logins,
        "suspicious_ips": "$suspicious_ips"
    }
EOF
}

# Function to get performance metrics
get_performance_metrics() {
    local avg_response_time="0"
    local requests_per_minute="0"
    local error_rate="0"
    
    # Calculate from access logs if available
    if [ -f "$LOG_DIR/access.log" ]; then
        # Simple calculation - in real implementation, you'd parse actual access logs
        local total_requests=$(wc -l < "$LOG_DIR/access.log" 2>/dev/null || echo 0)
        if [ "$total_requests" -gt 0 ]; then
            requests_per_minute=$(echo "scale=2; $total_requests / 1440" | bc 2>/dev/null || echo "0")  # Assuming 24 hours
        fi
    fi
    
    cat << EOF
    "performance_metrics": {
        "avg_response_time_ms": "$avg_response_time",
        "requests_per_minute": "$requests_per_minute",
        "error_rate_percent": "$error_rate"
    }
EOF
}

# Function to generate JSON report
generate_json_report() {
    log "Generating JSON report..."
    
    cat > "$REPORT_FILE" << EOF
{
    "report_date": "$DATE",
    "generated_at": "$TIMESTAMP",
    "hostname": "$(hostname)",
    "application": "$APP_NAME",
    $(get_system_metrics),
    $(get_application_metrics),
    $(get_log_analysis),
    $(get_database_metrics),
    $(get_redis_metrics),
    $(get_backup_status),
    $(get_security_metrics),
    $(get_performance_metrics)
}
EOF
    
    log "JSON report generated: $REPORT_FILE"
}

# Function to generate HTML report
generate_html_report() {
    log "Generating HTML report..."
    
    cat > "$HTML_REPORT" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Online Shop Daily Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header { text-align: center; color: #333; border-bottom: 2px solid #007bff; padding-bottom: 10px; margin-bottom: 20px; }
        .metric-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .metric-card { background: #f8f9fa; padding: 15px; border-radius: 6px; border-left: 4px solid #007bff; }
        .metric-title { font-weight: bold; color: #495057; margin-bottom: 10px; }
        .metric-value { font-size: 1.2em; color: #28a745; }
        .status-ok { color: #28a745; }
        .status-warning { color: #ffc107; }
        .status-error { color: #dc3545; }
        .footer { text-align: center; margin-top: 20px; color: #6c757d; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Online Shop Daily Report</h1>
            <p>Generated on: TIMESTAMP_PLACEHOLDER</p>
        </div>
        
        <div class="metric-grid">
            <div class="metric-card">
                <div class="metric-title">System Health</div>
                <div>CPU Usage: <span class="metric-value">CPU_PLACEHOLDER%</span></div>
                <div>Memory Usage: <span class="metric-value">MEMORY_PLACEHOLDER%</span></div>
                <div>Disk Usage: <span class="metric-value">DISK_PLACEHOLDER%</span></div>
            </div>
            
            <div class="metric-card">
                <div class="metric-title">Application Status</div>
                <div>Processes: <span class="metric-value">PROCESSES_PLACEHOLDER</span></div>
                <div>Status: <span class="status-ok">Running</span></div>
            </div>
            
            <div class="metric-card">
                <div class="metric-title">Database</div>
                <div>Status: <span class="status-ok">DB_STATUS_PLACEHOLDER</span></div>
                <div>Connections: <span class="metric-value">DB_CONNECTIONS_PLACEHOLDER</span></div>
                <div>Size: <span class="metric-value">DB_SIZE_PLACEHOLDER</span></div>
            </div>
            
            <div class="metric-card">
                <div class="metric-title">Redis Cache</div>
                <div>Status: <span class="status-ok">REDIS_STATUS_PLACEHOLDER</span></div>
                <div>Memory: <span class="metric-value">REDIS_MEMORY_PLACEHOLDER</span></div>
                <div>Keys: <span class="metric-value">REDIS_KEYS_PLACEHOLDER</span></div>
            </div>
            
            <div class="metric-card">
                <div class="metric-title">Log Summary</div>
                <div>Errors: <span class="status-error">ERROR_COUNT_PLACEHOLDER</span></div>
                <div>Warnings: <span class="status-warning">WARNING_COUNT_PLACEHOLDER</span></div>
                <div>Total Requests: <span class="metric-value">REQUESTS_PLACEHOLDER</span></div>
            </div>
            
            <div class="metric-card">
                <div class="metric-title">Backup Status</div>
                <div>Last Backup: <span class="metric-value">BACKUP_DATE_PLACEHOLDER</span></div>
                <div>Size: <span class="metric-value">BACKUP_SIZE_PLACEHOLDER</span></div>
            </div>
        </div>
        
        <div class="footer">
            <p>Report generated automatically by Online Shop monitoring system</p>
        </div>
    </div>
</body>
</html>
EOF
    
    # Replace placeholders with actual values from JSON report
    if [ -f "$REPORT_FILE" ]; then
        # This is a simplified replacement - in a real implementation, you'd use jq or similar
        sed -i "s/TIMESTAMP_PLACEHOLDER/$TIMESTAMP/g" "$HTML_REPORT"
        sed -i "s/CPU_PLACEHOLDER/$(grep -o '"cpu_usage_percent": "[^"]*"' "$REPORT_FILE" | cut -d'"' -f4)/g" "$HTML_REPORT"
        sed -i "s/MEMORY_PLACEHOLDER/$(grep -o '"memory_usage_percent": "[^"]*"' "$REPORT_FILE" | cut -d'"' -f4)/g" "$HTML_REPORT"
        sed -i "s/DISK_PLACEHOLDER/$(grep -o '"disk_usage_percent": "[^"]*"' "$REPORT_FILE" | cut -d'"' -f4)/g" "$HTML_REPORT"
    fi
    
    log "HTML report generated: $HTML_REPORT"
}

# Function to send report notification
send_report_notification() {
    local report_file="$1"
    
    if [ -n "${DAILY_REPORT_WEBHOOK:-}" ]; then
        # Send notification with report summary
        local summary=$(cat "$report_file" | head -20)
        curl -s -X POST "$DAILY_REPORT_WEBHOOK" \
            -H "Content-Type: application/json" \
            -d "{\"message\":\"Daily report generated\",\"file\":\"$report_file\",\"timestamp\":\"$(date -Iseconds)\"}" \
            >/dev/null 2>&1 || log "Failed to send report notification"
    fi
}

# Function to cleanup old reports
cleanup_old_reports() {
    log "Cleaning up old reports..."
    
    # Keep reports for 30 days
    find "$REPORT_DIR" -name "daily_report_*.json" -mtime +30 -delete 2>/dev/null || true
    find "$REPORT_DIR" -name "daily_report_*.html" -mtime +30 -delete 2>/dev/null || true
    
    log "Old reports cleaned up"
}

# Main execution
main() {
    log "Starting daily report generation for $APP_NAME..."
    
    # Generate reports
    generate_json_report
    generate_html_report
    
    # Send notification
    send_report_notification "$REPORT_FILE"
    
    # Cleanup old reports
    cleanup_old_reports
    
    log "Daily report generation completed successfully"
    log "JSON Report: $REPORT_FILE"
    log "HTML Report: $HTML_REPORT"
}

# Handle script interruption
trap 'log "Daily report generation interrupted"; exit 1' INT TERM

# Run main function
main "$@"