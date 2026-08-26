/** A timestamp emitted by Go's time.Time JSON encoder. */
export type ISODateTime = string

export const USER_STATUS = {
  MUTED: 0,
  NORMAL: 1,
} as const

export type UserStatus = (typeof USER_STATUS)[keyof typeof USER_STATUS]

export const POST_VISIBILITY = {
  PRIVATE: 0,
  PUBLIC: 1,
} as const

export type PostVisibility =
  (typeof POST_VISIBILITY)[keyof typeof POST_VISIBILITY]

export interface User {
  id: number
  create_time: ISODateTime
  update_time: ISODateTime
  username: string
  nickname: string
  status: UserStatus
  introduction: string
  is_admin: boolean
}

export interface UserProfileData {
  user: User | null
}

export interface Post {
  id: number
  create_time: ISODateTime
  update_time: ISODateTime
  user_id: number
  title: string
  content: string
  visible: PostVisibility
  pinned_until: ISODateTime | null
  is_pinned: boolean
  user: User | null
}

export type PostListItem = Omit<Post, 'content'> & {
  like_count: number
  comment_count: number
}

export interface PostDetail {
  post: Post
  like_count: number
  comment_count: number
  is_liked: boolean
}

export interface Comment {
  id: number
  create_time: ISODateTime
  update_time: ISODateTime
  post_id: number
  user_id: number
  content: string
  user: User | null
}

export interface PageData<T> {
  list: T[]
  pageNo: number
  pageSize: number
  /**
   * This is a server-side hint, not an exact total. A full final page can still
   * report true, so consumers must also handle a subsequent empty page.
   */
  hasMore: boolean
}

export interface LoginData {
  token: string
}

export interface LikeToggleData {
  liked: boolean
  like_count: number
}

export interface PinPostData {
  pinned_until: ISODateTime
  is_pinned: boolean
}

/** The JSON envelope used by every endpoint except /livez and /readyz. */
export interface ApiEnvelope<T = unknown> {
  code: number
  message: string
  data?: T
}

export interface ApiResponse<T> {
  data: T
  message: string
  requestId: string
}

export interface ApiMessage {
  message: string
  requestId: string
}

export interface HealthData {
  status: 'ok' | 'unavailable'
}

export interface PageQuery {
  pageNo?: number
  pageSize?: number
}

export const SEARCH_SCOPE = {
  ALL: 'all',
  USERS: 'users',
  POSTS: 'posts',
} as const

export type SearchScope =
  (typeof SEARCH_SCOPE)[keyof typeof SEARCH_SCOPE]

export interface SearchQuery extends PageQuery {
  q: string
  scope?: SearchScope
}

export interface SearchUser {
  id: number
  username: string
  nickname: string
  introduction: string
}

export interface SearchPostAuthor {
  id: number
  username: string
  nickname: string
}

export interface SearchPost {
  id: number
  create_time: ISODateTime
  user_id: number
  title: string
  excerpt: string
  pinned_until: ISODateTime | null
  is_pinned: boolean
  user: SearchPostAuthor | null
  like_count: number
  comment_count: number
}

export interface SearchData {
  query: string
  scope: SearchScope
  users: PageData<SearchUser>
  posts: PageData<SearchPost>
}

export interface RegisterRequest {
  username: string
  password: string
  nickname?: string
}

export interface LoginRequest {
  username: string
  password: string
}

/**
 * The backend overwrites both profile columns on every update. Keeping both
 * properties required here prevents a partial form submission from erasing the
 * omitted value.
 */
export interface UpdateProfileRequest {
  nickname: string
  introduction: string
}

export interface CreatePostRequest {
  title: string
  content: string
}

/** The backend requires at least one non-null field for a post update. */
export type UpdatePostRequest =
  | { title: string; content?: string }
  | { title?: string; content: string }

export interface SetPostVisibilityRequest {
  visible: PostVisibility
}

export interface CreateCommentRequest {
  content: string
}

// Response-oriented aliases kept explicit at the transport boundary.
export type LoginResponse = LoginData
export type UserResponse = UserProfileData
export type LikeToggleResponse = LikeToggleData
export type PaginatedResponse<T> = PageData<T>
