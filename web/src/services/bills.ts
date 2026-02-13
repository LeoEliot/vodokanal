import api from './api'
import type {
  Bill,
  CreateBillRequest,
  Tariff,
  Payment,
  PaymentMethod,
  PaginationParams,
  PaginatedResponse
} from '../types'

export const billsService = {
  // Bills
  async getBills(params?: PaginationParams): Promise<PaginatedResponse<Bill>> {
    const response = await api.get<PaginatedResponse<Bill>>('/bills', { params })
    return response.data
  },

  async getBill(id: string): Promise<Bill> {
    const response = await api.get<Bill>(`/bills/${id}`)
    return response.data
  },

  async getSubscriberBills(subscriberId: string, params?: PaginationParams): Promise<Bill[]> {
    const response = await api.get<Bill[]>(`/bills/subscriber/${subscriberId}`, { params })
    return response.data
  },

  async getAccountBills(accountNumber: string): Promise<Bill[]> {
    const response = await api.get<Bill[]>(`/bills/account/${accountNumber}`)
    return response.data
  },

  async createBill(data: CreateBillRequest): Promise<Bill> {
    const response = await api.post<Bill>('/bills', data)
    return response.data
  },

  async updateBill(id: string, data: Partial<Bill>): Promise<Bill> {
    const response = await api.put<Bill>(`/bills/${id}`, data)
    return response.data
  },

  async cancelBill(id: string): Promise<Bill> {
    const response = await api.post<Bill>(`/bills/${id}/cancel`)
    return response.data
  },

  async recalculateBill(id: string): Promise<Bill> {
    const response = await api.post<Bill>(`/bills/${id}/recalculate`)
    return response.data
  },

  // Tariffs
  async getTariffs(): Promise<Tariff[]> {
    const response = await api.get<Tariff[]>('/tariffs')
    return response.data
  },

  async getActiveTariff(): Promise<Tariff> {
    const response = await api.get<Tariff>('/tariffs/active')
    return response.data
  },

  async createTariff(data: Partial<Tariff>): Promise<Tariff> {
    const response = await api.post<Tariff>('/tariffs', data)
    return response.data
  },

  async updateTariff(id: string, data: Partial<Tariff>): Promise<Tariff> {
    const response = await api.put<Tariff>(`/tariffs/${id}`, data)
    return response.data
  },

  async deleteTariff(id: string): Promise<void> {
    await api.delete(`/tariffs/${id}`)
  },

  // Reports
  async getDebtorsReport(): Promise<{
    total_debtors: number
    total_amount: string
    bills: Bill[]
  }> {
    const response = await api.get('/reports/debtors')
    return response.data
  },

  async getMonthlyReport(month?: string): Promise<{
    month: string
    total_bills: number
    total_issued: string
    total_paid: string
    bills: Bill[]
  }> {
    const response = await api.get('/reports/monthly', {
      params: month ? { month } : undefined
    })
    return response.data
  },
}

export const paymentsService = {
  async getPayments(params?: PaginationParams): Promise<PaginatedResponse<Payment>> {
    const response = await api.get<PaginatedResponse<Payment>>('/payments', { params })
    return response.data
  },

  async getPayment(id: string): Promise<Payment> {
    const response = await api.get<Payment>(`/payments/${id}`)
    return response.data
  },

  async createPayment(data: {
    bill_id: string
    amount?: number
    method?: string
  }): Promise<{
    payment_id: string
    amount: string
    payment_url?: string
    qr_code_url?: string
    status: string
  }> {
    const response = await api.post('/payments', data)
    return response.data
  },

  async cancelPayment(id: string): Promise<Payment> {
    const response = await api.post<Payment>(`/payments/${id}/cancel`)
    return response.data
  },

  async getPaymentStatus(id: string): Promise<{ status: string; gateway_status: string }> {
    const response = await api.get<{ status: string; gateway_status: string }>(`/payments/${id}/status`)
    return response.data
  },

  async getBillPayments(billId: string): Promise<Payment[]> {
    const response = await api.get<Payment[]>(`/bills/${billId}/payments`)
    return response.data
  },

  async createRefund(paymentId: string, data: { amount: string; reason: string }): Promise<void> {
    await api.post(`/payments/${paymentId}/refund`, data)
  },

  async getPaymentMethods(): Promise<PaymentMethod[]> {
    const response = await api.get<PaymentMethod[]>('/payment-methods')
    return response.data
  },
}
