import { useQuery } from '@tanstack/react-query'
import { dashboardService } from '../services/dashboard'
import { StatCard } from '../components/ui'

// Icons as components
const UsersIcon = () => (
  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
  </svg>
)

const DocumentIcon = () => (
  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
  </svg>
)

const CurrencyIcon = () => (
  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
)

const TicketIcon = () => (
  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z" />
  </svg>
)

const TrendUpIcon = () => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
  </svg>
)

const TrendDownIcon = () => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6" />
  </svg>
)

export const DashboardPage = () => {
  const { data: stats, isLoading } = useQuery({
    queryKey: ['dashboard', 'stats'],
    queryFn: () => dashboardService.getStats(),
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Панель управления</h1>
        <p className="text-gray-500 mt-1">Обзор системы Водоканал</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard
          title="Абонентов"
          value={stats?.subscribers.total || 0}
          change={stats?.subscribers.new_this_month ? ((stats.subscribers.new_this_month / stats.subscribers.total) * 100).toFixed(1) : undefined}
          icon={<UsersIcon />}
          color="blue"
        />
        <StatCard
          title="Показаний за месяц"
          value={stats?.readings.total_this_month || 0}
          icon={<DocumentIcon />}
          color="green"
        />
        <StatCard
          title="Счетов выставлено"
          value={stats?.bills.total_issued || 0}
          icon={<CurrencyIcon />}
          color="yellow"
        />
        <StatCard
          title="Открытых заявок"
          value={stats?.tickets.open || 0}
          icon={<TicketIcon />}
          color="purple"
        />
      </div>

      {/* Secondary Stats */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Bills Stats */}
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Счета</h3>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-gray-600">Оплачено</span>
              <span className="font-semibold text-green-600">{stats?.bills.total_paid || 0}</span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-2">
              <div
                className="bg-green-600 h-2 rounded-full"
                style={{
                  width: `${stats?.bills.total_issued ? (stats.bills.total_paid / stats.bills.total_issued) * 100 : 0}%`
                }}
              ></div>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-600">Просрочено</span>
              <span className="font-semibold text-red-600">{stats?.bills.overdue_count || 0}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-600">К оплате</span>
              <span className="font-semibold text-gray-900">{stats?.bills.total_outstanding || 0} ₽</span>
            </div>
          </div>
        </div>

        {/* Readings Stats */}
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Показания счетчиков</h3>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-gray-600">Проверено</span>
              <span className="font-semibold text-green-600 flex items-center gap-1">
                <TrendUpIcon />
                {stats?.readings.verified || 0}
              </span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-2">
              <div
                className="bg-blue-600 h-2 rounded-full"
                style={{
                  width: `${stats?.readings.total_this_month ? (stats.readings.verified / stats.readings.total_this_month) * 100 : 0}%`
                }}
              ></div>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-600">Ожидают проверки</span>
              <span className="font-semibold text-yellow-600 flex items-center gap-1">
                <TrendDownIcon />
                {stats?.readings.pending_verification || 0}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Tickets Stats */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Заявки</h3>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="bg-blue-50 rounded-lg p-4">
            <p className="text-sm text-blue-600 font-medium">Открыто</p>
            <p className="text-2xl font-bold text-blue-700 mt-1">{stats?.tickets.open || 0}</p>
          </div>
          <div className="bg-yellow-50 rounded-lg p-4">
            <p className="text-sm text-yellow-600 font-medium">В работе</p>
            <p className="text-2xl font-bold text-yellow-700 mt-1">{stats?.tickets.in_progress || 0}</p>
          </div>
          <div className="bg-green-50 rounded-lg p-4">
            <p className="text-sm text-green-600 font-medium">Решено сегодня</p>
            <p className="text-2xl font-bold text-green-700 mt-1">{stats?.tickets.resolved_today || 0}</p>
          </div>
        </div>
      </div>

      {/* Quick Links */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Быстрые действия</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <a
            href="/readings"
            className="flex flex-col items-center p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
          >
            <div className="text-blue-600"><DocumentIcon /></div>
            <span className="text-sm font-medium text-gray-700 mt-2">Добавить показания</span>
          </a>
          <a
            href="/bills"
            className="flex flex-col items-center p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
          >
            <div className="text-green-600"><CurrencyIcon /></div>
            <span className="text-sm font-medium text-gray-700 mt-2">Оплатить счет</span>
          </a>
          <a
            href="/tickets"
            className="flex flex-col items-center p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
          >
            <div className="text-purple-600"><TicketIcon /></div>
            <span className="text-sm font-medium text-gray-700 mt-2">Создать заявку</span>
          </a>
          <a
            href="/subscribers"
            className="flex flex-col items-center p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
          >
            <div className="text-yellow-600"><UsersIcon /></div>
            <span className="text-sm font-medium text-gray-700 mt-2">Абоненты</span>
          </a>
        </div>
      </div>
    </div>
  )
}
