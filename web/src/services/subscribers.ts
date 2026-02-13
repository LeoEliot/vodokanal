import api from './api'
import type { Subscriber, CreateSubscriberRequest } from '@/types'

interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  limit: number
}

export const subscribersService = {
  async list(page = 1, limit = 20): Promise<PaginatedResponse<Subscriber>> {
    const response = await api.get<Subscriber[]>('/subscribers', {
      params: { page, limit },
    })
    return {
      data: response.data,
      total: response.data.length,
      page,
      limit,
    }
  },

  async get(id: number): Promise<Subscriber> {
    const response = await api.get<Subscriber>(`/subscribers/${id}`)
    return response.data
  },

  async getByAccountNumber(accountNumber: string): Promise<Subscriber> {
    const response = await api.get<Subscriber>(`/subscribers/account/${accountNumber}`)
    return response.data
  },

  async create(data: CreateSubscriberRequest): Promise<Subscriber> {
    const response = await api.post<Subscriber>('/subscribers', data)
    return response.data
  },

  async update(id: number, data: Partial<Subscriber>): Promise<Subscriber> {
    const response = await api.put<Subscriber>(`/subscribers/${id}`, data)
    return response.data
  },

  async delete(id: number): Promise<void> {
    await api.delete(`/subscribers/${id}`)
  },
}
