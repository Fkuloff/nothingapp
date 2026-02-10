# Развертывание на сервере

## Быстрый старт

### 1. На сервере создайте директорию и скопируйте файлы

```bash
mkdir ~/messenger && cd ~/messenger
# Скопируйте сюда docker-compose.prod.yml и .env.production.example
```

### 2. Настройте .env

```bash
cp .env.production.example .env
nano .env
```

Обязательно измените:
- `POSTGRES_PASSWORD` - безопасный пароль
- `JWT_SECRET` - минимум 32 символа (генерация: `openssl rand -base64 32`)
- `ALLOWED_ORIGINS` - ваш домен
- `REGISTRY` - замените `your-username` на ваш GitHub username

### 3. Запустите

```bash
docker-compose up -d
```

## Управление

```bash
# Обновление
docker-compose pull && docker-compose up -d

# Логи
docker-compose logs -f

# Перезапуск
docker-compose restart

# Остановка
docker-compose down

# Backup БД
docker-compose exec postgres pg_dump -U messenger messenger > backup.sql

# Restore БД
cat backup.sql | docker-compose exec -T postgres psql -U messenger messenger
```

## SSL (опционально)

```bash
sudo apt install nginx certbot python3-certbot-nginx
```

Создайте `/etc/nginx/sites-available/messenger`:

```nginx
server {
    listen 80;
    server_name your-domain.com;
    client_max_body_size 500M;

    location / {
        proxy_pass http://localhost:80;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /ws {
        proxy_pass http://localhost:80/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/messenger /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d your-domain.com
```