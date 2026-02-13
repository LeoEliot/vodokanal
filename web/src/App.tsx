import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from './components/Layout'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { DashboardPage } from './pages/DashboardPage'
import { SubscribersPage } from './pages/SubscribersPage'
import { ReadingsPage } from './pages/ReadingsPage'
import { BillsPage } from './pages/BillsPage'
import { TicketsPage } from './pages/TicketsPage'

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route path="/" element={<Layout />}>
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<DashboardPage />} />
        <Route path="subscribers" element={<SubscribersPage />} />
        <Route path="readings" element={<ReadingsPage />} />
        <Route path="bills" element={<BillsPage />} />
        <Route path="tickets" element={<TicketsPage />} />
      </Route>
    </Routes>
  )
}

export default App
