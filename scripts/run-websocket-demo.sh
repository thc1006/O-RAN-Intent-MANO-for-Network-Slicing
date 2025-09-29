#!/bin/bash

# O-RAN Network Slicing WebSocket Demo Runner
# This script sets up and runs the WebSocket demo service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DEFAULT_PORT=8080
DOCKER_MODE=false
DEV_MODE=false

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_help() {
    echo "O-RAN Network Slicing WebSocket Demo"
    echo "===================================="
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -p, --port PORT     Set server port (default: 8080)"
    echo "  -d, --docker        Run using Docker"
    echo "  -dev, --dev-mode    Run in development mode with hot reload"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                  # Run on default port 8080"
    echo "  $0 -p 9090          # Run on port 9090"
    echo "  $0 --docker         # Run using Docker"
    echo "  $0 --dev-mode       # Run in development mode"
    echo ""
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed. Please install Go 1.21 or later."
        exit 1
    fi

    # Check tmux
    if ! command -v tmux &> /dev/null; then
        log_warning "tmux is not installed. Installing tmux..."
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y tmux
        elif command -v yum &> /dev/null; then
            sudo yum install -y tmux
        elif command -v brew &> /dev/null; then
            brew install tmux
        else
            log_error "Could not install tmux. Please install it manually."
            exit 1
        fi
    fi

    # Check Claude CLI (optional)
    if ! command -v claude &> /dev/null; then
        log_warning "Claude CLI is not installed. The service will run in fallback mode."
        log_info "To install Claude CLI, visit: https://claude.ai/cli"
    else
        log_success "Claude CLI is available"
        # Test Claude CLI
        if claude --dangerously-skip-permissions "test" &> /dev/null; then
            log_success "Claude CLI is working correctly"
        else
            log_warning "Claude CLI test failed. Service will use fallback mode."
        fi
    fi

    log_success "Prerequisites check completed"
}

build_server() {
    log_info "Building WebSocket server..."

    cd "$(dirname "$0")/.."

    if [ "$DEV_MODE" = true ]; then
        log_info "Installing air for hot reload..."
        go install github.com/cosmtrek/air@latest
    fi

    # Build the server
    go build -o bin/websocket-server ./cmd/websocket-server/

    log_success "Server built successfully"
}

run_docker() {
    log_info "Starting WebSocket server with Docker..."

    cd "$(dirname "$0")/.."

    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker first."
        exit 1
    fi

    # Check if docker-compose is available
    if command -v docker-compose &> /dev/null; then
        COMPOSE_CMD="docker-compose"
    elif docker compose version &> /dev/null; then
        COMPOSE_CMD="docker compose"
    else
        log_error "Docker Compose is not available. Please install Docker Compose."
        exit 1
    fi

    # Stop any existing containers
    $COMPOSE_CMD -f docker-compose.websocket.yml down

    # Build and start
    WEBSOCKET_PORT=$DEFAULT_PORT $COMPOSE_CMD -f docker-compose.websocket.yml up --build
}

run_native() {
    log_info "Starting WebSocket server (native)..."

    cd "$(dirname "$0")/.."

    if [ "$DEV_MODE" = true ]; then
        log_info "Starting in development mode with hot reload..."
        air -c .air.toml
    else
        ./bin/websocket-server -addr ":$DEFAULT_PORT"
    fi
}

create_air_config() {
    cat > .air.toml << EOF
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = ["-addr", ":$DEFAULT_PORT"]
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/websocket-server/"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata", "web/node_modules"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
EOF
}

cleanup() {
    log_info "Cleaning up..."
    if [ "$DOCKER_MODE" = true ]; then
        cd "$(dirname "$0")/.."
        if command -v docker-compose &> /dev/null; then
            docker-compose -f docker-compose.websocket.yml down
        elif docker compose version &> /dev/null; then
            docker compose -f docker-compose.websocket.yml down
        fi
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--port)
            DEFAULT_PORT="$2"
            shift 2
            ;;
        -d|--docker)
            DOCKER_MODE=true
            shift
            ;;
        -dev|--dev-mode)
            DEV_MODE=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Trap cleanup on exit
trap cleanup EXIT

# Main execution
echo "🚀 O-RAN Network Slicing WebSocket Demo"
echo "========================================"
echo ""

check_prerequisites

if [ "$DOCKER_MODE" = true ]; then
    run_docker
else
    build_server

    if [ "$DEV_MODE" = true ]; then
        create_air_config
    fi

    log_success "Starting server on port $DEFAULT_PORT"
    log_info "Frontend: http://localhost:$DEFAULT_PORT"
    log_info "WebSocket: ws://localhost:$DEFAULT_PORT/ws"
    log_info "Health: http://localhost:$DEFAULT_PORT/health"
    echo ""
    log_info "Press Ctrl+C to stop the server"
    echo ""

    run_native
fi