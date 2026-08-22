import { formatSummary } from './format'

const IMAGE_PATH_PATTERN = /\.(?:avif|gif|jpe?g|png|webp)$/iu
const MARKDOWN_IMAGE_PATTERN = /^!\[([^\]]*)\]\((https?:\/\/[^\s)]+)\)$/iu

export interface TextContentBlock {
  type: 'text'
  content: string
}

export interface ImageContentBlock {
  type: 'image'
  url: string
  alt: string
}

export type PostContentBlock = TextContentBlock | ImageContentBlock

export function isImageUrl(value: string): boolean {
  if (!value || value !== value.trim() || /\s/u.test(value)) return false

  try {
    const url = new URL(value)
    if (url.protocol !== 'https:' && url.protocol !== 'http:') return false
    if (!url.hostname || url.username || url.password) return false

    return IMAGE_PATH_PATTERN.test(url.pathname)
  } catch {
    return false
  }
}

export function parsePostContent(content: string): PostContentBlock[] {
  const blocks: PostContentBlock[] = []
  const textLines: string[] = []

  const flushText = () => {
    const text = trimBlankLines(textLines.join('\n'))
    textLines.length = 0
    if (text) blocks.push({ type: 'text', content: text })
  }

  for (const line of content.replace(/\r\n?/gu, '\n').split('\n')) {
    const image = parseImageLine(line)
    if (!image) {
      textLines.push(line)
      continue
    }
    flushText()
    blocks.push(image)
  }
  flushText()

  return blocks
}

export function getFirstPostImage(content: string): string | null {
  const image = parsePostContent(content).find(
    (block): block is ImageContentBlock => block.type === 'image',
  )
  return image?.url ?? null
}

export function getPostTextSummary(content: string, maxLength = 120): string {
  const blocks = parsePostContent(content)
  const text = blocks
    .filter((block): block is TextContentBlock => block.type === 'text')
    .map((block) => block.content)
    .join('\n')
  const summary = formatSummary(text, maxLength)
  if (summary) return summary

  const imageCount = blocks.filter((block) => block.type === 'image').length
  return imageCount > 1 ? `分享了 ${imageCount} 张图片` : '分享了一张图片'
}

function parseImageLine(line: string): ImageContentBlock | null {
  const trimmed = line.trim()
  if (!trimmed) return null

  const markdownMatch = trimmed.match(MARKDOWN_IMAGE_PATTERN)
  const url = markdownMatch?.[2] ?? trimmed
  if (!isImageUrl(url)) return null

  return {
    type: 'image',
    url,
    alt: sanitizeAlt(markdownMatch?.[1]) || '帖子图片',
  }
}

function sanitizeAlt(value?: string): string {
  if (!value) return ''
  return Array.from(value.replace(/[\u0000-\u001f\u007f]/gu, '').trim()).slice(0, 160).join('')
}

function trimBlankLines(value: string): string {
  return value.replace(/^(?:[ \t]*\n)+|(?:\n[ \t]*)+$/gu, '')
}
