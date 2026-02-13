import api from './api'
import type {
  Ticket,
  CreateTicketRequest,
  Comment,
  Employee,
  PaginationParams,
  PaginatedResponse
} from '../types'

export const ticketsService = {
  // Tickets
  async getTickets(params?: PaginationParams & {
    status?: string
    priority?: string
    category?: string
    assigned_to?: string
  }): Promise<PaginatedResponse<Ticket>> {
    const response = await api.get<PaginatedResponse<Ticket>>('/tickets', { params })
    return response.data
  },

  async getTicket(id: string): Promise<Ticket> {
    const response = await api.get<Ticket>(`/tickets/${id}`)
    return response.data
  },

  async createTicket(data: CreateTicketRequest): Promise<Ticket> {
    const response = await api.post<Ticket>('/tickets', data)
    return response.data
  },

  async updateTicket(id: string, data: Partial<Ticket>): Promise<Ticket> {
    const response = await api.put<Ticket>(`/tickets/${id}`, data)
    return response.data
  },

  async deleteTicket(id: string): Promise<void> {
    await api.delete(`/tickets/${id}`)
  },

  async closeTicket(id: string, data: { resolution?: string; rating?: number; feedback?: string }): Promise<Ticket> {
    const response = await api.post<Ticket>(`/tickets/${id}/close`, data)
    return response.data
  },

  async reopenTicket(id: string): Promise<Ticket> {
    const response = await api.post<Ticket>(`/tickets/${id}/reopen`)
    return response.data
  },

  async assignTicket(id: string, employeeId: string): Promise<Ticket> {
    const response = await api.post<Ticket>(`/tickets/${id}/assign`, { employee_id: employeeId })
    return response.data
  },

  async getSubscriberTickets(subscriberId: string): Promise<Ticket[]> {
    const response = await api.get<Ticket[]>(`/subscribers/${subscriberId}/tickets`)
    return response.data
  },

  // Comments
  async getTicketComments(ticketId: string): Promise<Comment[]> {
    const response = await api.get<Comment[]>(`/tickets/${ticketId}/comments`)
    return response.data
  },

  async addComment(ticketId: string, content: string): Promise<Comment> {
    const response = await api.post<Comment>(`/tickets/${ticketId}/comments`, { content })
    return response.data
  },

  async updateComment(id: string, content: string): Promise<Comment> {
    const response = await api.put<Comment>(`/comments/${id}`, { content })
    return response.data
  },

  async deleteComment(id: string): Promise<void> {
    await api.delete(`/comments/${id}`)
  },

  // History
  async getTicketHistory(ticketId: string): Promise<any[]> {
    const response = await api.get<any[]>(`/tickets/${ticketId}/history`)
    return response.data
  },

  // Employees
  async getEmployees(): Promise<Employee[]> {
    const response = await api.get<Employee[]>('/employees')
    return response.data
  },

  async getEmployee(id: string): Promise<Employee> {
    const response = await api.get<Employee>(`/employees/${id}`)
    return response.data
  },

  async createEmployee(data: Partial<Employee>): Promise<Employee> {
    const response = await api.post<Employee>('/employees', data)
    return response.data
  },

  async updateEmployee(id: string, data: Partial<Employee>): Promise<Employee> {
    const response = await api.put<Employee>(`/employees/${id}`, data)
    return response.data
  },

  async deleteEmployee(id: string): Promise<void> {
    await api.delete(`/employees/${id}`)
  },

  // Statistics
  async getTicketsStats(): Promise<{
    by_status: Record<string, number>
    by_priority: Record<string, number>
  }> {
    const response = await api.get('/stats/tickets')
    return response.data
  },

  async getEmployeesStats(): Promise<Array<{
    employee_id: string
    name: string
    role: string
    assigned_total: number
    assigned_open: number
    assigned_closed: number
  }>> {
    const response = await api.get('/stats/employees')
    return response.data
  },
}
