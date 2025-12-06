function initChat(chatID, userID) {
    const ws = new WebSocket(`ws://localhost:8080/ws/chat/${chatID}?user_id=${userID}`);
    const messagesDiv = document.getElementById('messages');
    const form = document.getElementById('chat-form');
    const input = document.getElementById('message');

    ws.onmessage = function(event) {
        const data = JSON.parse(event.data);
        // Игнорируем своё сообщение (чтобы избежать дубликата после optimistic update)
        if (data.userID === userID) {
            return;
        }
        const p = document.createElement('p');
        p.innerHTML = `<strong>Other:</strong> ${data.text}`;
        messagesDiv.appendChild(p);
        messagesDiv.scrollTop = messagesDiv.scrollHeight;
    };

    form.onsubmit = function(e) {
        e.preventDefault();
        if (input.value) {
            // Optimistic update: сразу добавляем своё сообщение
            const p = document.createElement('p');
            p.innerHTML = `<strong>You:</strong> ${input.value}`;
            messagesDiv.appendChild(p);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;

            // Отправляем текст (сервер добавит userID)
            ws.send(input.value);
            input.value = '';
        }
    };
}