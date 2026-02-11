const USE_PROXY = import.meta.env.VITE_USE_PROXY === 'true'
const BASE_URL = USE_PROXY ? '' : import.meta.env.VITE_API_BASE_URL ?? ''
const TOKEN_KEY = 'auth_token'

export function getAuthToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setAuthToken(token?: string) {
  if (!token) {
    localStorage.removeItem(TOKEN_KEY)
    return
  }
  localStorage.setItem(TOKEN_KEY, token)
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getAuthToken()

  const response = await fetch(`${BASE_URL}${path}`, {
    credentials: 'include',
    ...options,
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(options.headers ?? {}),
    },
  })

  if (!response.ok) {
    const errorBody = await response.text()
    throw new Error(errorBody || `Request failed: ${response.status}`)
  }

  if (response.status === 204) {
    return undefined as T
  }

  const rawBody = await response.text()
  if (!rawBody) {
    return undefined as T
  }

  const contentType = response.headers.get('content-type') ?? ''
  if (contentType.includes('application/json')) {
    return JSON.parse(rawBody) as T
  }

  return rawBody as unknown as T
}

export function httpGet<T>(path: string, init?: RequestInit) {
  return request<T>(path, {
    ...init,
    method: 'GET',
  })
}

export function httpPost<T>(path: string, body?: unknown, init?: RequestInit) {
  const payload =
    body instanceof FormData || body instanceof URLSearchParams || typeof body === 'string'
      ? (body as BodyInit)
      : JSON.stringify(body ?? {})

  const defaultHeaders: HeadersInit = {}
  if (body instanceof URLSearchParams) {
    defaultHeaders['Content-Type'] = 'application/x-www-form-urlencoded'
  } else if (!(body instanceof FormData) && typeof body !== 'string') {
    defaultHeaders['Content-Type'] = 'application/json'
  }

  return request<T>(path, {
    ...init,
    method: 'POST',
    body: payload,
    headers: {
      ...defaultHeaders,
      ...(init?.headers ?? {}),
    },
  })
}

export function httpDelete<T>(path: string, init?: RequestInit) {
  return request<T>(path, {
    ...init,
    method: 'DELETE',
  })
}

