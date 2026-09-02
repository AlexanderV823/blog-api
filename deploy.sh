#!/bin/bash
# Скрипт автоматического развертывания Blog-API

echo "=== 1. Подготовка рабочей директории ==="
sudo mkdir -p /var/www && cd /var/www

echo "=== 2. Скачивание конфигураций с GitHub ==="
sudo curl -sSLO https://raw.githubusercontent.com/AlexanderV823/blog-api/main/docker-compose.yml
sudo curl -sSLO https://raw.githubusercontent.com/AlexanderV823/blog-api/main/.env.example

echo "=== 3. Инициализация .env и генерация JWT_SECRET ==="
sudo cp .env.example .env

# Генерируем случайный 32-байтовый шестнадцатеричный ключ
# и автоматически заменяем им дефолтный JWT_SECRET в файле .env
JWT_GEN=$(openssl rand -hex 32)
sudo sed -i "s/^JWT_SECRET=.*/JWT_SECRET=$JWT_GEN/" .env
echo "✓ Уникальный криптографический JWT_SECRET успешно сгенерирован и добавлен в .env"

echo "=== 4. Настройка специфических переменных ==="
echo "Сейчас откроется редактор. ОБЯЗАТЕЛЬНО измените:"
echo "1) DB_HOST=postgres (вместо 127.0.0.1)"
echo "2) DB_PASSWORD=ваш_надежный_пароль"
echo ""
read -p "Нажмите [Enter], чтобы открыть редактор nano..."

sudo nano .env

echo "=== 5. Запуск Docker Compose ==="
sudo docker compose up -d --build
