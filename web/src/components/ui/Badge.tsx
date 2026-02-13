import React from 'react'

interface BadgeProps {
  children: React.ReactNode
  variant?: 'success' | 'warning' | 'danger' | 'info' | 'gray' | 'primary'
  size?: 'sm' | 'md'
  className?: string
}

export const Badge: React.FC<BadgeProps> = ({
  children,
  variant = 'gray',
  size = 'md',
  className = ''
}) => {
  const variantStyles = {
    success: 'bg-green-100 text-green-800',
    warning: 'bg-yellow-100 text-yellow-800',
    danger: 'bg-red-100 text-red-800',
    info: 'bg-blue-100 text-blue-800',
    gray: 'bg-gray-100 text-gray-800',
    primary: 'bg-blue-600 text-white',
  }

  const sizeStyles = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-1 text-sm',
  }

  return (
    <span className={`inline-flex items-center font-medium rounded-full ${variantStyles[variant]} ${sizeStyles[size]} ${className}`}>
      {children}
    </span>
  )
}

// Status Badge for common statuses
interface StatusBadgeProps {
  status: string
  className?: string
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status, className = '' }) => {
  const statusConfig: Record<string, { variant: 'success' | 'warning' | 'danger' | 'info' | 'gray'; label: string }> = {
    // Common statuses
    active: { variant: 'success', label: 'Активен' },
    inactive: { variant: 'gray', label: 'Неактивен' },
    pending: { variant: 'warning', label: 'Ожидает' },
    draft: { variant: 'gray', label: 'Черновик' },
    open: { variant: 'info', label: 'Открыт' },
    in_progress: { variant: 'primary', label: 'В работе' },
    resolved: { variant: 'success', label: 'Решён' },
    closed: { variant: 'gray', label: 'Закрыт' },
    cancelled: { variant: 'danger', label: 'Отменён' },
    paid: { variant: 'success', label: 'Оплачен' },
    unpaid: { variant: 'danger', label: 'Не оплачен' },
    overdue: { variant: 'danger', label: 'Просрочен' },
    issued: { variant: 'info', label: 'Выставлен' },
    verified: { variant: 'success', label: 'Проверен' },
    rejected: { variant: 'danger', label: 'Отклонён' },
    submitted: { variant: 'info', label: 'Отправлен' },
    completed: { variant: 'success', label: 'Завершён' },
    failed: { variant: 'danger', label: 'Ошибка' },
    processing: { variant: 'warning', label: 'Обработка' },
  }

  const config = statusConfig[status.toLowerCase()] || { variant: 'gray', label: status }

  return (
    <Badge variant={config.variant} className={className}>
      {config.label}
    </Badge>
  )
}

// Priority Badge
interface PriorityBadgeProps {
  priority: 'low' | 'medium' | 'high' | 'urgent'
  className?: string
}

export const PriorityBadge: React.FC<PriorityBadgeProps> = ({ priority, className = '' }) => {
  const priorityConfig = {
    low: { variant: 'gray' as const, label: 'Низкий' },
    medium: { variant: 'info' as const, label: 'Средний' },
    high: { variant: 'warning' as const, label: 'Высокий' },
    urgent: { variant: 'danger' as const, label: 'Срочный' },
  }

  const config = priorityConfig[priority]

  return (
    <Badge variant={config.variant} className={className}>
      {config.label}
    </Badge>
  )
}
