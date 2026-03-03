#!/bin/bash
# AmmanGate Docker Setup and Build Script
# This script builds and deploys AmmanGate with all dependencies

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "======================================"
echo " AmmanGate Docker Setup"
echo "======================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is installed
check_docker() {
    print_info "Checking Docker installation..."
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed!"
        echo "Please install Docker first:"
        echo "  curl -fsSL https://get.docker.com | sh"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        print_error "Docker Compose is not installed!"
        echo "Please install Docker Compose first"
        exit 1
    fi

    print_info "Docker and Docker Compose are installed"
}

# Build Docker images
build_images() {
    print_info "Building AmmanGate Docker images..."
    cd "$PROJECT_DIR"

    # Use docker compose or docker-compose based on availability
    if docker compose version &> /dev/null; then
        docker compose build
    else
        docker-compose build
    fi

    print_info "Docker images built successfully"
}

# Start containers
start_containers() {
    print_info "Starting AmmanGate containers..."
    cd "$PROJECT_DIR"

    # Use docker compose or docker-compose based on availability
    if docker compose version &> /dev/null; then
        docker compose up -d
    else
        docker-compose up -d
    fi

    print_info "AmmanGate containers started"
}

# Stop containers
stop_containers() {
    print_info "Stopping AmmanGate containers..."
    cd "$PROJECT_DIR"

    if docker compose version &> /dev/null; then
        docker compose down
    else
        docker-compose down
    fi

    print_info "AmmanGate containers stopped"
}

# View logs
view_logs() {
    cd "$PROJECT_DIR"

    if docker compose version &> /dev/null; then
        docker compose logs -f
    else
        docker-compose logs -f
    fi
}

# Show status
show_status() {
    cd "$PROJECT_DIR"

    if docker compose version &> /dev/null; then
        docker compose ps
    else
        docker-compose ps
    fi
}

# Run shell in container
run_shell() {
    cd "$PROJECT_DIR"

    print_info "Opening shell in AmmanGate container..."
    if docker compose version &> /dev/null; then
        docker compose exec bodyguard-core /bin/bash
    else
        docker-compose exec bodyguard-core /bin/bash
    fi
}

# Update ClamAV definitions
update_clamav() {
    print_info "Updating ClamAV virus definitions..."
    cd "$PROJECT_DIR"

    if docker compose version &> /dev/null; then
        docker compose exec bodyguard-core freshclam --datadir=/var/lib/clamav
    else
        docker-compose exec bodyguard-core freshclam --datadir=/var/lib/clamav
    fi

    print_info "ClamAV definitions updated"
}

# Show help
show_help() {
    cat << EOF
AmmanGate Docker Management Script

Usage: $0 [COMMAND]

Commands:
    build       Build Docker images
    start       Start containers
    stop        Stop containers
    restart     Restart containers
    logs        View container logs
    status      Show container status
    shell       Open shell in container
    update-av   Update ClamAV virus definitions
    clean       Remove containers and volumes
    help        Show this help message

Examples:
    $0 build           # Build images
    $0 start           # Start containers
    $0 logs            # View logs
    $0 update-av       # Update antivirus definitions

EOF
}

# Clean everything
clean_all() {
    print_warn "This will remove all containers and volumes!"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cd "$PROJECT_DIR"

        if docker compose version &> /dev/null; then
            docker compose down -v
        else
            docker-compose down -v
        fi

        print_info "Containers and volumes removed"
    else
        print_info "Cancelled"
    fi
}

# Main function
main() {
    case "${1:-help}" in
        build)
            check_docker
            build_images
            ;;
        start)
            check_docker
            start_containers
            ;;
        stop)
            stop_containers
            ;;
        restart)
            stop_containers
            start_containers
            ;;
        logs)
            view_logs
            ;;
        status)
            show_status
            ;;
        shell)
            run_shell
            ;;
        update-av)
            update_clamav
            ;;
        clean)
            clean_all
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "Unknown command: $1"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
