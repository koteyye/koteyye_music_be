#!/bin/bash

# Koteyye Music SSL Initialization Script
# This script sets up SSL certificates for production deployment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DOMAINS=(music.kotey-ye.ru kotey-ye.ru)
EMAIL="admin@kotey-ye.ru"  # Change this to your email
RSA_KEY_SIZE=4096
DATA_PATH="./certbot"
NGINX_CONF_PATH="./nginx/conf.d"

echo -e "${BLUE}🚀 Starting SSL initialization for Koteyye Music...${NC}"

# Check if email is provided
if [[ "$EMAIL" == "admin@kotey-ye.ru" ]]; then
    echo -e "${RED}❌ Error: Please change the email address in the script!${NC}"
    echo -e "${YELLOW}   Edit the EMAIL variable in this script to your real email address.${NC}"
    exit 1
fi

# Check if running as root
if [[ $EUID -eq 0 ]]; then
    echo -e "${YELLOW}⚠️  Warning: Running as root. Consider using a non-root user with sudo privileges.${NC}"
fi

# Function to print status
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if docker and docker-compose are installed
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    print_error "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Determine docker-compose command
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
else
    DOCKER_COMPOSE="docker compose"
fi

print_status "Using: $DOCKER_COMPOSE"

# Create necessary directories
echo -e "${BLUE}📁 Creating directory structure...${NC}"
mkdir -p "$DATA_PATH/conf"
mkdir -p "$DATA_PATH/www"
mkdir -p "$NGINX_CONF_PATH"

# Download recommended TLS parameters
echo -e "${BLUE}📜 Downloading recommended TLS parameters...${NC}"
if [ ! -f "$DATA_PATH/conf/options-ssl-nginx.conf" ]; then
    curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot-nginx/certbot_nginx/_internal/tls_configs/options-ssl-nginx.conf > "$DATA_PATH/conf/options-ssl-nginx.conf"
    print_status "Downloaded options-ssl-nginx.conf"
fi

if [ ! -f "$DATA_PATH/conf/ssl-dhparams.pem" ]; then
    curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot/certbot/ssl-dhparams.pem > "$DATA_PATH/conf/ssl-dhparams.pem"
    print_status "Downloaded ssl-dhparams.pem"
fi

# Create dummy certificates for initial nginx startup
echo -e "${BLUE}🔧 Creating dummy certificates...${NC}"
for domain in "${DOMAINS[@]}"; do
    path="/etc/letsencrypt/live/$domain"
    mkdir -p "$DATA_PATH/conf/live/$domain"
    
    if [ ! -f "$DATA_PATH/conf/live/$domain/fullchain.pem" ]; then
        echo -e "${YELLOW}📜 Creating dummy certificate for $domain${NC}"
        docker run --rm -v "$PWD/$DATA_PATH/conf:/etc/letsencrypt" \
            certbot/certbot \
            bash -c "
                openssl req -x509 -nodes -newkey rsa:$RSA_KEY_SIZE -days 1 \
                    -keyout '$path/privkey.pem' \
                    -out '$path/fullchain.pem' \
                    -subj '/CN=localhost'
            "
        print_status "Created dummy certificate for $domain"
    fi
done

# Start nginx to validate configuration
echo -e "${BLUE}🔄 Starting nginx with dummy certificates...${NC}"
$DOCKER_COMPOSE -f docker-compose.prod.yml up -d nginx

# Wait for nginx to start
sleep 5

# Check if nginx started successfully
if ! docker ps | grep -q koteyye_nginx; then
    print_error "Nginx failed to start. Please check the configuration."
    $DOCKER_COMPOSE -f docker-compose.prod.yml logs nginx
    exit 1
fi

print_status "Nginx started successfully with dummy certificates"

# Request real certificates
echo -e "${BLUE}🔐 Requesting real SSL certificates...${NC}"
for domain in "${DOMAINS[@]}"; do
    echo -e "${YELLOW}📜 Requesting certificate for $domain...${NC}"
    
    # Remove dummy certificate
    docker run --rm -v "$PWD/$DATA_PATH/conf:/etc/letsencrypt" \
        certbot/certbot \
        bash -c "rm -rf /etc/letsencrypt/live/$domain"
    
    # Request real certificate
    docker run --rm -v "$PWD/$DATA_PATH/conf:/etc/letsencrypt" \
        -v "$PWD/$DATA_PATH/www:/var/www/certbot" \
        certbot/certbot \
        certonly \
            --webroot \
            --webroot-path=/var/www/certbot \
            --email "$EMAIL" \
            --agree-tos \
            --no-eff-email \
            --force-renewal \
            -d "$domain"
    
    if [ $? -eq 0 ]; then
        print_status "Certificate obtained for $domain"
    else
        print_error "Failed to obtain certificate for $domain"
        exit 1
    fi
done

# Reload nginx with real certificates
echo -e "${BLUE}🔄 Reloading nginx with real certificates...${NC}"
$DOCKER_COMPOSE -f docker-compose.prod.yml exec nginx nginx -s reload

if [ $? -eq 0 ]; then
    print_status "Nginx reloaded successfully"
else
    print_error "Failed to reload nginx"
    exit 1
fi

# Final verification
echo -e "${BLUE}🔍 Verifying SSL setup...${NC}"
for domain in "${DOMAINS[@]}"; do
    if curl -sSf "https://$domain" > /dev/null 2>&1; then
        print_status "SSL verification successful for $domain"
    else
        print_warning "SSL verification failed for $domain (this might be normal if DNS is not propagated yet)"
    fi
done

echo ""
echo -e "${GREEN}🎉 SSL initialization completed successfully!${NC}"
echo ""
echo -e "${BLUE}Next steps:${NC}"
echo "1. Make sure your DNS records point to this server:"
for domain in "${DOMAINS[@]}"; do
    echo "   - $domain → $(curl -s ifconfig.me)"
done
echo "2. Test your domains:"
for domain in "${DOMAINS[@]}"; do
    echo "   - https://$domain"
done
echo "3. Certificates will auto-renew every 12 hours"
echo ""
echo -e "${YELLOW}📝 Note: If DNS is not propagated yet, SSL verification might fail.${NC}"
echo -e "${YELLOW}   Wait for DNS propagation and run this script again if needed.${NC}"