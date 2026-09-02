const API_BASE = '/api';

const clientAPI = {
    getToken() { return localStorage.getItem('jwt_token') || ''; },
    getUser() { return JSON.parse(localStorage.getItem('user_info')) || null; },

    saveSession(token, user) {
        localStorage.setItem('jwt_token', token);
        localStorage.setItem('user_info', JSON.stringify(user));
    },

    clearSession() {
        localStorage.removeItem('jwt_token');
        localStorage.removeItem('user_info');
    },

    async request(endpoint, method = 'GET', body = null) {
        const headers = { 'Content-Type': 'application/json' };
        const token = this.getToken();
        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        const config = { method, headers };
        if (body) {
            config.body = JSON.stringify(body);
        }

        try {
            const res = await fetch(API_BASE + endpoint, config);

            // Если сервер вернул 204 No Content
            if (res.status === 204) return null;

            const data = await res.json();
            if (!res.ok) {
                // Если база данных недоступна (500/503) — пробрасываем безопасный текст
                throw new Error(data.error || 'Произошла непредвиденная ошибка на сервере');
            }
            return data;
        } catch (err) {
            throw err;
        }
    }
};
