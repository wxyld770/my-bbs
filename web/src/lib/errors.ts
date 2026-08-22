import { ApiError } from './api'

export function getErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status >= 500 && error.requestId) {
      return `${error.message}（请求编号：${error.requestId}）`
    }
    return error.message
  }
  if (error instanceof Error && error.message) return error.message
  return '操作失败，请稍后重试'
}
