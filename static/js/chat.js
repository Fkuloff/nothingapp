// static/js/chat.js
function initChat(chatID, userID, otherUsername) {
    const ws = new WebSocket(`ws://localhost:8080/ws/chat/${chatID}?user_id=${userID}`);
    const messagesDiv = document.getElementById('messages');
    const form = document.getElementById('chat-form');
    const input = document.getElementById('message');
    const replyPreview = document.getElementById('reply-preview');
    const cancelBtn = document.getElementById('cancel-reply');
    let replyToID = null;

    // Event delegation для всех reply-кнопок (initial + новые)
    messagesDiv.addEventListener('click', function(e) {
        if (e.target.classList.contains('reply-btn')) {
            const id = parseInt(e.target.dataset.msgId, 10);
            const text = e.target.dataset.text || '[No text]';
            if (!isNaN(id)) {
                setReply(id, text);
            } else {
                console.error('Invalid reply ID:', e.target.dataset.msgId);
            }
        }
    });

    function escapeJS(text) {
        // Полный эскейпинг для data-атрибутов (не для innerHTML)
        return text
            .replace(/\\/g, '\\\\')
            .replace(/"/g, '\\"')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/\n/g, '&#10;');
    }

    function setReply(id, text) {
        replyToID = id;
        replyPreview.innerHTML = `Replying to: ${text.substring(0, 50)}...`;
        replyPreview.style.display = 'block';
        cancelBtn.style.display = 'inline-block';
    }

    // Сделать cancelReply global для onclick в шаблоне
    window.cancelReply = function() {
        replyToID = null;
        replyPreview.style.display = 'none';
        cancelBtn.style.display = 'none';
    };

    ws.onmessage = function(event) {
        try {
            const data = JSON.parse(event.data);
            const className = (data.userID === userID) ? 'message-self' : 'message-other';
            const sender = (data.userID === userID) ? 'You' : otherUsername;
            appendMessage(data.text, className, getCurrentTime(), sender, data.replyToID, data.id);
        } catch (err) {
            console.error('Parse error:', err);
        }
    };

    ws.onerror = function(err) {
        console.error('WebSocket error:', err);
    };

    ws.onclose = function() {
        console.log('WebSocket closed');
    };

    form.onsubmit = function(e) {
        e.preventDefault();
        const text = input.value.trim();
        if (text === '') return;

        const msgData = { text: text, reply_to_id: replyToID || 0 };
        ws.send(JSON.stringify(msgData));

        input.value = '';
        window.cancelReply();  // Сброс (global)
    };

    function appendMessage(text, className, time, sender, replyToID, msgId) {
        const div = document.createElement('div');
        div.className = 'message ' + className;
        div.dataset.text = text;
        div.id = 'msg-' + msgId;

        let innerHTML = '';
        if (replyToID && replyToID > 0) {
            const replyElem = document.getElementById('msg-' + replyToID);
            let replyText = '[Message ' + replyToID + ']';
            if (replyElem) {
                replyText = replyElem.dataset.text.substring(0, 50);
            }
            innerHTML += `<div class="reply-preview">Replying to: ${replyText}...</div>`;
        }

        innerHTML += `<span class="message-sender">${sender}:</span> ${text}<div class="message-time">${time}</div>`;
        // Кнопка без onclick — delegation обработает
        innerHTML += `<button class="reply-btn" data-msg-id="${msgId}" data-text="${escapeJS(text)}">Reply</button>`;

        div.innerHTML = innerHTML;
        messagesDiv.appendChild(div);
        messagesDiv.scrollTop = messagesDiv.scrollHeight;
    }

    function getCurrentTime() {
        const now = new Date();
        return now.getHours().toString().padStart(2, '0') + ':' + now.getMinutes().toString().padStart(2, '0');
    }
}