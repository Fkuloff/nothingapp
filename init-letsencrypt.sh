#!/bin/bash

# init-letsencrypt.sh
# Script to obtain initial SSL certificates from Let's Encrypt
# Run this ONCE before starting the full docker-compose stack

set -e

# Configuration
domains=(nothingapp.ru www.nothingapp.ru minio.nothingapp.ru minio-console.nothingapp.ru)
email="${ACME_EMAIL:-}" # Set your email here or via environment variable
staging=0 # Set to 1 for testing to avoid rate limits
data_path="./certbot"
rsa_key_size=4096

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Let's Encrypt Certificate Initialization ===${NC}"
echo ""

# Check if email is set
if [ -z "$email" ]; then
    echo -e "${RED}Error: Email is required for Let's Encrypt registration.${NC}"
    echo "Set ACME_EMAIL environment variable or edit this script."
    echo "Example: ACME_EMAIL=your@email.com ./init-letsencrypt.sh"
    exit 1
fi

# Check if certificates already exist
if [ -d "$data_path/conf/live/${domains[0]}" ]; then
    echo -e "${YELLOW}Existing certificates found for ${domains[0]}.${NC}"
    read -p "Do you want to replace them? (y/N) " decision
    if [ "$decision" != "Y" ] && [ "$decision" != "y" ]; then
        echo "Keeping existing certificates."
        exit 0
    fi
fi

# Create directories
echo "Creating certificate directories..."
mkdir -p "$data_path/conf/live/${domains[0]}"
mkdir -p "$data_path/www"

# Download recommended TLS parameters
if [ ! -e "$data_path/conf/options-ssl-nginx.conf" ]; then
    echo "Downloading recommended TLS parameters..."
    curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot-nginx/certbot_nginx/_internal/tls_configs/options-ssl-nginx.conf > "$data_path/conf/options-ssl-nginx.conf"
fi

if [ ! -e "$data_path/conf/ssl-dhparams.pem" ]; then
    echo "Downloading DH parameters..."
    curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot/certbot/ssl-dhparams.pem > "$data_path/conf/ssl-dhparams.pem"
fi

# Create dummy certificate for nginx to start
echo "Creating dummy certificate for ${domains[0]}..."
openssl req -x509 -nodes -newkey rsa:$rsa_key_size -days 1 \
    -keyout "$data_path/conf/live/${domains[0]}/privkey.pem" \
    -out "$data_path/conf/live/${domains[0]}/fullchain.pem" \
    -subj "/CN=localhost" 2>/dev/null

echo ""
echo -e "${GREEN}Dummy certificate created successfully!${NC}"
echo ""
echo "Now starting nginx with dummy certificate..."

# Start nginx with dummy certificate
docker compose -f docker-compose.prod.yml up -d nginx-proxy

echo "Waiting for nginx to start..."
sleep 5

# Delete dummy certificate
echo "Deleting dummy certificate..."
rm -rf "$data_path/conf/live/${domains[0]}"

# Request real certificate
echo ""
echo -e "${GREEN}Requesting Let's Encrypt certificate...${NC}"
echo "Domains: ${domains[*]}"
echo "Email: $email"
echo ""

# Build domain arguments
domain_args=""
for domain in "${domains[@]}"; do
    domain_args="$domain_args -d $domain"
done

# Select staging or production
if [ $staging != "0" ]; then
    staging_arg="--staging"
    echo -e "${YELLOW}Using staging server (for testing)${NC}"
else
    staging_arg=""
    echo "Using production server"
fi

# Run certbot
docker compose -f docker-compose.prod.yml run --rm certbot certonly \
    --webroot \
    --webroot-path=/var/www/certbot \
    $staging_arg \
    --email $email \
    --agree-tos \
    --no-eff-email \
    --force-renewal \
    $domain_args

echo ""
echo -e "${GREEN}=== Certificate obtained successfully! ===${NC}"
echo ""
echo "Reloading nginx..."
docker compose -f docker-compose.prod.yml exec nginx-proxy nginx -s reload

echo ""
echo -e "${GREEN}=== Setup Complete! ===${NC}"
echo ""
echo "Your certificates are stored in: $data_path/conf/live/${domains[0]}/"
echo ""
echo "To start the full stack:"
echo "  docker compose -f docker-compose.prod.yml up -d"
echo ""
echo "Certificates will be automatically renewed by the certbot service."
