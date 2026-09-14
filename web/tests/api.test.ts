import { describe, expect, it, vi } from 'vitest'
import {
  ApiError,
  DEFAULT_API_TIMEOUT_MS,
  api,
  apiRequest,
} from '../src/lib/api'

function pendingFetch(): typeof fetch {
  return vi.fn((_input: RequestInfo | URL, init?: RequestInit) => new Promise<Response>((_resolve, reject) => {
    const signal = init?.signal
    const rejectAbort = () => reject(new DOMException('aborted', 'AbortError'))
    if (signal?.aborted) {
      rejectAbort()
      return
    }
    signal?.addEventListener('abort', rejectAbort, { once: true })
  }))
}

function jsonResponse(data: unknown): Response {
  return {
    ok: true,
    status: 200,
    headers: new Headers(),
    json: async () => ({ code: 0, message: 'ok', data }),
  } as Response
}

describe('apiRequest request lifecycle', () => {
  it('aborts every request at the shared default timeout and reports a timeout', async () => {
    vi.useFakeTimers()
    vi.stubGlobal('fetch', pendingFetch())

    const assertion = expect(apiRequest('/slow')).rejects.toMatchObject({
      name: 'ApiError',
      code: -4,
      message: '请求超时，请稍后重试',
      isNetworkError: false,
      isTimeoutError: true,
      isCancelledError: false,
    } satisfies Partial<ApiError>)

    await vi.advanceTimersByTimeAsync(DEFAULT_API_TIMEOUT_MS)
    await assertion
  })

  it('keeps caller cancellation distinct from a timeout', async () => {
    vi.useFakeTimers()
    vi.stubGlobal('fetch', pendingFetch())
    const controller = new AbortController()

    const assertion = expect(apiRequest('/cancelled', {
      signal: controller.signal,
    })).rejects.toMatchObject({
      name: 'ApiError',
      code: -3,
      message: '请求已取消',
      isNetworkError: false,
      isTimeoutError: false,
      isCancelledError: true,
    } satisfies Partial<ApiError>)

    controller.abort()
    await assertion
  })

  it('reports transport failures separately from cancellation and timeout', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('connection failed')))

    await expect(apiRequest('/unavailable')).rejects.toMatchObject({
      name: 'ApiError',
      code: -1,
      message: '网络连接失败，请稍后重试',
      isNetworkError: true,
      isTimeoutError: false,
      isCancelledError: false,
    } satisfies Partial<ApiError>)
  })

  it('uses idempotent methods for desired like state', async () => {
    const fetchMock = vi.fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse({ liked: true, like_count: 4 }))
      .mockResolvedValueOnce(jsonResponse({ liked: false, like_count: 3 }))
    vi.stubGlobal('fetch', fetchMock)

    await api.likePost('token', 7)
    await api.unlikePost('token', 7)

    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/posts/7/like', expect.objectContaining({ method: 'PUT' }))
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/posts/7/like', expect.objectContaining({ method: 'DELETE' }))
  })
})
