import api from './api'
import type { Reading, CreateReadingRequest, ReadingSubmission, PaginationParams, PaginatedResponse } from '../types'

export const readingsService = {
  // Get all readings with pagination
  async getReadings(params?: PaginationParams): Promise<PaginatedResponse<Reading>> {
    const response = await api.get<PaginatedResponse<Reading>>('/readings', { params })
    return response.data
  },

  // Get reading by ID
  async getReading(id: string): Promise<Reading> {
    const response = await api.get<Reading>(`/readings/${id}`)
    return response.data
  },

  // Get readings by subscriber
  async getSubscriberReadings(subscriberId: string, params?: PaginationParams): Promise<Reading[]> {
    const response = await api.get<Reading[]>(`/readings/subscriber/${subscriberId}`, { params })
    return response.data
  },

  // Get readings by counter
  async getCounterReadings(counterId: string): Promise<Reading[]> {
    const response = await api.get<Reading[]>(`/readings/counter/${counterId}`)
    return response.data
  },

  // Create single reading
  async createReading(data: CreateReadingRequest): Promise<Reading> {
    const response = await api.post<Reading>('/readings', data)
    return response.data
  },

  // Submit readings for a subscriber (multiple counters)
  async submitReadings(data: ReadingSubmission): Promise<Reading[]> {
    const response = await api.post<Reading[]>('/readings/submit', data)
    return response.data
  },

  // Update reading
  async updateReading(id: string, data: Partial<CreateReadingRequest>): Promise<Reading> {
    const response = await api.put<Reading>(`/readings/${id}`, data)
    return response.data
  },

  // Verify reading
  async verifyReading(id: string): Promise<Reading> {
    const response = await api.post<Reading>(`/readings/${id}/verify`)
    return response.data
  },

  // Reject reading
  async rejectReading(id: string, reason: string): Promise<Reading> {
    const response = await api.post<Reading>(`/readings/${id}/reject`, { reason })
    return response.data
  },

  // Delete reading
  async deleteReading(id: string): Promise<void> {
    await api.delete(`/readings/${id}`)
  },

  // Get pending readings
  async getPendingReadings(): Promise<Reading[]> {
    const response = await api.get<Reading[]>('/readings/pending')
    return response.data
  },
}
