import { useEffect } from 'react'
import { Route, Routes, useLocation } from 'react-router-dom'
import { AuthDialog } from './components/AuthDialog'
import { ComposerDialog } from './components/ComposerDialog'
import { Shell } from './components/Shell'
import { ToastRegion } from './components/ToastRegion'
import { HomePage } from './pages/HomePage'
import { MePage } from './pages/MePage'
import { NotFoundPage } from './pages/NotFoundPage'
import { PostPage } from './pages/PostPage'
import { UserPage } from './pages/UserPage'

function ScrollToTop() {
  const { pathname } = useLocation()
  useEffect(() => {
    window.scrollTo({ top: 0, behavior: 'instant' })
  }, [pathname])
  return null
}

export default function App() {
  return (
    <>
      <ScrollToTop />
      <Routes>
        <Route element={<Shell />}>
          <Route index element={<HomePage />} />
          <Route path="post/:id" element={<PostPage />} />
          <Route path="me" element={<MePage />} />
          <Route path="u/:id" element={<UserPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
      <AuthDialog />
      <ComposerDialog />
      <ToastRegion />
    </>
  )
}
