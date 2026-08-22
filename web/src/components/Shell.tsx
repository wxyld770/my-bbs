import { Home, LogIn, PenLine, UserRound } from 'lucide-react'
import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { getDisplayName } from '../lib/format'
import { Avatar } from './Avatar'

export function Shell() {
  const { user, isAuthenticated } = useAuth()
  const { openAuth, openComposer } = useUI()

  const compose = () => {
    if (!isAuthenticated) {
      openAuth('login')
      return
    }
    openComposer()
  }

  return (
    <div className="app">
      <header className="site-header">
        <div className="site-header__inner">
          <NavLink className="brand" to="/" aria-label="野集首页">
            <span className="brand__mark" aria-hidden="true">野</span>
            <span className="brand__name">
              <strong>野集</strong>
              <small>YEJI COMMUNITY</small>
            </span>
          </NavLink>

          <nav className="main-nav" aria-label="主要导航">
            <NavLink className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`} to="/" end>
              <Home size={16} aria-hidden="true" />
              广场
            </NavLink>
            {isAuthenticated && (
              <NavLink className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`} to="/me">
                <UserRound size={16} aria-hidden="true" />
                我的
              </NavLink>
            )}
          </nav>

          <div className="site-header__actions">
            {isAuthenticated && user ? (
              <NavLink className="button button--soft button--small" to="/me" title={getDisplayName(user)}>
                <Avatar user={user} size="sm" />
                {getDisplayName(user)}
              </NavLink>
            ) : (
              <button className="button button--soft button--small" type="button" onClick={() => openAuth('login')}>
                <LogIn size={16} aria-hidden="true" />
                登录
              </button>
            )}
            <button className="button button--primary" type="button" onClick={compose}>
              <PenLine size={17} aria-hidden="true" />
              <span>写点什么</span>
            </button>
          </div>
        </div>
      </header>

      <main>
        <Outlet />
      </main>

      <nav className="mobile-nav" aria-label="移动端导航">
        <NavLink className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`} to="/" end>
          <Home size={17} aria-hidden="true" />
          广场
        </NavLink>
        <button className="nav-link" type="button" onClick={compose}>
          <PenLine size={17} aria-hidden="true" />
          发布
        </button>
        {isAuthenticated ? (
          <NavLink className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`} to="/me">
            <UserRound size={17} aria-hidden="true" />
            我的
          </NavLink>
        ) : (
          <button className="nav-link" type="button" onClick={() => openAuth('login')}>
            <LogIn size={17} aria-hidden="true" />
            登录
          </button>
        )}
      </nav>

      <footer className="site-footer">
        <div className="site-footer__inner">
          <p>认真表达，也认真倾听。愿每一次相遇都不被辜负。</p>
          <span>YEJI · OPEN SQUARE</span>
        </div>
      </footer>
    </div>
  )
}
