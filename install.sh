#!/bin/bash
# Orris One-Click Installation Script
# Usage:
#   Install: curl -fsSL https://raw.githubusercontent.com/orris-inc/orris/main/install.sh | bash
#   Update:  curl -fsSL https://raw.githubusercontent.com/orris-inc/orris/main/install.sh | bash -s -- update
#   Or locally: ./install.sh update

set -e

# Command (install or update)
ACTION="${1:-install}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
INSTALL_DIR="${INSTALL_DIR:-./orris}"
DOMAIN="${DOMAIN:-}"
ADMIN_EMAIL="${ADMIN_EMAIL:-}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"

print_banner() {
    echo -e "${BLUE}"
    echo "  ___  ____  ____  ___  ____  "
    echo " / _ \|  _ \|  _ \|_ _|/ ___| "
    echo "| | | | |_) | |_) || | \___ \ "
    echo "| |_| |  _ <|  _ < | |  ___) |"
    echo " \___/|_| \_\_| \_\___||____/ "
    echo -e "${NC}"
    echo -e "${GREEN}Orris One-Click Installation${NC}"
    echo ""
}

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "$1 is not installed. Please install it first."
        exit 1
    fi
}

check_dependencies() {
    log_info "Checking dependencies..."

    check_command "docker"

    # Check for docker compose (v2) or docker-compose (v1)
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
    elif command -v docker-compose &> /dev/null; then
        DOCKER_COMPOSE="docker-compose"
    else
        log_error "Docker Compose is not installed. Please install it first."
        exit 1
    fi

    log_info "Dependencies check passed."
}

generate_secret() {
    openssl rand -base64 32 2>/dev/null || head -c 32 /dev/urandom | base64
}

prompt_config() {
    echo ""
    log_info "Configuration Setup"
    echo "Press Enter to use default values (shown in brackets)"
    echo ""

    # Domain
    if [ -z "$DOMAIN" ]; then
        read -p "Domain name (leave empty for localhost): " DOMAIN
    fi

    # Admin email
    if [ -z "$ADMIN_EMAIL" ]; then
        read -p "Admin email: " ADMIN_EMAIL
    fi

    # Admin password
    if [ -z "$ADMIN_PASSWORD" ]; then
        read -s -p "Admin password (min 8 chars): " ADMIN_PASSWORD
        echo ""
    fi

    # Generate secrets
    JWT_SECRET=$(generate_secret)
    FORWARD_SECRET=$(generate_secret)
    DB_PASSWORD=$(generate_secret | tr -dc 'a-zA-Z0-9' | head -c 16)
}

create_install_dir() {
    log_info "Creating installation directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"
    mkdir -p configs/sub data/mysql data/redis
}

download_files() {
    log_info "Downloading configuration files..."

    REPO_URL="https://raw.githubusercontent.com/orris-inc/orris/main"

    # Download docker-compose.yml
    curl -fsSL "$REPO_URL/docker-compose.yml" -o docker-compose.yml

    # Download .env.example
    curl -fsSL "$REPO_URL/.env.example" -o .env.example

    # Download Caddyfile
    curl -fsSL "$REPO_URL/Caddyfile" -o Caddyfile

    # Download subscription template
    curl -fsSL "$REPO_URL/configs/sub/custom.clash.yaml" -o configs/sub/custom.clash.yaml 2>/dev/null || true

    log_info "Configuration files downloaded."
}

configure_env() {
    log_info "Configuring environment variables..."

    cp .env.example .env

    # Set production mode
    sed -i.bak 's/ORRIS_SERVER_MODE=debug/ORRIS_SERVER_MODE=release/' .env

    # Set database password
    sed -i.bak "s|ORRIS_DATABASE_PASSWORD=password|ORRIS_DATABASE_PASSWORD=$DB_PASSWORD|" .env

    # Set JWT secret
    sed -i.bak "s|ORRIS_AUTH_JWT_SECRET=change-me-in-production|ORRIS_AUTH_JWT_SECRET=$JWT_SECRET|" .env

    # Set forward token secret
    sed -i.bak "s|ORRIS_FORWARD_TOKEN_SIGNING_SECRET=change-me-in-production|ORRIS_FORWARD_TOKEN_SIGNING_SECRET=$FORWARD_SECRET|" .env

    # Set admin credentials
    if [ -n "$ADMIN_EMAIL" ]; then
        sed -i.bak "s|ORRIS_ADMIN_EMAIL=|ORRIS_ADMIN_EMAIL=$ADMIN_EMAIL|" .env
    fi
    if [ -n "$ADMIN_PASSWORD" ]; then
        sed -i.bak "s|ORRIS_ADMIN_PASSWORD=|ORRIS_ADMIN_PASSWORD=$ADMIN_PASSWORD|" .env
    fi

    # Set domain-related configs
    if [ -n "$DOMAIN" ]; then
        sed -i.bak "s|ORRIS_SERVER_BASE_URL=|ORRIS_SERVER_BASE_URL=https://$DOMAIN/api|" .env
        sed -i.bak "s|ORRIS_SERVER_ALLOWED_ORIGINS=|ORRIS_SERVER_ALLOWED_ORIGINS=https://$DOMAIN|" .env
        sed -i.bak "s|ORRIS_SERVER_FRONTEND_CALLBACK_URL=|ORRIS_SERVER_FRONTEND_CALLBACK_URL=https://$DOMAIN/auth/callback|" .env
        sed -i.bak "s|ORRIS_SUBSCRIPTION_BASE_URL=|ORRIS_SUBSCRIPTION_BASE_URL=https://$DOMAIN/api|" .env
        sed -i.bak "s/ORRIS_AUTH_COOKIE_SECURE=false/ORRIS_AUTH_COOKIE_SECURE=true/" .env
        sed -i.bak "s|ORRIS_AUTH_COOKIE_DOMAIN=|ORRIS_AUTH_COOKIE_DOMAIN=$DOMAIN|" .env
    else
        sed -i.bak "s|ORRIS_SERVER_BASE_URL=|ORRIS_SERVER_BASE_URL=http://localhost/api|" .env
        sed -i.bak "s|ORRIS_SERVER_ALLOWED_ORIGINS=|ORRIS_SERVER_ALLOWED_ORIGINS=http://localhost|" .env
        sed -i.bak "s|ORRIS_SERVER_FRONTEND_CALLBACK_URL=|ORRIS_SERVER_FRONTEND_CALLBACK_URL=http://localhost/auth/callback|" .env
        sed -i.bak "s|ORRIS_SUBSCRIPTION_BASE_URL=|ORRIS_SUBSCRIPTION_BASE_URL=http://localhost/api|" .env
    fi

    # Clean up backup files
    rm -f .env.bak

    log_info "Environment configured."
}

configure_caddy() {
    if [ -n "$DOMAIN" ]; then
        log_info "Configuring Caddy for domain: $DOMAIN"

        cat > Caddyfile << EOF
# Caddyfile for Orris
# Automatic HTTPS enabled for $DOMAIN

$DOMAIN {
    # API routes -> backend service
    handle /api/* {
        uri strip_prefix /api
        reverse_proxy orris:8080
    }

    # All other routes -> frontend service
    handle {
        reverse_proxy frontend:80
    }
}
EOF
    else
        log_info "Configuring Caddy for localhost (HTTP only)"

        cat > Caddyfile << EOF
# Caddyfile for Orris
# HTTP mode for localhost

:80 {
    # API routes -> backend service
    handle /api/* {
        uri strip_prefix /api
        reverse_proxy orris:8080
    }

    # All other routes -> frontend service
    handle {
        reverse_proxy frontend:80
    }
}
EOF
    fi
}

start_services() {
    log_info "Starting services..."

    $DOCKER_COMPOSE pull
    $DOCKER_COMPOSE up -d

    log_info "Waiting for services to be ready..."
    sleep 10
}

run_migrations() {
    log_info "Running database migrations..."

    # Wait for MySQL to be fully ready
    for i in {1..30}; do
        if docker exec orris_mysql mysqladmin ping -h localhost -u root -p"$DB_PASSWORD" &>/dev/null; then
            break
        fi
        echo -n "."
        sleep 2
    done
    echo ""

    # Run migrations
    docker exec orris_app /app/orris migrate up

    log_info "Database migrations completed."
}

print_summary() {
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}   Installation Complete!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""

    if [ -n "$DOMAIN" ]; then
        echo -e "  URL:        ${BLUE}https://$DOMAIN${NC}"
    else
        echo -e "  URL:        ${BLUE}http://localhost${NC}"
    fi

    if [ -n "$ADMIN_EMAIL" ]; then
        echo -e "  Admin:      ${BLUE}$ADMIN_EMAIL${NC}"
    fi

    echo ""
    echo "  Useful commands:"
    echo "    cd $INSTALL_DIR"
    echo "    $DOCKER_COMPOSE ps        # View status"
    echo "    $DOCKER_COMPOSE logs -f   # View logs"
    echo "    $DOCKER_COMPOSE down      # Stop services"
    echo "    ./install.sh update       # Update to latest version"
    echo ""

    if [ -n "$DOMAIN" ]; then
        log_warn "Make sure your DNS is pointing to this server!"
        log_warn "Caddy will automatically obtain SSL certificates."
    fi

    echo ""
    log_info "For more information, visit: https://github.com/orris-inc/orris"
}

# Update function - pulls latest images, restarts services, and runs migrations
do_update() {
    echo -e "${BLUE}"
    echo "  ___  ____  ____  ___  ____  "
    echo " / _ \|  _ \|  _ \|_ _|/ ___| "
    echo "| | | | |_) | |_) || | \___ \ "
    echo "| |_| |  _ <|  _ < | |  ___) |"
    echo " \___/|_| \_\_| \_\___||____/ "
    echo -e "${NC}"
    echo -e "${GREEN}Orris Update${NC}"
    echo ""

    check_dependencies

    # Check if we're in the right directory (has docker-compose.yml)
    if [ ! -f "docker-compose.yml" ]; then
        log_error "docker-compose.yml not found in current directory."
        log_error "Please run this command from your Orris installation directory."
        exit 1
    fi

    log_info "Pulling latest images..."
    $DOCKER_COMPOSE pull

    log_info "Restarting services with new images..."
    $DOCKER_COMPOSE up -d

    log_info "Waiting for services to be ready..."
    sleep 10

    # Wait for app container to be healthy
    for i in {1..30}; do
        if docker exec orris_app /app/orris migrate status &>/dev/null; then
            break
        fi
        echo -n "."
        sleep 2
    done
    echo ""

    log_info "Running database migrations..."
    docker exec orris_app /app/orris migrate up

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}   Update Complete!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "  Useful commands:"
    echo "    $DOCKER_COMPOSE ps        # View status"
    echo "    $DOCKER_COMPOSE logs -f   # View logs"
    echo ""
    log_info "Update completed successfully!"
}

# Main installation flow
do_install() {
    print_banner
    check_dependencies
    prompt_config
    create_install_dir
    download_files
    configure_env
    configure_caddy
    start_services
    run_migrations
    print_summary
}

# Show usage
show_usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  install   Install Orris (default)"
    echo "  update    Update Orris to the latest version"
    echo "  help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0              # Install Orris"
    echo "  $0 install      # Install Orris"
    echo "  $0 update       # Update existing installation"
    echo ""
    echo "Remote usage:"
    echo "  curl -fsSL https://raw.githubusercontent.com/orris-inc/orris/main/install.sh | bash"
    echo "  curl -fsSL https://raw.githubusercontent.com/orris-inc/orris/main/install.sh | bash -s -- update"
}

# Main entry point
case "$ACTION" in
    install)
        do_install
        ;;
    update)
        do_update
        ;;
    help|--help|-h)
        show_usage
        ;;
    *)
        log_error "Unknown command: $ACTION"
        show_usage
        exit 1
        ;;
esac
