#!/bin/bash

API_URL="http://localhost:8080/api"
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

TMP_RESP="/tmp/api_resp.txt"
cleanup() { rm -f "$TMP_RESP"; }
trap cleanup EXIT

log_err_and_exit() {
    local step_name="$1" expected_code="$2" actual_code="$3"
    echo -e "${RED}ОШИБКА${NC}"
    echo "--------------------------------------------------"
    echo "Сбой на этапе: $step_name | Ожидался: $expected_code | Получен: $actual_code"

    if [ "$actual_code" = "000" ]; then
        echo -e "${RED}[КРИТИЧЕСКИЙ СБОЙ] Сервер Блог-API недоступен по адресу $API_URL.${NC}"
        echo "Убедитесь, что приложение успешно запущено и слушает порт 8080 перед стартом тестов."
    else
        echo "Тело ответа сервера:"
        [ -f "$TMP_RESP" ] && cat "$TMP_RESP" || echo "[Файл ответа пуст]"
    fi
    echo "--------------------------------------------------"
    exit 1
}

echo "=== Запуск расширенной сквозной автопроверки Блог-API ==="

# 1. Health Check
HTTP_CODE=$(curl -s -o "$TMP_RESP" -w "%{http_code}" -X GET "$API_URL/health")
[ "$HTTP_CODE" -eq 200 ] || log_err_and_exit "Health Check" "200" "$HTTP_CODE"

RANDOM_ID=$((1 + RANDOM % 10000))
USERNAME_A="user_a_$RANDOM_ID" ; EMAIL_A="user_a_$RANDOM_ID@test.com" ; PASSWORD_A="Password123!"
USERNAME_B="user_b_$RANDOM_ID" ; EMAIL_B="user_b_$RANDOM_ID@test.com" ; PASSWORD_B="Password123!"

# 2. Регистрация и Вход Пользователя А
curl -s -o /dev/null -X POST "$API_URL/register" -H "Content-Type: application/json" -d "{\"username\":\"$USERNAME_A\",\"email\":\"$EMAIL_A\",\"password\":\"$PASSWORD_A\"}"
HTTP_CODE=$(curl -s -o "$TMP_RESP" -w "%{http_code}" -X POST "$API_URL/login" -H "Content-Type: application/json" -d "{\"email\":\"$EMAIL_A\",\"password\":\"$PASSWORD_A\"}")
TOKEN_A=$(grep -o '"token":"[^"]*' "$TMP_RESP" | head -n1 | grep -o '[^"]*' | tail -n1)

# 3. Регистрация и Вход Пользователя Б
curl -s -o /dev/null -X POST "$API_URL/register" -H "Content-Type: application/json" -d "{\"username\":\"$USERNAME_B\",\"email\":\"$EMAIL_B\",\"password\":\"$PASSWORD_B\"}"
HTTP_CODE=$(curl -s -o "$TMP_RESP" -w "%{http_code}" -X POST "$API_URL/login" -H "Content-Type: application/json" -d "{\"email\":\"$EMAIL_B\",\"password\":\"$PASSWORD_B\"}")
TOKEN_B=$(grep -o '"token":"[^"]*' "$TMP_RESP" | head -n1 | grep -o '[^"]*' | tail -n1)

# 4. Создание поста Пользователем А
echo -n "4. POST /posts (Пользователь А): "
HTTP_CODE=$(curl -s -o "$TMP_RESP" -w "%{http_code}" -X POST "$API_URL/posts" -H "Authorization: Bearer $TOKEN_A" -H "Content-Type: application/json" -d '{"title":"Оригинальный заголовок поста","content":"Контент публикации автора А"}')
[ "$HTTP_CODE" -eq 201 ] || log_err_and_exit "Создание поста" "201" "$HTTP_CODE"
POST_ID=$(grep -o '"id":[0-9]*' "$TMP_RESP" | head -n1 | grep -o '[0-9]*')
echo -e "${GREEN}УСПЕШНО (ID: $POST_ID)${NC}"

# 5. Обновление собственного поста Пользователем А
echo -n "5. PUT /posts/$POST_ID (Собственный пост Пользователя А): "
HTTP_CODE=$(curl -s -o "$TMP_RESP" -w "%{http_code}" -X PUT "$API_URL/posts/$POST_ID" -H "Authorization: Bearer $TOKEN_A" -H "Content-Type: application/json" -d '{"title":"Обновленный заголовок поста","content":"Обновленный контент публикации автора А"}')
[ "$HTTP_CODE" -eq 200 ] || log_err_and_exit "Обновление собственного поста" "200" "$HTTP_CODE"
echo -e "${GREEN}УСПЕШНО${NC}"

# 6. Запрет изменения чужого поста Пользователем Б
echo -n "6. PUT /posts/$POST_ID (Чужой пост от Пользователя Б): "
# ИСПРАВЛЕНО: title изменен на 5+ символов, контент изменен на 10+ символов для прохождения первичного валидатора
HTTP_CODE=$(curl -s -o "$TMP_RESP" -w "%{http_code}" -X PUT "$API_URL/posts/$POST_ID" -H "Authorization: Bearer $TOKEN_B" -H "Content-Type: application/json" -d '{"title":"Атака Хакера","content":"Попытка несанкционированного изменения чужого контента"}')
[ "$HTTP_CODE" -eq 403 ] || log_err_and_exit "Изменение чужого поста" "403" "$HTTP_CODE"
echo -e "${GREEN}УСПЕШНО (Заблокировано с кодом 403)${NC}"

# 7. Создание комментария Пользователем А к собственному посту
HTTP_CODE=$(curl -s -o "$TMP_RESP" -w "%{http_code}" -X POST "$API_URL/posts/$POST_ID/comments" -H "Authorization: Bearer $TOKEN_A" -H "Content-Type: application/json" -d '{"content":"Комментарий автора А"}')
COMMENT_ID=$(grep -o '"id":[0-9]*' "$TMP_RESP" | head -n1 | grep -o '[0-9]*')

# 8. Запрет изменения чужого комментария Пользователем Б
echo -n "8. PUT /comments/$COMMENT_ID (Чужой комментарий от Пользователя Б): "
HTTP_CODE=$(curl -s -o "$TMP_RESP" -w "%{http_code}" -X PUT "$API_URL/comments/$COMMENT_ID" -H "Authorization: Bearer $TOKEN_B" -H "Content-Type: application/json" -d '{"content":"Пытаюсь изменить чужой коммент"}')
[ "$HTTP_CODE" -eq 403 ] || log_err_and_exit "Изменение чужого комментария" "403" "$HTTP_CODE"
echo -e "${GREEN}УСПЕШНО (Заблокировано с кодом 403)${NC}"

# 9. Запрет удаления чужого комментария Пользователем Б
echo -n "9. DELETE /comments/$COMMENT_ID (Чужой комментарий от Пользователя Б): "
HTTP_CODE=$(curl -s -o "$TMP_RESP" -w "%{http_code}" -X DELETE "$API_URL/comments/$COMMENT_ID" -H "Authorization: Bearer $TOKEN_B")
[ "$HTTP_CODE" -eq 403 ] || log_err_and_exit "Удаление чужого комментария" "403" "$HTTP_CODE"
echo -e "${GREEN}УСПЕШНО (Заблокировано с кодом 403)${NC}"

# 10. Запрет удаления чужого поста Пользователем Б
echo -n "10. DELETE /posts/$POST_ID (Чужой пост от Пользователя Б): "
HTTP_CODE=$(curl -s -o "$TMP_RESP" -w "%{http_code}" -X DELETE "$API_URL/posts/$POST_ID" -H "Authorization: Bearer $TOKEN_B")
[ "$HTTP_CODE" -eq 403 ] || log_err_and_exit "Удаление чужого поста" "403" "$HTTP_CODE"
echo -e "${GREEN}УСПЕШНО (Заблокировано с кодом 403)${NC}"

# 11. Очистка ресурсов автором А
curl -s -o /dev/null -X DELETE "$API_URL/comments/$COMMENT_ID" -H "Authorization: Bearer $TOKEN_A"
curl -s -o /dev/null -X DELETE "$API_URL/posts/$POST_ID" -H "Authorization: Bearer $TOKEN_A"

echo -e "${GREEN}=== Все E2E сценарии тестирования успешно пройдены! ===${NC}"
