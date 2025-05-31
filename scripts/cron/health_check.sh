#!/bin/bash

# Health check script for online shop application
# This script monitors application health and sends alerts if issues are detected

set -euo pipefail

# Configuration
APP_NAME="online-shop"
APP_URL="${APP_URL:-http://localhost:8080}"
HEALTH_ENDPOINT="${HEALTH_ENDPOINT:-/health}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-online_shop}"
DB_USER="${DB_USER:-postgres}"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
RABBITMQ_HOST="${RABBITMQ_HOST:-localhost}"
RABBITMQ_PORT="${RABBITMQ_PORT:-5672}"
RABBITMQ_USER="${RABBITMQ_USER:-guest}"
LOG_FILE="/var/log/online-shop/health_check.log"
ALERT_WEBHOOK="${ALERT_WEBHOOK:-}"
MAX_RESPONSE_TIME=5000  # milliseconds
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

# Create log directory if it doesn't exist
mkdir -p "$(dirname "$LOG_FILE")"

# Function to log messages
log() {
    echo "[$TIMESTAMP] $1" | tee -a "$LOG_FILE"
}

# Function to send alert
send_alert() {
    local severity="$1"
    local service="$2"
    local message="$3"
    
    log "ALERT [$severity] $service: $message"
    
    # Send webhook notification if configured
    if [ -n "$ALERT_WEBHOOK" ]; then
        curl -s -X POST "$ALERT_WEBHOOK" \
            -H "Content-Type: application/json" \
            -d "{
                \"severity\":\"$severity\",
                \"service\":\"$service\",
                \"message\":\"$message\",
                \"timestamp\":\"$(date -Iseconds)\",
                \"hostname\":\"$(hostname)\"
            }" >/dev/null 2>&1 || log "Failed to send webhook alert"
    fi
}

# Function to check HTTP endpoint
check_http_endpoint() {
    local url="$1"
    local expected_status="${2:-200}"
    local timeout="${3:-10}"
    
    log "Checking HTTP endpoint: $url"
    
    local start_time=$(date +%s%3N)
    local response=$(curl -s -w "%{http_code}|%{time_total}" -m "$timeout" "$url" 2>/dev/null || echo "000|0")
    local end_time=$(date +%s%3N)
    
    local status_code=$(echo "$response" | cut -d'|' -f1)
    local response_time_ms=$(echo "($end_time - $start_time)" | bc)
    
    if [ "$status_code" = "$expected_status" ]; then
        if [ "$response_time_ms" -gt "$MAX_RESPONSE_TIME" ]; then
            send_alert "warning" "HTTP" "Slow response from $url: ${response_time_ms}ms"
            return 1
        else
            log "HTTP endpoint OK: $url (${response_time_ms}ms)"
            return 0
        fi
    else
        send_alert "critical" "HTTP" "HTTP endpoint failed: $url (status: $status_code)"
        return 1
    fi
}

# Function to check database connectivity
check_database() {
    log "Checking database connectivity..."
    
    if command -v pg_isready >/dev/null 2>&1; then
        if PGPASSWORD="$PGPASSWORD" pg_isready -h "$DB_HOST" -p "$DB_PORT" -d "$DB_NAME" -U "$DB_USER" >/dev/null 2>&1; then
            log "Database connectivity OK"
            
            # Check database performance
            local query_time=$(PGPASSWORD="$PGPASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -d "$DB_NAME" -U "$DB_USER" -t -c "SELECT EXTRACT(EPOCH FROM NOW());" 2>/dev/null | tr -d ' ')
            if [ -n "$query_time" ]; then
                log "Database query OK"
                return 0
            else
                send_alert "warning" "Database" "Database query failed"
                return 1
            fi
        else
            send_alert "critical" "Database" "Database connection failed"
            return 1
        fi
    else
        log "WARNING: pg_isready not available, skipping database check"
        return 0
    fi
}

# Function to check Redis connectivity
check_redis() {
    log "Checking Redis connectivity..."
    
    if command -v redis-cli >/dev/null 2>&1; then
        if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping >/dev/null 2>&1; then
            log "Redis connectivity OK"
            return 0
        else
            send_alert "critical" "Redis" "Redis connection failed"
            return 1
        fi
    else
        log "WARNING: redis-cli not available, skipping Redis check"
        return 0
    fi
}

# Function to check RabbitMQ connectivity
check_rabbitmq() {
    log "Checking RabbitMQ connectivity..."
    
    if command -v rabbitmqctl >/dev/null 2>&1; then
        if rabbitmqctl -n "rabbit@$RABBITMQ_HOST" status >/dev/null 2>&1; then
            log "RabbitMQ connectivity OK"
            return 0
        else
            send_alert "critical" "RabbitMQ" "RabbitMQ connection failed"
            return 1
        fi
    else
        # Alternative check using netcat
        if command -v nc >/dev/null 2>&1; then
            if nc -z "$RABBITMQ_HOST" "$RABBITMQ_PORT" 2>/dev/null; then
                log "RabbitMQ port accessible"
                return 0
            else
                send_alert "critical" "RabbitMQ" "RabbitMQ port not accessible"
                return 1
            fi
        else
            log "WARNING: rabbitmqctl and nc not available, skipping RabbitMQ check"
            return 0
        fi
    fi
}

# Function to check disk space
check_disk_space() {
    log "Checking disk space..."
    
    local threshold=90
    local usage=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
    
    if [ "$usage" -gt "$threshold" ]; then
        send_alert "critical" "Disk" "Disk usage is $usage% (threshold: $threshold%)"
        return 1
    elif [ "$usage" -gt 80 ]; then
        send_alert "warning" "Disk" "Disk usage is $usage%"
        return 1
    else
        log "Disk space OK: $usage%"
        return 0
    fi
}

# Function to check memory usage
check_memory() {
    log "Checking memory usage..."
    
    local threshold=90
    local usage=$(free | awk 'NR==2{printf "%.0f", $3*100/$2}')
    
    if [ "$usage" -gt "$threshold" ]; then
        send_alert "critical" "Memory" "Memory usage is $usage% (threshold: $threshold%)"
        return 1
    elif [ "$usage" -gt 80 ]; then
        send_alert "warning" "Memory" "Memory usage is $usage%"
        return 1
    else
        log "Memory usage OK: $usage%"
        return 0
    fi
}

# Function to check application processes
check_processes() {
    log "Checking application processes..."
    
    local app_processes=$(pgrep -f "$APP_NAME" | wc -l)
    
    if [ "$app_processes" -eq 0 ]; then
        send_alert "critical" "Process" "No $APP_NAME processes running"
        return 1
    else
        log "Application processes OK: $app_processes running"
        return 0
    fi
}

# Function to check log file sizes
check_log_sizes() {
    log "Checking log file sizes..."
    
    local log_dir="/var/log/online-shop"
    local max_size_mb=100
    
    if [ -d "$log_dir" ]; then
        while IFS= read -r -d '' log_file; do
            local size_mb=$(du -m "$log_file" | cut -f1)
            if [ "$size_mb" -gt "$max_size_mb" ]; then
                send_alert "warning" "Logs" "Log file $log_file is ${size_mb}MB (threshold: ${max_size_mb}MB)"
            fi
        done < <(find "$log_dir" -name "*.log" -print0)
        
        log "Log file sizes checked"
        return 0
    else
        log "WARNING: Log directory $log_dir not found"
        return 0
    fi
}

# Function to generate health report
generate_health_report() {
    local overall_status="$1"
    local failed_checks="$2"
    
    cat > "/tmp/health_report_$(date +%Y%m%d_%H%M%S).json" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "hostname": "$(hostname)",
    "overall_status": "$overall_status",
    "failed_checks": $failed_checks,
    "checks": {
        "http_endpoint": "$(check_http_endpoint "$APP_URL$HEALTH_ENDPOINT" && echo "OK" || echo "FAILED")",
        "database": "$(check_database && echo "OK" || echo "FAILED")",
        "redis": "$(check_redis && echo "OK" || echo "FAILED")",
        "rabbitmq": "$(check_rabbitmq && echo "OK" || echo "FAILED")",
        "disk_space": "$(check_disk_space && echo "OK" || echo "FAILED")",
        "memory": "$(check_memory && echo "OK" || echo "FAILED")",
        "processes": "$(check_processes && echo "OK" || echo "FAILED")"
    }
}
EOF
}

# Main execution
main() {
    log "Starting health check for $APP_NAME..."
    
    local failed_checks=0
    local checks=(
        "check_http_endpoint $APP_URL$HEALTH_ENDPOINT"
        "check_database"
        "check_redis"
        "check_rabbitmq"
        "check_disk_space"
        "check_memory"
        "check_processes"
        "check_log_sizes"
    )
    
    # Run all health checks
    for check in "${checks[@]}"; do
        if ! eval "$check"; then
            ((failed_checks++))
        fi
    done
    
    # Determine overall status
    local overall_status="healthy"
    if [ "$failed_checks" -gt 0 ]; then
        if [ "$failed_checks" -gt 3 ]; then
            overall_status="critical"
        else
            overall_status="degraded"
        fi
    fi
    
    log "Health check completed: $overall_status ($failed_checks failed checks)"
    
    # Send summary alert if there are failures
    if [ "$failed_checks" -gt 0 ]; then
        send_alert "$overall_status" "HealthCheck" "$failed_checks health checks failed"
    fi
    
    # Generate health report
    generate_health_report "$overall_status" "$failed_checks"
    
    # Exit with error code if critical
    if [ "$overall_status" = "critical" ]; then
        exit 1
    fi
}

# Handle script interruption
trap 'log "Health check interrupted"; exit 1' INT TERM

# Run main function
main "$@"