import { Outlet, Link, useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/store/useAuthStore'

export function Layout() {
  const navigate = useNavigate()
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)

  const handleLogout = async () => {
    await useAuthStore.getState().logout()
    navigate('/login')
  }

  return (
    <div className="layout">
      <nav>
        <ul>
          <li><Link to="/dashboard">Главная</Link></li>
          <li><Link to="/subscribers">Абоненты</Link></li>
          <li><Link to="/readings">Показания</Link></li>
          <li><Link to="/bills">Счета</Link></li>
          <li><Link to="/tickets">Обращения</Link></li>
        </ul>
        <button onClick={handleLogout}>Выйти</button>
      </nav>
      <main>
        <Outlet />
      </main>
    </div>
  )
}
