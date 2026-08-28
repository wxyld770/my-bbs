import type {
  ApiEnvelope,
  ApiMessage,
  ApiResponse,
  Comment,
  CreateCommentRequest,
  CreatePostRequest,
  HotPostData,
  InvitationData,
  LikeToggleData,
  LoginData,
  LoginRequest,
  PageData,
  PageQuery,
  PinPostData,
  PostDetail,
  PostListItem,
  PostVisibility,
  RegisterRequest,
  SearchData,
  SearchQuery,
  UpdatePostRequest,
  UpdateAvatarRequest,
  UpdateProfileRequest,
  UserProfileData,
} from '../types'

export const API_BASE_PATH = '/api'
export const TOKEN_STORAGE_KEY = 'my-bbs.token'
export const REQUEST_ID_HEADER = 'X-Request-ID'

const SESSION_AUTH_CODES = new Set([40100, 40101, 40102, 40103, 40104])

export type AuthFailureReason =
  | 'unauthorized'
  | 'invalid'
  | 'expired'
  | 'missing'
  | 'malformed'

export type QueryValue = string | number | boolean | null | undefined

export interface ApiRequestOptions
  extends Omit<RequestInit, 'body' | 'headers' | 'method'> {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  token?: string | null
  query?: Record<string, QueryValue>
  headers?: HeadersInit
  /** Reject a code=0 response when its data field is absent. */
  requireData?: boolean
}

interface ApiErrorOptions {
  status: number
  code: number
  requestId: string
  cause?: unknown
}

/** A public, serializable representation of an HTTP/API failure. */
export class ApiError extends Error {
  readonly status: number
  readonly code: number
  readonly requestId: string

  constructor(message: string, options: ApiErrorOptions) {
    super(message, options.cause === undefined ? undefined : { cause: options.cause })
    this.name = 'ApiError'
    this.status = options.status
    this.code = options.code
    this.requestId = options.requestId
  }

  get isNetworkError(): boolean {
    return this.status === 0
  }

  get isSessionAuthError(): boolean {
    return isSessionAuthError(this)
  }
}

export const tokenStore = {
  get(): string | null {
    try {
      return globalThis.localStorage?.getItem(TOKEN_STORAGE_KEY) ?? null
    } catch {
      return null
    }
  },

  set(token: string): void {
    try {
      globalThis.localStorage?.setItem(TOKEN_STORAGE_KEY, token)
    } catch {
      // Storage can be unavailable in privacy mode; callers still own the token.
    }
  },

  clear(): void {
    try {
      globalThis.localStorage?.removeItem(TOKEN_STORAGE_KEY)
    } catch {
      // Clearing an unavailable store is already the desired effective state.
    }
  },
}

export function isSessionAuthError(error: unknown): error is ApiError {
  return (
    error instanceof ApiError &&
    error.status === 401 &&
    SESSION_AUTH_CODES.has(error.code)
  )
}

/** Login failure 40105 deliberately does not count as an expired session. */
export function shouldClearToken(error: unknown): boolean {
  return isSessionAuthError(error)
}

export function getAuthFailureReason(
  error: unknown,
): AuthFailureReason | null {
  if (!isSessionAuthError(error)) return null

  switch (error.code) {
    case 40101:
      return 'invalid'
    case 40102:
      return 'expired'
    case 40103:
      return 'missing'
    case 40104:
      return 'malformed'
    default:
      return 'unauthorized'
  }
}

/**
 * Low-level JSON-envelope request. The URL is always kept same-origin; callers
 * pass API paths such as `/posts` rather than an absolute server URL.
 */
export async function apiRequest<T>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<ApiResponse<T>> {
  const requestId = createRequestId()
  const url = buildApiUrl(path, options.query)
  const headers = new Headers(options.headers)
  headers.set('Accept', 'application/json')
  if (!headers.has(REQUEST_ID_HEADER)) {
    headers.set(REQUEST_ID_HEADER, requestId)
  }

  const token = options.token?.trim()
  if (token) headers.set('Authorization', `Bearer ${token}`)

  let body: BodyInit | undefined
  if (options.body !== undefined) {
    headers.set('Content-Type', 'application/json')
    body = JSON.stringify(options.body)
  }

  let response: Response
  try {
    response = await fetch(url, {
      ...options,
      method: options.method ?? 'GET',
      headers,
      body,
      credentials: options.credentials ?? 'same-origin',
      query: undefined,
      token: undefined,
      requireData: undefined,
    } as RequestInit)
  } catch (cause) {
    const aborted = cause instanceof DOMException && cause.name === 'AbortError'
    throw new ApiError(aborted ? '请求已取消' : '网络连接失败，请稍后重试', {
      status: 0,
      code: aborted ? -3 : -1,
      requestId,
      cause,
    })
  }

  const responseRequestId = response.headers.get(REQUEST_ID_HEADER) || requestId
  const envelope = await readEnvelope<T>(response, responseRequestId)

  if (!response.ok || envelope.code !== 0) {
    throw new ApiError(envelope.message || defaultHttpMessage(response.status), {
      status: response.status,
      code: envelope.code,
      requestId: responseRequestId,
    })
  }

  if (options.requireData && !Object.hasOwn(envelope, 'data')) {
    throw new ApiError('服务器响应缺少 data 字段', {
      status: response.status,
      code: -2,
      requestId: responseRequestId,
    })
  }

  return {
    data: envelope.data as T,
    message: envelope.message,
    requestId: responseRequestId,
  }
}

async function requestData<T>(
  path: string,
  options?: ApiRequestOptions,
): Promise<T> {
  const response = await apiRequest<T>(path, { ...options, requireData: true })
  return response.data
}

async function requestMessage(
  path: string,
  options?: ApiRequestOptions,
): Promise<ApiMessage> {
  const response = await apiRequest<undefined>(path, options)
  return { message: response.message, requestId: response.requestId }
}

function pageQuery(query: PageQuery = {}): Record<string, QueryValue> {
  return { pageNo: query.pageNo, pageSize: query.pageSize }
}

export const api = {
  register(input: RegisterRequest): Promise<ApiMessage> {
    return requestMessage('/register', { method: 'POST', body: input })
  },

  login(input: LoginRequest): Promise<LoginData> {
    return requestData('/login', { method: 'POST', body: input })
  },

  logout(token: string): Promise<ApiMessage> {
    return requestMessage('/logout', { method: 'POST', token })
  },

  getMe(token: string): Promise<UserProfileData> {
    return requestData('/user/me', { token })
  },

  getUser(userId: number): Promise<UserProfileData> {
    return requestData(`/users/${encodeId(userId)}`)
  },

  updateProfile(
    token: string,
    input: UpdateProfileRequest,
  ): Promise<ApiMessage> {
    return requestMessage('/user/profile', {
      method: 'POST',
      token,
      body: input,
    })
  },

  updateAvatar(
    token: string,
    input: UpdateAvatarRequest,
  ): Promise<ApiMessage> {
    return requestMessage('/user/avatar', {
      method: 'POST',
      token,
      body: input,
    })
  },

  createInvitation(token: string): Promise<InvitationData> {
    return requestData('/invitations', {
      method: 'POST',
      token,
    })
  },

  muteUser(token: string, userId: number): Promise<ApiMessage> {
    return requestMessage(`/users/${encodeId(userId)}/mute`, {
      method: 'POST',
      token,
    })
  },

  unmuteUser(token: string, userId: number): Promise<ApiMessage> {
    return requestMessage(`/users/${encodeId(userId)}/unmute`, {
      method: 'POST',
      token,
    })
  },

  search(query: SearchQuery, signal?: AbortSignal): Promise<SearchData> {
    return requestData('/search', {
      signal,
      query: {
        q: query.q,
        scope: query.scope,
        pageNo: query.pageNo,
        pageSize: query.pageSize,
      },
    })
  },

  listPosts(query: PageQuery = {}, signal?: AbortSignal): Promise<PageData<PostListItem>> {
    return requestData('/posts', { query: pageQuery(query), signal })
  },

  listHotPosts(): Promise<HotPostData> {
    return requestData('/posts/hot')
  },

  listMyPosts(
    token: string,
    query: PageQuery = {},
  ): Promise<PageData<PostListItem>> {
    return requestData('/user/posts', {
      method: 'POST',
      token,
      query: pageQuery(query),
    })
  },

  listUserPosts(
    userId: number,
    query: PageQuery = {},
  ): Promise<PageData<PostListItem>> {
    return requestData(`/users/${encodeId(userId)}/posts`, {
      query: pageQuery(query),
    })
  },

  getPost(postId: number, token?: string | null): Promise<PostDetail> {
    return requestData(`/posts/${encodeId(postId)}`, { token })
  },

  createPost(token: string, input: CreatePostRequest): Promise<ApiMessage> {
    return requestMessage('/posts/create', {
      method: 'POST',
      token,
      body: input,
    })
  },

  updatePost(
    token: string,
    postId: number,
    input: UpdatePostRequest,
  ): Promise<ApiMessage> {
    return requestMessage(`/posts/update/${encodeId(postId)}`, {
      method: 'POST',
      token,
      body: input,
    })
  },

  deletePost(token: string, postId: number): Promise<ApiMessage> {
    return requestMessage(`/posts/del/${encodeId(postId)}`, {
      method: 'POST',
      token,
    })
  },

  pinPost(token: string, postId: number): Promise<PinPostData> {
    return requestData(`/posts/pin/${encodeId(postId)}`, {
      method: 'POST',
      token,
    })
  },

  unpinPost(token: string, postId: number): Promise<ApiMessage> {
    return requestMessage(`/posts/unpin/${encodeId(postId)}`, {
      method: 'POST',
      token,
    })
  },

  setPostVisibility(
    token: string,
    postId: number,
    visible: PostVisibility,
  ): Promise<ApiMessage> {
    return requestMessage(`/posts/visible/${encodeId(postId)}`, {
      method: 'POST',
      token,
      body: { visible },
    })
  },

  listComments(
    postId: number,
    query: PageQuery = {},
  ): Promise<PageData<Comment>> {
    return requestData(`/posts/${encodeId(postId)}/comments`, {
      query: pageQuery(query),
    })
  },

  createComment(
    token: string,
    postId: number,
    input: CreateCommentRequest,
  ): Promise<ApiMessage> {
    return requestMessage(`/posts/${encodeId(postId)}/comments/create`, {
      method: 'POST',
      token,
      body: input,
    })
  },

  deleteComment(token: string, commentId: number): Promise<ApiMessage> {
    return requestMessage(`/comments/del/${encodeId(commentId)}`, {
      method: 'POST',
      token,
    })
  },

  toggleLike(token: string, postId: number): Promise<LikeToggleData> {
    return requestData(`/posts/${encodeId(postId)}/like`, {
      method: 'POST',
      token,
    })
  },
}

function buildApiUrl(
  path: string,
  query?: Record<string, QueryValue>,
): string {
  if (/^[a-z][a-z\d+.-]*:/i.test(path) || path.startsWith('//')) {
    throw new TypeError('apiRequest 只接受相对 API 路径')
  }

  const normalizedPath = path.startsWith(API_BASE_PATH)
    ? path
    : `${API_BASE_PATH}/${path.replace(/^\/+/, '')}`
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(query ?? {})) {
    if (value !== undefined && value !== null) search.set(key, String(value))
  }
  const queryString = search.toString()
  return queryString ? `${normalizedPath}?${queryString}` : normalizedPath
}

function encodeId(value: number): string {
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new TypeError('资源 ID 必须是正整数')
  }
  return String(value)
}

async function readEnvelope<T>(
  response: Response,
  requestId: string,
): Promise<ApiEnvelope<T>> {
  let value: unknown
  try {
    value = await response.json()
  } catch (cause) {
    throw new ApiError('服务器返回了无法解析的响应', {
      status: response.status,
      code: -2,
      requestId,
      cause,
    })
  }

  if (!isEnvelope<T>(value)) {
    throw new ApiError('服务器响应格式不正确', {
      status: response.status,
      code: -2,
      requestId,
    })
  }
  return value
}

function isEnvelope<T>(value: unknown): value is ApiEnvelope<T> {
  if (typeof value !== 'object' || value === null) return false
  const candidate = value as Record<string, unknown>
  return typeof candidate.code === 'number' && typeof candidate.message === 'string'
}

function createRequestId(): string {
  try {
    return globalThis.crypto.randomUUID()
  } catch {
    return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`
  }
}

function defaultHttpMessage(status: number): string {
  if (status >= 500) return '服务器暂时不可用，请稍后重试'
  if (status === 404) return '请求的资源不存在'
  return '请求失败，请稍后重试'
}
