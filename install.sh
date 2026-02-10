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
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default values
INSTALL_DIR="${INSTALL_DIR:-./orris}"
DOMAIN="${DOMAIN:-}"
ADMIN_EMAIL="${ADMIN_EMAIL:-}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"

# Installation mode and component flags
INSTALL_MODE=""          # full or custom
USE_BUILTIN_CADDY="yes"
USE_BUILTIN_MYSQL="yes"
USE_BUILTIN_REDIS="yes"

# External service configs
EXT_MYSQL_HOST=""
EXT_MYSQL_PORT="3306"
EXT_MYSQL_USER="root"
EXT_MYSQL_PASSWORD=""
EXT_MYSQL_DATABASE="orris"

EXT_REDIS_HOST=""
EXT_REDIS_PORT="6379"
EXT_REDIS_PASSWORD=""

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
        return 1
    fi
    return 0
}

detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
    elif [ "$(uname)" == "Darwin" ]; then
        OS="macos"
    else
        OS="unknown"
    fi
}

install_docker() {
    detect_os

    case "$OS" in
        ubuntu|debian)
            log_info "Installing Docker on $OS..."
            sudo apt-get update
            sudo apt-get install -y ca-certificates curl gnupg
            sudo install -m 0755 -d /etc/apt/keyrings
            curl -fsSL https://download.docker.com/linux/$OS/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
            sudo chmod a+r /etc/apt/keyrings/docker.gpg
            echo \
                "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$OS \
                $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
                sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
            sudo apt-get update
            sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            sudo systemctl start docker
            sudo systemctl enable docker
            # Add current user to docker group
            sudo usermod -aG docker $USER
            log_info "Docker installed successfully."
            log_warn "You may need to log out and back in for docker group changes to take effect."
            ;;
        centos|rhel|fedora|rocky|almalinux|alinux)
            log_info "Installing Docker on $OS..."
            sudo yum install -y yum-utils
            sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
            sudo yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            sudo systemctl start docker
            sudo systemctl enable docker
            sudo usermod -aG docker $USER
            log_info "Docker installed successfully."
            log_warn "You may need to log out and back in for docker group changes to take effect."
            ;;
        macos)
            log_error "Docker Desktop is required for macOS."
            log_error "Please download and install from: https://www.docker.com/products/docker-desktop/"
            log_error "After installation, start Docker Desktop and run this script again."
            exit 1
            ;;
        *)
            log_warn "Automatic Docker installation not supported for $OS."
            log_info "Attempting to install using official convenience script..."
            curl -fsSL https://get.docker.com | sudo sh
            sudo systemctl start docker
            sudo systemctl enable docker
            sudo usermod -aG docker $USER
            log_info "Docker installed successfully."
            log_warn "You may need to log out and back in for docker group changes to take effect."
            ;;
    esac
}

check_dependencies() {
    log_info "Checking dependencies..."

    # Check if Docker is installed
    if ! check_command "docker"; then
        log_warn "Docker is not installed."
        read -p "Would you like to install Docker automatically? [Y/n] " INSTALL_DOCKER < /dev/tty
        INSTALL_DOCKER="${INSTALL_DOCKER:-Y}"
        if [[ "$INSTALL_DOCKER" =~ ^[Yy]$ ]]; then
            install_docker
            # Re-check after installation
            if ! check_command "docker"; then
                log_error "Docker installation failed. Please install manually."
                exit 1
            fi
        else
            log_error "Docker is required. Please install it first."
            exit 1
        fi
    fi

    # Check for docker compose (v2) or docker-compose (v1)
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
    elif command -v docker-compose &> /dev/null; then
        DOCKER_COMPOSE="docker-compose"
    else
        log_error "Docker Compose is not installed."
        log_error "If you just installed Docker, please log out and back in, then run this script again."
        exit 1
    fi

    # Check if Docker daemon is running
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running."
        log_info "Attempting to start Docker..."
        if command -v systemctl &> /dev/null; then
            sudo systemctl start docker
            sleep 3
            if ! docker info &> /dev/null; then
                log_error "Failed to start Docker. Please start it manually."
                exit 1
            fi
        else
            log_error "Please start Docker manually and run this script again."
            exit 1
        fi
    fi

    log_info "Dependencies check passed."
}

generate_secret() {
    openssl rand -base64 32 2>/dev/null || head -c 32 /dev/urandom | base64
}

select_install_mode() {
    echo ""
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║           Select Installation Mode                         ║${NC}"
    echo -e "${CYAN}╠════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${CYAN}║  1) Full Installation (Recommended)                        ║${NC}"
    echo -e "${CYAN}║     - Includes: Caddy (reverse proxy), MySQL, Redis        ║${NC}"
    echo -e "${CYAN}║     - Best for: New deployments, quick setup               ║${NC}"
    echo -e "${CYAN}║                                                            ║${NC}"
    echo -e "${CYAN}║  2) Custom Installation                                    ║${NC}"
    echo -e "${CYAN}║     - Choose which components to include                   ║${NC}"
    echo -e "${CYAN}║     - Use your own reverse proxy, MySQL, or Redis          ║${NC}"
    echo -e "${CYAN}║     - Best for: Existing infrastructure, advanced users    ║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    while true; do
        read -p "Select mode [1/2] (default: 1): " MODE_CHOICE < /dev/tty
        MODE_CHOICE="${MODE_CHOICE:-1}"
        case "$MODE_CHOICE" in
            1)
                INSTALL_MODE="full"
                USE_BUILTIN_CADDY="yes"
                USE_BUILTIN_MYSQL="yes"
                USE_BUILTIN_REDIS="yes"
                log_info "Full installation mode selected."
                break
                ;;
            2)
                INSTALL_MODE="custom"
                select_components
                break
                ;;
            *)
                log_error "Invalid choice. Please enter 1 or 2."
                ;;
        esac
    done
}

select_components() {
    echo ""
    log_info "Custom Installation - Select Components"
    echo "For each component, choose whether to use the built-in Docker service"
    echo "or configure an external service."
    echo ""

    # Caddy (Reverse Proxy)
    echo -e "${YELLOW}━━━ Reverse Proxy (Caddy) ━━━${NC}"
    echo "  1) Use built-in Caddy (automatic HTTPS, easy setup)"
    echo "  2) Use external reverse proxy (Nginx, Traefik, etc.)"
    read -p "  Choice [1/2] (default: 1): " CADDY_CHOICE < /dev/tty
    CADDY_CHOICE="${CADDY_CHOICE:-1}"
    if [ "$CADDY_CHOICE" == "2" ]; then
        USE_BUILTIN_CADDY="no"
        echo ""
        log_warn "You'll need to configure your reverse proxy manually."
        log_warn "Backend API runs on port 8080, Frontend on port 3000."
    fi
    echo ""

    # MySQL
    echo -e "${YELLOW}━━━ MySQL Database ━━━${NC}"
    echo "  1) Use built-in MySQL (Docker container)"
    echo "  2) Use external MySQL (existing server, RDS, etc.)"
    read -p "  Choice [1/2] (default: 1): " MYSQL_CHOICE < /dev/tty
    MYSQL_CHOICE="${MYSQL_CHOICE:-1}"
    if [ "$MYSQL_CHOICE" == "2" ]; then
        USE_BUILTIN_MYSQL="no"
        prompt_external_mysql
    fi
    echo ""

    # Redis
    echo -e "${YELLOW}━━━ Redis Cache ━━━${NC}"
    echo "  1) Use built-in Redis (Docker container)"
    echo "  2) Use external Redis (existing server, ElastiCache, etc.)"
    read -p "  Choice [1/2] (default: 1): " REDIS_CHOICE < /dev/tty
    REDIS_CHOICE="${REDIS_CHOICE:-1}"
    if [ "$REDIS_CHOICE" == "2" ]; then
        USE_BUILTIN_REDIS="no"
        prompt_external_redis
    fi
    echo ""

    # Summary
    echo -e "${CYAN}━━━ Configuration Summary ━━━${NC}"
    echo "  Caddy:  $([ "$USE_BUILTIN_CADDY" == "yes" ] && echo "Built-in" || echo "External")"
    echo "  MySQL:  $([ "$USE_BUILTIN_MYSQL" == "yes" ] && echo "Built-in" || echo "External ($EXT_MYSQL_HOST)")"
    echo "  Redis:  $([ "$USE_BUILTIN_REDIS" == "yes" ] && echo "Built-in" || echo "External ($EXT_REDIS_HOST)")"
    echo ""
}

prompt_external_mysql() {
    echo ""
    log_info "Configure External MySQL Connection"
    read -p "  MySQL Host: " EXT_MYSQL_HOST < /dev/tty
    read -p "  MySQL Port [3306]: " EXT_MYSQL_PORT < /dev/tty
    EXT_MYSQL_PORT="${EXT_MYSQL_PORT:-3306}"
    read -p "  MySQL User [root]: " EXT_MYSQL_USER < /dev/tty
    EXT_MYSQL_USER="${EXT_MYSQL_USER:-root}"
    read -s -p "  MySQL Password: " EXT_MYSQL_PASSWORD < /dev/tty
    echo ""
    read -p "  MySQL Database [orris]: " EXT_MYSQL_DATABASE < /dev/tty
    EXT_MYSQL_DATABASE="${EXT_MYSQL_DATABASE:-orris}"

    # Validate connection
    log_info "Testing MySQL connection..."
    if check_command "mysql"; then
        if mysql -h "$EXT_MYSQL_HOST" -P "$EXT_MYSQL_PORT" -u "$EXT_MYSQL_USER" -p"$EXT_MYSQL_PASSWORD" -e "SELECT 1" &>/dev/null; then
            log_info "MySQL connection successful."
        else
            log_warn "Could not connect to MySQL. Please verify your credentials."
            log_warn "Installation will continue, but you may need to fix the connection later."
        fi
    else
        log_warn "mysql client not installed, skipping connection test."
    fi
}

prompt_external_redis() {
    echo ""
    log_info "Configure External Redis Connection"
    read -p "  Redis Host: " EXT_REDIS_HOST < /dev/tty
    read -p "  Redis Port [6379]: " EXT_REDIS_PORT < /dev/tty
    EXT_REDIS_PORT="${EXT_REDIS_PORT:-6379}"
    read -s -p "  Redis Password (leave empty if none): " EXT_REDIS_PASSWORD < /dev/tty
    echo ""

    # Validate connection
    log_info "Testing Redis connection..."
    if check_command "redis-cli"; then
        if [ -n "$EXT_REDIS_PASSWORD" ]; then
            REDIS_TEST=$(redis-cli -h "$EXT_REDIS_HOST" -p "$EXT_REDIS_PORT" -a "$EXT_REDIS_PASSWORD" ping 2>/dev/null)
        else
            REDIS_TEST=$(redis-cli -h "$EXT_REDIS_HOST" -p "$EXT_REDIS_PORT" ping 2>/dev/null)
        fi
        if [ "$REDIS_TEST" == "PONG" ]; then
            log_info "Redis connection successful."
        else
            log_warn "Could not connect to Redis. Please verify your configuration."
            log_warn "Installation will continue, but you may need to fix the connection later."
        fi
    else
        log_warn "redis-cli not installed, skipping connection test."
    fi
}

prompt_config() {
    echo ""
    log_info "Basic Configuration"
    echo "Press Enter to use default values (shown in brackets)"
    echo ""

    # Domain
    if [ -z "$DOMAIN" ]; then
        read -p "Domain name (leave empty for localhost): " DOMAIN < /dev/tty
    fi

    # Admin email
    if [ -z "$ADMIN_EMAIL" ]; then
        read -p "Admin email: " ADMIN_EMAIL < /dev/tty
    fi

    # Admin password
    if [ -z "$ADMIN_PASSWORD" ]; then
        read -s -p "Admin password (min 8 chars): " ADMIN_PASSWORD < /dev/tty
        echo ""
    fi

    # Generate secrets
    JWT_SECRET=$(generate_secret)
    FORWARD_SECRET=$(generate_secret)

    # Only generate DB password if using built-in MySQL
    if [ "$USE_BUILTIN_MYSQL" == "yes" ]; then
        DB_PASSWORD=$(generate_secret | tr -dc 'a-zA-Z0-9' | head -c 16)
    else
        DB_PASSWORD="$EXT_MYSQL_PASSWORD"
    fi
}

create_install_dir() {
    log_info "Creating installation directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"
    mkdir -p configs/sub

    # Only create data directories for built-in services
    if [ "$USE_BUILTIN_MYSQL" == "yes" ]; then
        mkdir -p data/mysql
    fi
    if [ "$USE_BUILTIN_REDIS" == "yes" ]; then
        mkdir -p data/redis
    fi
}

download_files() {
    log_info "Downloading configuration files..."

    REPO_URL="https://raw.githubusercontent.com/orris-inc/orris/main"

    # Download .env.example
    curl -fsSL "$REPO_URL/.env.example" -o .env.example

    # Download Caddyfile (only if using built-in Caddy)
    if [ "$USE_BUILTIN_CADDY" == "yes" ]; then
        curl -fsSL "$REPO_URL/Caddyfile" -o Caddyfile
    fi

    # Download subscription template
    curl -fsSL "$REPO_URL/configs/sub/custom.clash.yaml" -o configs/sub/custom.clash.yaml 2>/dev/null || true

    log_info "Configuration files downloaded."
}

generate_docker_compose() {
    log_info "Generating docker-compose.yml..."

    # Start with header and Caddy (if using built-in)
    cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
EOF

    # Add Caddy service first if using built-in
    if [ "$USE_BUILTIN_CADDY" == "yes" ]; then
        cat >> docker-compose.yml << 'EOF'
  caddy:
    image: caddy:2-alpine
    container_name: orris_caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    depends_on:
      - frontend
      - orris

EOF
    fi

    # Add frontend service
    cat >> docker-compose.yml << 'EOF'
  frontend:
    image: ghcr.io/orris-inc/orris-frontend:latest
    container_name: orris_frontend
    restart: unless-stopped
EOF

    if [ "$USE_BUILTIN_CADDY" == "yes" ]; then
        cat >> docker-compose.yml << 'EOF'
    expose:
      - "80"
EOF
    else
        cat >> docker-compose.yml << 'EOF'
    ports:
      - "3000:80"
EOF
    fi

    cat >> docker-compose.yml << 'EOF'
    depends_on:
      - orris

EOF

    # Add orris service
    cat >> docker-compose.yml << 'EOF'
  orris:
    image: ghcr.io/orris-inc/orris:latest
    container_name: orris_app
    restart: unless-stopped
EOF

    if [ "$USE_BUILTIN_CADDY" == "yes" ]; then
        cat >> docker-compose.yml << 'EOF'
    expose:
      - "8080"
EOF
    else
        cat >> docker-compose.yml << 'EOF'
    ports:
      - "8080:8080"
EOF
    fi

    cat >> docker-compose.yml << 'EOF'
    volumes:
      - ./configs:/app/configs:ro
    env_file:
      - .env
EOF

    # Add depends_on for built-in services
    if [ "$USE_BUILTIN_MYSQL" == "yes" ] || [ "$USE_BUILTIN_REDIS" == "yes" ]; then
        echo "    depends_on:" >> docker-compose.yml
        if [ "$USE_BUILTIN_MYSQL" == "yes" ]; then
            cat >> docker-compose.yml << 'EOF'
      mysql:
        condition: service_healthy
EOF
        fi
        if [ "$USE_BUILTIN_REDIS" == "yes" ]; then
            cat >> docker-compose.yml << 'EOF'
      redis:
        condition: service_healthy
EOF
        fi
    fi

    # Add MySQL service if using built-in
    if [ "$USE_BUILTIN_MYSQL" == "yes" ]; then
        cat >> docker-compose.yml << 'EOF'

  mysql:
    image: mysql:8.0
    container_name: orris_mysql
    restart: unless-stopped
    env_file:
      - .env
    environment:
      MYSQL_ROOT_PASSWORD: ${ORRIS_DATABASE_PASSWORD:-password}
      MYSQL_DATABASE: ${ORRIS_DATABASE_DATABASE:-orris}
      MYSQL_USER: ${ORRIS_DATABASE_USERNAME:-orris}
      MYSQL_PASSWORD: ${ORRIS_DATABASE_PASSWORD:-password}
    ports:
      - "${ORRIS_DATABASE_PORT:-3306}:3306"
    volumes:
      - ./data/mysql:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      timeout: 20s
      retries: 10
EOF
    fi

    # Add Redis service if using built-in
    if [ "$USE_BUILTIN_REDIS" == "yes" ]; then
        cat >> docker-compose.yml << 'EOF'

  redis:
    image: redis:7-alpine
    container_name: orris_redis
    restart: unless-stopped
    ports:
      - "${ORRIS_REDIS_PORT:-6379}:6379"
    volumes:
      - ./data/redis:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    command: redis-server --appendonly yes
EOF
    fi

    # Add volumes section
    if [ "$USE_BUILTIN_CADDY" == "yes" ]; then
        cat >> docker-compose.yml << 'EOF'

volumes:
  caddy_data:
  caddy_config:
EOF
    fi

    log_info "docker-compose.yml generated."
}

configure_env() {
    log_info "Configuring environment variables..."

    cp .env.example .env

    # Set production mode
    sed -i.bak 's/ORRIS_SERVER_MODE=debug/ORRIS_SERVER_MODE=release/' .env

    # Configure database connection
    if [ "$USE_BUILTIN_MYSQL" == "yes" ]; then
        sed -i.bak "s|ORRIS_DATABASE_HOST=localhost|ORRIS_DATABASE_HOST=mysql|" .env
        sed -i.bak "s|ORRIS_DATABASE_PASSWORD=password|ORRIS_DATABASE_PASSWORD=$DB_PASSWORD|" .env
    else
        sed -i.bak "s|ORRIS_DATABASE_HOST=localhost|ORRIS_DATABASE_HOST=$EXT_MYSQL_HOST|" .env
        sed -i.bak "s|ORRIS_DATABASE_PORT=3306|ORRIS_DATABASE_PORT=$EXT_MYSQL_PORT|" .env
        sed -i.bak "s|ORRIS_DATABASE_USER=root|ORRIS_DATABASE_USER=$EXT_MYSQL_USER|" .env
        sed -i.bak "s|ORRIS_DATABASE_PASSWORD=password|ORRIS_DATABASE_PASSWORD=$EXT_MYSQL_PASSWORD|" .env
        sed -i.bak "s|ORRIS_DATABASE_NAME=orris|ORRIS_DATABASE_NAME=$EXT_MYSQL_DATABASE|" .env
    fi

    # Configure Redis connection
    if [ "$USE_BUILTIN_REDIS" == "yes" ]; then
        sed -i.bak "s|ORRIS_REDIS_HOST=localhost|ORRIS_REDIS_HOST=redis|" .env
    else
        sed -i.bak "s|ORRIS_REDIS_HOST=localhost|ORRIS_REDIS_HOST=$EXT_REDIS_HOST|" .env
        sed -i.bak "s|ORRIS_REDIS_PORT=6379|ORRIS_REDIS_PORT=$EXT_REDIS_PORT|" .env
        if [ -n "$EXT_REDIS_PASSWORD" ]; then
            sed -i.bak "s|ORRIS_REDIS_PASSWORD=|ORRIS_REDIS_PASSWORD=$EXT_REDIS_PASSWORD|" .env
        fi
    fi

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
    # Skip if not using built-in Caddy
    if [ "$USE_BUILTIN_CADDY" != "yes" ]; then
        return
    fi

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
    if [ "$USE_BUILTIN_MYSQL" == "yes" ]; then
        for i in {1..30}; do
            if docker exec orris_mysql mysqladmin ping -h localhost -u root -p"$DB_PASSWORD" &>/dev/null; then
                break
            fi
            echo -n "."
            sleep 2
        done
        echo ""
    else
        # For external MySQL, wait for app to be ready
        for i in {1..30}; do
            if docker exec orris_app /app/orris migrate status &>/dev/null; then
                break
            fi
            echo -n "."
            sleep 2
        done
        echo ""
    fi

    # Run migrations
    docker exec orris_app /app/orris migrate up

    log_info "Database migrations completed."

    # Restart orris app to trigger admin user seeding
    # seedAdminUser runs on server startup but the first attempt fails because
    # the users table does not exist yet (migrations had not been applied).
    log_info "Restarting orris to apply initial configuration..."
    $DOCKER_COMPOSE restart orris
    sleep 5
}

print_reverse_proxy_examples() {
    local PROXY_DOMAIN="${DOMAIN:-example.com}"
    local USE_HTTPS="yes"
    if [ -z "$DOMAIN" ]; then
        USE_HTTPS="no"
    fi

    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  Reverse Proxy Configuration Examples${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "  Backend API:  http://localhost:8080"
    echo "  Frontend:     http://localhost:3000"
    echo "  Routing:      /api/* -> Backend, /* -> Frontend"
    echo ""

    # Nginx configuration
    echo -e "${YELLOW}▸ Nginx Configuration${NC}"
    echo ""
    if [ "$USE_HTTPS" == "yes" ]; then
        cat << EOF
  server {
      listen 80;
      server_name $PROXY_DOMAIN;
      return 301 https://\$server_name\$request_uri;
  }

  server {
      listen 443 ssl http2;
      server_name $PROXY_DOMAIN;

      ssl_certificate     /path/to/cert.pem;
      ssl_certificate_key /path/to/key.pem;

      # API routes -> backend
      location /api/ {
          rewrite ^/api/(.*)\$ /\$1 break;
          proxy_pass http://127.0.0.1:8080;
          proxy_set_header Host \$host;
          proxy_set_header X-Real-IP \$remote_addr;
          proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
          proxy_set_header X-Forwarded-Proto \$scheme;
      }

      # All other routes -> frontend
      location / {
          proxy_pass http://127.0.0.1:3000;
          proxy_set_header Host \$host;
          proxy_set_header X-Real-IP \$remote_addr;
          proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
          proxy_set_header X-Forwarded-Proto \$scheme;
      }
  }
EOF
    else
        cat << EOF
  server {
      listen 80;
      server_name localhost;

      # API routes -> backend
      location /api/ {
          rewrite ^/api/(.*)\$ /\$1 break;
          proxy_pass http://127.0.0.1:8080;
          proxy_set_header Host \$host;
          proxy_set_header X-Real-IP \$remote_addr;
          proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
          proxy_set_header X-Forwarded-Proto \$scheme;
      }

      # All other routes -> frontend
      location / {
          proxy_pass http://127.0.0.1:3000;
          proxy_set_header Host \$host;
          proxy_set_header X-Real-IP \$remote_addr;
          proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
          proxy_set_header X-Forwarded-Proto \$scheme;
      }
  }
EOF
    fi

    echo ""
    echo -e "${YELLOW}▸ Caddy Configuration${NC}"
    echo ""
    if [ "$USE_HTTPS" == "yes" ]; then
        cat << EOF
  $PROXY_DOMAIN {
      handle /api/* {
          uri strip_prefix /api
          reverse_proxy 127.0.0.1:8080
      }

      handle {
          reverse_proxy 127.0.0.1:3000
      }
  }
EOF
    else
        cat << EOF
  :80 {
      handle /api/* {
          uri strip_prefix /api
          reverse_proxy 127.0.0.1:8080
      }

      handle {
          reverse_proxy 127.0.0.1:3000
      }
  }
EOF
    fi

    echo ""
    echo -e "${YELLOW}▸ Traefik Configuration (docker-compose labels)${NC}"
    echo ""
    cat << EOF
  # Add to orris service in docker-compose.yml:
  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.orris-api.rule=Host(\`$PROXY_DOMAIN\`) && PathPrefix(\`/api\`)"
    - "traefik.http.routers.orris-api.middlewares=strip-api"
    - "traefik.http.middlewares.strip-api.stripprefix.prefixes=/api"
    - "traefik.http.services.orris-api.loadbalancer.server.port=8080"

  # Add to frontend service in docker-compose.yml:
  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.orris-frontend.rule=Host(\`$PROXY_DOMAIN\`)"
    - "traefik.http.services.orris-frontend.loadbalancer.server.port=80"
EOF

    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    # Save config examples to files
    save_proxy_config_files "$PROXY_DOMAIN" "$USE_HTTPS"
}

save_proxy_config_files() {
    local PROXY_DOMAIN="$1"
    local USE_HTTPS="$2"

    mkdir -p ./proxy-examples

    # Save Nginx config
    if [ "$USE_HTTPS" == "yes" ]; then
        cat > ./proxy-examples/nginx.conf << EOF
server {
    listen 80;
    server_name $PROXY_DOMAIN;
    return 301 https://\$server_name\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name $PROXY_DOMAIN;

    ssl_certificate     /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # API routes -> backend
    location /api/ {
        rewrite ^/api/(.*)\$ /\$1 break;
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # All other routes -> frontend
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
    else
        cat > ./proxy-examples/nginx.conf << EOF
server {
    listen 80;
    server_name localhost;

    # API routes -> backend
    location /api/ {
        rewrite ^/api/(.*)\$ /\$1 break;
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # All other routes -> frontend
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
    fi

    # Save Caddy config
    if [ "$USE_HTTPS" == "yes" ]; then
        cat > ./proxy-examples/Caddyfile << EOF
$PROXY_DOMAIN {
    handle /api/* {
        uri strip_prefix /api
        reverse_proxy 127.0.0.1:8080
    }

    handle {
        reverse_proxy 127.0.0.1:3000
    }
}
EOF
    else
        cat > ./proxy-examples/Caddyfile << EOF
:80 {
    handle /api/* {
        uri strip_prefix /api
        reverse_proxy 127.0.0.1:8080
    }

    handle {
        reverse_proxy 127.0.0.1:3000
    }
}
EOF
    fi

    log_info "Proxy configuration examples saved to ./proxy-examples/"
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
        if [ "$USE_BUILTIN_CADDY" == "yes" ]; then
            echo -e "  URL:        ${BLUE}http://localhost${NC}"
        else
            echo -e "  Frontend:   ${BLUE}http://localhost:3000${NC}"
            echo -e "  Backend:    ${BLUE}http://localhost:8080${NC}"
        fi
    fi

    if [ -n "$ADMIN_EMAIL" ]; then
        echo -e "  Admin:      ${BLUE}$ADMIN_EMAIL${NC}"
    fi

    echo ""
    echo -e "  ${CYAN}Components:${NC}"
    echo "    Caddy:  $([ "$USE_BUILTIN_CADDY" == "yes" ] && echo "Built-in (Docker)" || echo "External")"
    echo "    MySQL:  $([ "$USE_BUILTIN_MYSQL" == "yes" ] && echo "Built-in (Docker)" || echo "External ($EXT_MYSQL_HOST)")"
    echo "    Redis:  $([ "$USE_BUILTIN_REDIS" == "yes" ] && echo "Built-in (Docker)" || echo "External ($EXT_REDIS_HOST)")"

    echo ""
    echo "  Useful commands:"
    echo "    cd $INSTALL_DIR"
    echo "    $DOCKER_COMPOSE ps        # View status"
    echo "    $DOCKER_COMPOSE logs -f   # View logs"
    echo "    $DOCKER_COMPOSE down      # Stop services"
    echo "    ./install.sh update       # Update to latest version"
    echo ""

    if [ "$USE_BUILTIN_CADDY" != "yes" ]; then
        log_warn "Remember to configure your reverse proxy!"
        echo ""
        print_reverse_proxy_examples
    fi

    if [ -n "$DOMAIN" ] && [ "$USE_BUILTIN_CADDY" == "yes" ]; then
        log_warn "Make sure your DNS is pointing to this server!"
        log_warn "Caddy will automatically obtain SSL certificates."
    fi

    echo ""
    log_info "For more information, visit: https://github.com/orris-inc/orris"
}

# Uninstall function - stops services and optionally removes data
do_uninstall() {
    echo -e "${BLUE}"
    echo "  ___  ____  ____  ___  ____  "
    echo " / _ \|  _ \|  _ \|_ _|/ ___| "
    echo "| | | | |_) | |_) || | \___ \ "
    echo "| |_| |  _ <|  _ < | |  ___) |"
    echo " \___/|_| \_\_| \_\___||____/ "
    echo -e "${NC}"
    echo -e "${RED}Orris Uninstall${NC}"
    echo ""

    # Check if we're in the right directory
    if [ ! -f "docker-compose.yml" ]; then
        log_error "docker-compose.yml not found in current directory."
        log_error "Please run this command from your Orris installation directory."
        exit 1
    fi

    # Check for docker compose
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
    elif command -v docker-compose &> /dev/null; then
        DOCKER_COMPOSE="docker-compose"
    else
        log_error "Docker Compose is not installed."
        exit 1
    fi

    echo -e "${YELLOW}WARNING: This will stop and remove all Orris services.${NC}"
    echo ""
    read -p "Are you sure you want to uninstall Orris? [y/N] " CONFIRM < /dev/tty
    if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
        log_info "Uninstall cancelled."
        exit 0
    fi

    echo ""
    log_info "Stopping services..."
    $DOCKER_COMPOSE down

    # Ask about removing volumes (Docker volumes)
    echo ""
    read -p "Remove Docker volumes (caddy_data, caddy_config)? [y/N] " REMOVE_VOLUMES < /dev/tty
    if [[ "$REMOVE_VOLUMES" =~ ^[Yy]$ ]]; then
        log_info "Removing Docker volumes..."
        $DOCKER_COMPOSE down -v
    fi

    # Ask about removing data directory
    echo ""
    echo -e "${YELLOW}Data directories contain:${NC}"
    echo "  - ./data/mysql  (database files)"
    echo "  - ./data/redis  (cache data)"
    echo ""
    read -p "Remove data directories? This will DELETE ALL DATA! [y/N] " REMOVE_DATA < /dev/tty
    if [[ "$REMOVE_DATA" =~ ^[Yy]$ ]]; then
        read -p "Type 'DELETE' to confirm data deletion: " CONFIRM_DELETE < /dev/tty
        if [ "$CONFIRM_DELETE" == "DELETE" ]; then
            log_info "Removing data directories..."
            rm -rf ./data/mysql ./data/redis
            log_info "Data directories removed."
        else
            log_info "Data deletion cancelled."
        fi
    fi

    # Ask about removing configuration files
    echo ""
    read -p "Remove configuration files (.env, docker-compose.yml, Caddyfile)? [y/N] " REMOVE_CONFIG < /dev/tty
    if [[ "$REMOVE_CONFIG" =~ ^[Yy]$ ]]; then
        log_info "Removing configuration files..."
        rm -f .env docker-compose.yml Caddyfile .env.example
        rm -rf configs
        log_info "Configuration files removed."
    fi

    # Ask about removing the entire installation directory
    CURRENT_DIR=$(pwd)
    echo ""
    read -p "Remove entire installation directory ($CURRENT_DIR)? [y/N] " REMOVE_ALL < /dev/tty
    if [[ "$REMOVE_ALL" =~ ^[Yy]$ ]]; then
        read -p "Type 'REMOVE' to confirm directory deletion: " CONFIRM_REMOVE < /dev/tty
        if [ "$CONFIRM_REMOVE" == "REMOVE" ]; then
            log_info "Removing installation directory..."
            cd ..
            rm -rf "$CURRENT_DIR"
            log_info "Installation directory removed."
        else
            log_info "Directory deletion cancelled."
        fi
    fi

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}   Uninstall Complete!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    log_info "Orris has been uninstalled."

    # Check if Docker images should be removed
    echo ""
    read -p "Remove Orris Docker images? [y/N] " REMOVE_IMAGES < /dev/tty
    if [[ "$REMOVE_IMAGES" =~ ^[Yy]$ ]]; then
        log_info "Removing Docker images..."
        docker rmi ghcr.io/orris-inc/orris:latest 2>/dev/null || true
        docker rmi ghcr.io/orris-inc/orris-frontend:latest 2>/dev/null || true
        docker rmi caddy:2-alpine 2>/dev/null || true
        docker rmi mysql:8.0 2>/dev/null || true
        docker rmi redis:7-alpine 2>/dev/null || true
        log_info "Docker images removed."
    fi
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
    select_install_mode
    prompt_config
    create_install_dir
    download_files
    generate_docker_compose
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
    echo "  install     Install Orris (default)"
    echo "  update      Update Orris to the latest version"
    echo "  uninstall   Uninstall Orris and optionally remove data"
    echo "  help        Show this help message"
    echo ""
    echo "Installation Modes:"
    echo "  Full:     All components included (Caddy, MySQL, Redis)"
    echo "  Custom:   Choose which components to use (external DB/Redis supported)"
    echo ""
    echo "Examples:"
    echo "  $0              # Install Orris"
    echo "  $0 install      # Install Orris"
    echo "  $0 update       # Update existing installation"
    echo "  $0 uninstall    # Uninstall Orris"
    echo ""
    echo "Remote usage:"
    echo "  curl -fsSL https://raw.githubusercontent.com/orris-inc/orris/main/install.sh | bash"
    echo "  curl -fsSL https://raw.githubusercontent.com/orris-inc/orris/main/install.sh | bash -s -- update"
    echo "  curl -fsSL https://raw.githubusercontent.com/orris-inc/orris/main/install.sh | bash -s -- uninstall"
}

# Main entry point
case "$ACTION" in
    install)
        do_install
        ;;
    update)
        do_update
        ;;
    uninstall|remove)
        do_uninstall
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
