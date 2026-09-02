let activePostId = null;
let isRegisterMode = false;

window.addEventListener('DOMContentLoaded', () => {
    renderAuthNav();
    showFeed();
});

function renderAuthNav() {
    const nav = document.getElementById('nav-auth');
    const token = clientAPI.getToken();
    const user = clientAPI.getUser();

    if (token && user) {
        document.getElementById('btn-new-post').classList.remove('hidden');
        nav.innerHTML = `
            <span style="font-size:0.9rem;">Вы: <b>${escapeHTML(user.username)}</b></span>
            <button onclick="handleLogout()" class="btn btn-secondary" style="padding:6px 12px;">Выйти</button>
        `;
    } else {
        document.getElementById('btn-new-post').classList.add('hidden');
        nav.innerHTML = `
            <button onclick="showAuth(false)" class="btn btn-secondary">Войти</button>
            <button onclick="showAuth(true)" class="btn btn-primary">Регистрация</button>
        `;
    }
}

function switchView(viewId) {
    ['view-feed', 'view-post', 'view-auth', 'view-create-post'].forEach(id => {
        document.getElementById(id).classList.add('hidden');
    });
    document.getElementById(viewId).classList.remove('hidden');
    document.getElementById('alert-zone').classList.add('hidden');
}

function showAlert(msg) {
    const zone = document.getElementById('alert-zone');
    zone.innerText = msg;
    zone.classList.remove('hidden');
}

async function showFeed() {
    switchView('view-feed');
    try {
        const data = await clientAPI.request('/posts?limit=50&offset=0');
        const container = document.getElementById('posts-container');
        container.innerHTML = data.posts && data.posts.length ? '' : '<p style="grid-column: 1/-1; text-align:center; color:gray;">Публикаций пока нет.</p>';

        if (data.posts) {
            data.posts.forEach(post => {
                const card = document.createElement('div');
                card.className = 'post-card';
                card.onclick = () => viewPostDetails(post.id);
                card.innerHTML = `
                    <div>
                        <h2>${escapeHTML(post.title)}</h2>
                        <p>${escapeHTML(post.content)}</p>
                    </div>
                    <div class="post-meta">
                        <span>Автор ID: ${post.author_id}</span>
                        <span>${new Date(post.created_at).toLocaleDateString('ru-RU')}</span>
                    </div>
                `;
                container.appendChild(card);
            });
        }
    } catch (err) {
        showAlert(err.message);
    }
}

async function viewPostDetails(postId) {
    activePostId = postId;
    switchView('view-post');
    try {
        const post = await clientAPI.request(`/posts/${postId}`);
        document.getElementById('post-content').innerHTML = `
            <h1>${escapeHTML(post.title)}</h1>
            <div style="font-size:0.8rem; color:gray;">Автор ID: ${post.author_id} | ${new Date(post.created_at).toLocaleString('ru-RU')}</div>
            <p>${escapeHTML(post.content)}</p>
        `;

        document.getElementById('comment-form-container').className = clientAPI.getToken() ? '' : 'hidden';
        loadComments(postId);
    } catch (err) {
        showAlert(err.message);
    }
}

async function loadComments(postId) {
    const container = document.getElementById('comments-container');
    container.innerHTML = '<small>Загрузка комментариев...</small>';
    try {
        const data = await clientAPI.request(`/posts/${postId}/comments?limit=100`);
        container.innerHTML = data.comments && data.comments.length ? '' : '<p style="color:gray; font-size:0.9rem;">Комментариев пока нет.</p>';

        if (data.comments) {
            const user = clientAPI.getUser();
            data.comments.forEach(c => {
                const div = document.createElement('div');
                div.className = 'comment-item';

                let deleteBtn = '';
                if (user && c.author_id === user.id) {
                    deleteBtn = `<a href="#" onclick="handleDeleteComment(${c.id})" style="color:#ef4444; font-size:0.8rem; text-decoration:none;">Удалить</a>`;
                }

                div.innerHTML = `
                    <div class="comment-header">
                        <span>ID автора: ${c.author_id}</span>
                        <span>${new Date(c.created_at).toLocaleDateString('ru-RU')} ${deleteBtn}</span>
                    </div>
                    <p>${escapeHTML(c.content)}</p>
                `;
                container.appendChild(div);
            });
        }
    } catch (err) {
        container.innerHTML = '<small style="color:red;">Не удалось загрузить комментарии</small>';
    }
}

function showAuth(regMode) {
    isRegisterMode = regMode;
    switchView('view-auth');
    document.getElementById('auth-title').innerText = isRegisterMode ? 'Регистрация' : 'Вход в аккаунт';
    document.getElementById('auth-submit-btn').innerText = isRegisterMode ? 'Создать профиль' : 'Войти';
    document.getElementById('auth-username-group').className = isRegisterMode ? '' : 'hidden';
    document.getElementById('auth-toggle-label').innerText = isRegisterMode ? 'Уже зарегистрированы?' : 'Еще нет аккаунта?';
    document.getElementById('auth-toggle-link').innerText = isRegisterMode ? 'Войти' : 'Зарегистрироваться';
}

function toggleAuthMode() { showAuth(!isRegisterMode); }

async function handleAuth(e) {
    e.preventDefault();
    const email = document.getElementById('auth-email').value;
    const password = document.getElementById('auth-password').value;
    const username = document.getElementById('auth-username').value;
    const endpoint = isRegisterMode ? '/register' : '/login';
    const body = isRegisterMode ? { username, email, password } : { email, password };

    try {
        const data = await clientAPI.request(endpoint, 'POST', body);
        clientAPI.saveSession(data.token, data.user);
        renderAuthNav();
        showFeed();
    } catch (err) {
        showAlert(err.message);
    }
}

function handleLogout() {
    clientAPI.clearSession();
    renderAuthNav();
    showFeed();
}

async function handleCreatePost(e) {
    e.preventDefault();
    const title = document.getElementById('post-title').value;
    const content = document.getElementById('post-body').value;
    try {
        await clientAPI.request('/posts', 'POST', { title, content });
        showFeed();
    } catch (err) {
        showAlert(err.message);
    }
}

async function handleCreateComment(e) {
    e.preventDefault();
    const inp = document.getElementById('input-comment');
    try {
        await clientAPI.request(`/posts/${activePostId}/comments`, 'POST', { content: inp.value });
        inp.value = '';
        loadComments(activePostId);
    } catch (err) {
        showAlert(err.message);
    }
}

async function handleDeleteComment(commentId) {
    if (!confirm('Удалить ваш комментарий?')) return;
    try {
        await clientAPI.request(`/comments/${commentId}`, 'DELETE');
        loadComments(activePostId);
    } catch (err) {
        showAlert(err.message);
    }
}

function escapeHTML(str) {
    if (!str) return '';
    return str.replace(/[&<>"']/g, m => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;' }[m]));
}
