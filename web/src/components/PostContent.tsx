import { ExternalLink, ImageOff } from 'lucide-react'
import { useMemo, useState } from 'react'
import { parsePostContent, type ImageContentBlock } from '../lib/postContent'

export function PostContent({ content }: { content: string }) {
  const blocks = useMemo(() => parsePostContent(content), [content])

  return (
    <div className="post-content">
      {blocks.map((block, index) =>
        block.type === 'image' ? (
          <PostImage block={block} key={`${block.url}-${index}`} />
        ) : (
          <div className="post-content__text" key={`text-${index}`}>{block.content}</div>
        ),
      )}
    </div>
  )
}

function PostImage({ block }: { block: ImageContentBlock }) {
  const [failed, setFailed] = useState(false)

  if (failed) {
    return (
      <a className="post-image-fallback" href={block.url} target="_blank" rel="noopener noreferrer nofollow">
        <ImageOff size={20} aria-hidden="true" />
        <span>
          图片加载失败，点击打开原图
          <small>{block.url}</small>
        </span>
        <ExternalLink size={17} aria-hidden="true" />
      </a>
    )
  }

  return (
    <figure className="post-image">
      <a href={block.url} target="_blank" rel="noopener noreferrer nofollow" aria-label="在新窗口查看原图">
        <img
          src={block.url}
          alt={block.alt}
          loading="lazy"
          decoding="async"
          referrerPolicy="no-referrer"
          onError={() => setFailed(true)}
        />
      </a>
      {block.alt !== '帖子图片' && <figcaption>{block.alt}</figcaption>}
    </figure>
  )
}
