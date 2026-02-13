import api from './api'
import type { DashboardStats } from '../types'

export const dashboardService = {
  async getStats(): Promise<DashboardStats> {
    const response = await api.get<DashboardStats>('/stats/dashboard')
    return response.data
  },

  async getChart(period: 'week' | 'month' | 'year' = 'month'): Promise<{
    labels: string[]
    datasets: Array<{
      label: string
      data: number[]
      color?: string
    }>
  }> {
    const response = await api.get('/stats/chart', { params: { period } })
    return response.data
  },

  async getRecentActivity(limit: number = 10): Promise<Array<{
    id: string
    type: 'payment' | 'ticket' | 'reading' | 'bill'
    message: string
    created_at: string
  }>> {
    const response = await api.get('/stats/activity', { params: { limit } })
    return response.data
  },
}
