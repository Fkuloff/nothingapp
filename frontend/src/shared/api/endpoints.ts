export const endpoints = {
  auth: {
    login: '/api/auth/login',
    register: '/api/auth/register',
    logout: '/api/auth/logout',
    me: '/api/auth/me',
  },
  profile: (userId?: number | string) => (userId ? `/api/profile/${userId}` : '/api/profile'),
  chats: {
    list: '/api/chats',
    create: '/api/chats',
    messages: (chatId: number | string) => `/api/chats/${chatId}/messages`,
  },
  contacts: {
    list: '/api/contacts',
    add: (userId: number | string) => `/api/contacts/${userId}`,
    remove: (userId: number | string) => `/api/contacts/${userId}`,
  },
  attachments: {
    upload: (chatId: number | string, messageId: number | string) =>
      `/api/chats/${chatId}/messages/${messageId}/attachments`,
    get: (id: number | string) => `/api/attachments/${id}`,
    remove: (id: number | string) => `/api/attachments/${id}`,
  },
  avatar: {
    upload: '/api/user/avatar',
    remove: '/api/user/avatar',
    get: (userId: number | string) => `/api/avatars/${userId}`,
  },
  presence: {
    get: (userId: number | string) => `/api/presence/${userId}`,
  },
  ws: {
    global: '/ws',
  },
}
