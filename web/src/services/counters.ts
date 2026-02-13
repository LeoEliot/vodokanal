import api from './api'
import type {
  Counter,
  CreateCounterRequest,
  PaginationParams,
  PaginatedResponse
} from '../types'

export const countersService = {
  async getCounters(params?: PaginationParams): Promise<PaginatedResponse<Counter>> {
    const response = await api.get<PaginatedResponse<Counter>>('/counters', { params })
    return response.data
  },

  async getCounter(id: string): Promise<Counter> {
    const response = await api.get<Counter>(`/counters/${id}`)
    return response.data
  },

  async getSubscriberCounters(subscriberId: string): Promise<Counter[]> {
    const response = await api.get<Counter[]>(`/counters/subscriber/${subscriberId}`)
    return response.data
  },

  async createCounter(data: CreateCounterRequest): Promise<Counter> {
    const response = await api.post<Counter>('/counters', data)
    return response.data
  },

  async updateCounter(id: string, data: Partial<CreateCounterRequest>): Promise<Counter> {
    const response = await api.put<Counter>(`/counters/${id}`, data)
    return response.data
  },

  async deleteCounter(id: string): Promise<void> {
    await api.delete(`/counters/${id}`)
  },

  async getExpiringCounters(days: number = 30): Promise<Counter[]> {
    const response = await api.get<Counter[]>('/counters/expiring', {
      params: { days }
    })
    return response.data
  },
}
