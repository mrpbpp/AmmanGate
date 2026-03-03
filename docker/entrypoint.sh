#!/bin/bash
set -e

echo "======================================"
echo " AmmanGate Docker Container"
echo "======================================"

# Function to update ClamAV definitions
update_clamav() {
    echo "Updating ClamAV virus definitions..."
    freshclam --datadir=/var/lib/clamav --config-file=/etc/clamav/clamd.conf || echo "Freshclam update failed, using built-in definitions"
}

# Function to start ClamAV
start_clamav() {
    echo "Starting ClamAV..."

    # Update virus definitions on first run or if database is old
    if [ ! -f "/var/lib/clamav/daily.cvd" ] || [ $(find /var/lib/clamav/daily.cvd -mtime +7 2>/dev/null) ]; then
        update_clamav
    fi

    # Start ClamAV daemon
    clamd -c /etc/clamav/clamd.conf --config-file=/etc/clamav/clamd.conf

    # Wait for ClamAV socket to be ready
    echo "Waiting for ClamAV to start..."
    for i in {1..30}; do
        if [ -S "/var/run/clamav/clamd.ctl" ]; then
            echo "ClamAV is ready!"
            break
        fi
        sleep 1
    done

    # Run a quick scan to verify ClamAV is working
    echo "Testing ClamAV..."
    clamdscan --version || echo "Warning: ClamAV may not be working properly"
}

# Function to start Suricata
start_suricata() {
    echo "Starting Suricata..."

    # Create Suricata log file if it doesn't exist
    touch /var/log/suricata/eve.json

    # Start Suricata in IDS mode
    suricata -c /etc/suricata/suricata.yaml -i eth0 -D --set pid-file=/var/run/suricata/suricata.pid

    echo "Suricata started"
}

# Function to setup network interfaces for Suricata
setup_network() {
    echo "Setting up network interfaces..."

    # Get the primary network interface
    INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1)

    if [ -z "$INTERFACE" ]; then
        echo "Warning: Could not detect network interface"
        INTERFACE="eth0"
    fi

    echo "Using network interface: $INTERFACE"
    export AMMANGATE_INTERFACE=$INTERFACE
}

# Main initialization
main() {
    echo "Initializing AmmanGate..."

    # Create necessary directories
    mkdir -p /ammangate/data
    mkdir -p /var/log/suricata
    mkdir -p /var/run/clamav
    mkdir -p /var/run/suricata
    mkdir -p /var/lib/clamav

    # Setup networking
    setup_network

    # Start ClamAV
    start_clamav

    # Start Suricata (optional - comment out if not needed)
    # start_suricata

    echo "======================================"
    echo " AmmanGate is starting..."
    echo " API: http://0.0.0.0:8787/v1"
    echo "======================================"

    # Execute the main command
    exec "$@"
}

# Run main function
main "$@"

# Keep container running if main process exits
echo "AmmanGate process exited. Container will stop."
