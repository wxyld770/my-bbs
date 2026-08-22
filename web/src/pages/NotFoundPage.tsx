import { Compass } from 'lucide-react'
import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <div className="page-wrap">
      <div className="error-state">
        <div className="error-state__icon"><Compass size={24} aria-hidden="true" /></div>
        <h2>走到一条没有名字的小路</h2>
        <p>这个页面不存在，回广场看看大家正在聊什么吧。</p>
        <Link className="button button--dark" to="/">返回广场</Link>
      </div>
    </div>
  )
}
