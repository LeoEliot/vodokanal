// ==================== Auth Types ====================
export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  password: string
  name: string
}

export interface AuthResponse {
  token: string
  type: string
  user?: User
}

export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'operator' | 'subscriber'
}

// ==================== Subscriber Types ====================
export interface Subscriber {
  id: string
  account_number: string
  last_name: string
  first_name: string
  middle_name?: string
  email: string
  phone: string
  address: string
  apartment?: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateSubscriberRequest {
  account_number: string
  last_name: string
  first_name: string
  middle_name?: string
  email: string
  phone: string
  address: string
  apartment?: string
}

export interface UpdateSubscriberRequest extends Partial<CreateSubscriberRequest> {}

export interface SubscribersResponse {
  data: Subscriber[]
  total: number
  page: number
  page_size: number
}

// ==================== Counter Types ====================
export interface Counter {
  id: string
  subscriber_id: string
  subscriber_name?: string
  serial_number: string
  type: 'hot' | 'cold' | 'sewage'
  brand?: string
  model?: string
  installation_date: string
  verification_date: string
  next_verification_date?: string
  is_active: boolean
  current_value?: number
  created_at: string
  updated_at: string
}

export interface CreateCounterRequest {
  subscriber_id: string
  serial_number: string
  type: 'hot' | 'cold' | 'sewage'
  brand?: string
  model?: string
  installation_date: string
  verification_date: string
}

// ==================== Reading Types ====================
export interface Reading {
  id: string
  subscriber_id: string
  subscriber_name?: string
  account_number?: string
  counter_id: string
  counter_type?: 'hot' | 'cold'
  counter_serial?: string
  value: number
  previous_value?: number
  consumption?: number
  date: string
  status: 'draft' | 'submitted' | 'verified' | 'rejected'
  verified_by?: string
  verified_at?: string
  rejection_reason?: string
  photo_url?: string
  created_at: string
  updated_at: string
}

export interface CreateReadingRequest {
  subscriber_id?: string
  counter_id: string
  value: number
  date?: string
  photo_url?: string
}

export interface ReadingSubmission {
  subscriber_id: string
  readings: {
    counter_id: string
    value: number
  }[]
  period: string
}

// ==================== Bill Types ====================
export interface Bill {
  id: string
  subscriber_id: string
  subscriber_name?: string
  account_number: string
  bill_number: string
  period_start: string
  period_end: string
  issued_at: string
  due_date: string

  // Usage details
  hot_water_usage: number
  cold_water_usage: number

  // Tariffs
  hot_water_tariff: number
  cold_water_tariff: number

  // Amounts
  hot_water_amount: number
  cold_water_amount: number
  sewage_amount: number
  service_charge: number
  maintenance_charge: number
  total_amount: number

  // Status
  status: 'draft' | 'issued' | 'paid' | 'overdue' | 'cancelled'
  paid_at?: string
  payment_id?: string

  created_at: string
  updated_at: string
}

export interface CreateBillRequest {
  subscriber_id: string
  period_start: string
  period_end: string
  hot_water_usage: number
  cold_water_usage: number
}

export interface Tariff {
  id: string
  name: string
  hot_water_price: number
  cold_water_price: number
  sewage_price: number
  service_charge: number
  valid_from: string
  valid_to?: string
  is_active: boolean
  created_at: string
}

// ==================== Payment Types ====================
export interface Payment {
  id: string
  bill_id: string
  subscriber_id: string
  amount: number
  currency: string
  method: 'card' | 'yookassa' | 'stripe' | 'sbp' | 'cash'
  status: 'pending' | 'processing' | 'completed' | 'failed' | 'refunded' | 'cancelled'
  gateway: 'yookassa' | 'stripe' | 'manual'
  gateway_tx_id?: string
  payment_url?: string
  qr_code_url?: string
  description?: string
  created_at: string
  completed_at?: string
  failed_at?: string
  refunded_amount?: number
  refund_reason?: string
}

export interface CreatePaymentRequest {
  bill_id: string
  amount: number
  method: 'card' | 'yookassa' | 'sbp'
  success_url?: string
  fail_url?: string
}

export interface PaymentResponse {
  payment_id: string
  amount: number
  currency: string
  status: string
  payment_url?: string
  qr_code_url?: string
  expires_at: string
}

export interface PaymentMethod {
  id: string
  name: string
  description: string
  icon: string
  enabled: boolean
}

// ==================== Ticket Types ====================
export interface Ticket {
  id: string
  subscriber_id: string
  subscriber_name?: string
  account_number: string
  title: string
  description: string
  category: 'leak' | 'repair' | 'billing' | 'quality' | 'other'
  priority: 'low' | 'medium' | 'high' | 'urgent'
  status: 'open' | 'in_progress' | 'resolved' | 'closed' | 'cancelled'

  // Address
  address: string
  apartment?: string

  // Assignment
  assigned_to?: string
  assigned_to_name?: string
  assigned_at?: string

  // Resolution
  resolution?: string
  resolved_at?: string
  resolved_by?: string

  // Scheduling
  scheduled_date?: string
  scheduled_time?: 'morning' | 'afternoon' | 'evening'

  // Contact
  contact_name: string
  contact_phone: string
  contact_email?: string

  // Feedback
  rating?: number
  feedback?: string

  // Timestamps
  created_at: string
  updated_at: string
  closed_at?: string

  source: 'web' | 'mobile' | 'phone' | 'email'
}

export interface CreateTicketRequest {
  subscriber_id?: string
  title: string
  description: string
  category: 'leak' | 'repair' | 'billing' | 'quality' | 'other'
  priority?: 'low' | 'medium' | 'high' | 'urgent'
  address: string
  apartment?: string
  contact_name: string
  contact_phone: string
  contact_email?: string
}

export interface Comment {
  id: string
  ticket_id: string
  content: string
  author_id: string
  author_name: string
  author_role: 'subscriber' | 'employee' | 'admin'
  is_internal: boolean
  created_at: string
}

export interface Employee {
  id: string
  name: string
  email: string
  phone: string
  role: 'manager' | 'operator' | 'technician'
  is_active: boolean
  skills: string[]
}

// ==================== Notification Types ====================
export interface Notification {
  id: string
  user_id: string
  type: 'email' | 'sms' | 'push'
  channel: 'payment_reminder' | 'reading_reminder' | 'ticket_update' | 'bill_generated'
  subject?: string
  body: string
  status: 'pending' | 'sent' | 'failed'
  created_at: string
  sent_at?: string
  error?: string
}

export interface NotificationTemplate {
  id: string
  channel: string
  subject: string
  body: string
  variables: string[]
}

// ==================== Dashboard Types ====================
export interface DashboardStats {
  subscribers: {
    total: number
    active: number
    new_this_month: number
  }
  readings: {
    total_this_month: number
    pending_verification: number
    verified: number
  }
  bills: {
    total_issued: number
    total_paid: number
    total_outstanding: number
    overdue_count: number
  }
  tickets: {
    open: number
    in_progress: number
    resolved_today: number
  }
}

export interface ChartData {
  labels: string[]
  datasets: {
    label: string
    data: number[]
    color?: string
  }[]
}

// ==================== Common Types ====================
export interface PaginationParams {
  page?: number
  page_size?: number
  sort?: string
  order?: 'asc' | 'desc'
  search?: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface ApiResponse<T> {
  data: T
  message?: string
}

export interface ApiError {
  message: string
  code?: string
  details?: Record<string, string[]>
}
