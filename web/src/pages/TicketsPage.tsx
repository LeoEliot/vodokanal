import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ticketsService } from '../services/tickets'
import { DataTable, Pagination, Button, Input, Textarea, FormModal, StatusBadge, PriorityBadge, Select, Modal, Card } from '../components/ui'
import type { Ticket, Column, CreateTicketRequest, Comment } from '../types'

const PlusIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
  </svg>
)

const ChatIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
  </svg>
)

const UserIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
  </svg>
)

const CheckIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
  </svg>
)

const categories = [
  { value: 'leak', label: 'Утечка воды' },
  { value: 'repair', label: 'Ремонт' },
  { value: 'billing', label: 'Биллинг' },
  { value: 'quality', label: 'Качество воды' },
  { value: 'other', label: 'Другое' },
]

const priorities = [
  { value: 'low', label: 'Низкий' },
  { value: 'medium', label: 'Средний' },
  { value: 'high', label: 'Высокий' },
  { value: 'urgent', label: 'Срочный' },
]

const categoryLabels: Record<string, string> = {
  leak: 'Утечка воды',
  repair: 'Ремонт',
  billing: 'Биллинг',
  quality: 'Качество воды',
  other: 'Другое',
}

export const TicketsPage = () => {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [categoryFilter, setCategoryFilter] = useState<string>('')
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null)
  const [commentsOpen, setCommentsOpen] = useState(false)
  const [formData, setFormData] = useState<CreateTicketRequest>({
    title: '',
    description: '',
    category: 'other',
    priority: 'medium',
    address: '',
    apartment: '',
    contact_name: '',
    contact_phone: '',
    contact_email: '',
  })

  const { data: ticketsData, isLoading } = useQuery({
    queryKey: ['tickets', page, pageSize, statusFilter, categoryFilter],
    queryFn: () => ticketsService.getTickets({ page, page_size: pageSize }),
  })

  const { data: employees } = useQuery({
    queryKey: ['employees'],
    queryFn: () => ticketsService.getEmployees(),
  })

  const { data: comments, refetch: refetchComments } = useQuery({
    queryKey: ['ticket-comments', selectedTicket?.id],
    queryFn: () => ticketsService.getTicketComments(selectedTicket!.id),
    enabled: !!selectedTicket && commentsOpen,
  })

  const { data: ticketStats } = useQuery({
    queryKey: ['tickets-stats'],
    queryFn: () => ticketsService.getTicketsStats(),
  })

  const createMutation = useMutation({
    mutationFn: ticketsService.createTicket,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tickets'] })
      setIsModalOpen(false)
      resetForm()
    },
  })

  const closeMutation = useMutation({
    mutationFn: ({ id, resolution }: { id: string; resolution: string }) =>
      ticketsService.closeTicket(id, { resolution }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tickets'] })
      setSelectedTicket(null)
    },
  })

  const reopenMutation = useMutation({
    mutationFn: (id: string) => ticketsService.reopenTicket(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tickets'] })
    },
  })

  const assignMutation = useMutation({
    mutationFn: ({ id, employeeId }: { id: string; employeeId: string }) =>
      ticketsService.assignTicket(id, employeeId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tickets'] })
    },
  })

  const addCommentMutation = useMutation({
    mutationFn: ({ ticketId, content }: { ticketId: string; content: string }) =>
      ticketsService.addComment(ticketId, content),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ticket-comments'] })
    },
  })

  const handleSubmit = () => {
    createMutation.mutate(formData)
  }

  const resetForm = () => {
    setFormData({
      title: '',
      description: '',
      category: 'other',
      priority: 'medium',
      address: '',
      apartment: '',
      contact_name: '',
      contact_phone: '',
      contact_email: '',
    })
  }

  const handleViewTicket = (ticket: Ticket) => {
    setSelectedTicket(ticket)
    setCommentsOpen(false)
  }

  const handleAddComment = () => {
    const input = document.getElementById('new-comment') as HTMLTextAreaElement
    if (input && input.value.trim() && selectedTicket) {
      addCommentMutation.mutate({
        ticketId: selectedTicket.id,
        content: input.value,
      })
      input.value = ''
    }
  }

  const columns: Column<Ticket>[] = [
    {
      key: 'id',
      title: 'Номер',
      sortable: true,
      render: (value) => `#${value.slice(-6).toUpperCase()}`,
    },
    {
      key: 'title',
      title: 'Тема',
      sortable: true,
      render: (value) => (
        <span className="font-medium text-gray-900">{value}</span>
      ),
    },
    {
      key: 'category',
      title: 'Категория',
      render: (value) => categoryLabels[value] || value,
    },
    {
      key: 'priority',
      title: 'Приоритет',
      render: (value) => <PriorityBadge priority={value as any} />,
    },
    {
      key: 'status',
      title: 'Статус',
      render: (value) => <StatusBadge status={value} />,
    },
    {
      key: 'address',
      title: 'Адрес',
      render: (_, row) => `${row.address}${row.apartment ? `, кв. ${row.apartment}` : ''}`,
    },
    {
      key: 'created_at',
      title: 'Создан',
      render: (value) => new Date(value).toLocaleDateString('ru-RU'),
    },
    {
      key: 'actions',
      title: '',
      render: (_, row) => (
        <Button
          size="sm"
          variant="ghost"
          onClick={() => handleViewTicket(row)}
        >
          Подробнее
        </Button>
      ),
    },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Обращения</h1>
          <p className="text-gray-500 mt-1">Управление обращениями абонентов</p>
        </div>
        <Button onClick={() => { resetForm(); setIsModalOpen(true) }}>
          <PlusIcon />
          Создать обращение
        </Button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <Card className="bg-blue-50 border-blue-200">
          <p className="text-sm text-blue-600 font-medium">Открыто</p>
          <p className="text-2xl font-bold text-blue-700 mt-1">
            {ticketStats?.by_status?.open || 0}
          </p>
        </Card>
        <Card className="bg-yellow-50 border-yellow-200">
          <p className="text-sm text-yellow-600 font-medium">В работе</p>
          <p className="text-2xl font-bold text-yellow-700 mt-1">
            {ticketStats?.by_status?.in_progress || 0}
          </p>
        </Card>
        <Card className="bg-green-50 border-green-200">
          <p className="text-sm text-green-600 font-medium">Решено</p>
          <p className="text-2xl font-bold text-green-700 mt-1">
            {ticketStats?.by_status?.resolved || 0}
          </p>
        </Card>
        <Card className="bg-gray-50 border-gray-200">
          <p className="text-sm text-gray-600 font-medium">Закрыто</p>
          <p className="text-2xl font-bold text-gray-700 mt-1">
            {ticketStats?.by_status?.closed || 0}
          </p>
        </Card>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
        <div className="flex flex-col sm:flex-row gap-4">
          <Select
            label="Статус"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            options={[
              { value: '', label: 'Все статусы' },
              { value: 'open', label: 'Открытые' },
              { value: 'in_progress', label: 'В работе' },
              { value: 'resolved', label: 'Решённые' },
              { value: 'closed', label: 'Закрытые' },
            ]}
            className="min-w-[200px]"
          />
          <Select
            label="Категория"
            value={categoryFilter}
            onChange={(e) => setCategoryFilter(e.target.value)}
            options={[{ value: '', label: 'Все категории' }, ...categories]}
            className="min-w-[200px]"
          />
        </div>
      </div>

      {/* Table */}
      <DataTable
        data={ticketsData?.data || []}
        columns={columns}
        keyField="id"
        isLoading={isLoading}
        emptyMessage="Нет обращений"
      />

      {/* Pagination */}
      {ticketsData && ticketsData.total > 0 && (
        <Pagination
          currentPage={page}
          totalPages={Math.ceil(ticketsData.total / pageSize)}
          total={ticketsData.total}
          pageSize={pageSize}
          onPageChange={setPage}
          onPageSizeChange={setPageSize}
        />
      )}

      {/* Create Ticket Modal */}
      <FormModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title="Создать обращение"
        onSubmit={handleSubmit}
        isSubmitting={createMutation.isPending}
        size="lg"
      >
        <div className="space-y-4">
          <Input
            label="Тема"
            value={formData.title}
            onChange={(e) => setFormData({ ...formData, title: e.target.value })}
            required
            fullWidth
          />
          <Textarea
            label="Описание"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            required
            fullWidth
          />
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Select
              label="Категория"
              value={formData.category}
              onChange={(e) => setFormData({ ...formData, category: e.target.value as any })}
              options={categories}
            />
            <Select
              label="Приоритет"
              value={formData.priority}
              onChange={(e) => setFormData({ ...formData, priority: e.target.value as any })}
              options={priorities}
            />
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-[1fr_auto] gap-4">
            <Input
              label="Адрес"
              value={formData.address}
              onChange={(e) => setFormData({ ...formData, address: e.target.value })}
              required
              placeholder="ул. Примерная, д. 1"
              fullWidth
            />
            <Input
              label="Квартира"
              value={formData.apartment}
              onChange={(e) => setFormData({ ...formData, apartment: e.target.value })}
              placeholder="123"
              className="w-24"
            />
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Input
              label="Контактное лицо"
              value={formData.contact_name}
              onChange={(e) => setFormData({ ...formData, contact_name: e.target.value })}
              required
              fullWidth
            />
            <Input
              label="Телефон"
              type="tel"
              value={formData.contact_phone}
              onChange={(e) => setFormData({ ...formData, contact_phone: e.target.value })}
              required
              fullWidth
            />
          </div>
          <Input
            label="Email"
            type="email"
            value={formData.contact_email}
            onChange={(e) => setFormData({ ...formData, contact_email: e.target.value })}
            fullWidth
          />
        </div>
      </FormModal>

      {/* Ticket Detail Modal */}
      <Modal
        isOpen={!!selectedTicket && !isModalOpen}
        onClose={() => { setSelectedTicket(null); setCommentsOpen(false) }}
        title={`Обращение #${selectedTicket?.id.slice(-6).toUpperCase()}`}
        size="lg"
        footer={
          <div className="flex items-center justify-between w-full">
            <div className="flex gap-2">
              {selectedTicket?.status !== 'closed' && selectedTicket?.status !== 'resolved' && (
                <>
                  {employees && employees.length > 0 && selectedTicket?.status !== 'assigned' && (
                    <Select
                      value=""
                      onChange={(e) => {
                        if (e.target.value && selectedTicket) {
                          assignMutation.mutate({ id: selectedTicket.id, employeeId: e.target.value })
                        }
                      }}
                      options={[
                        { value: '', label: 'Назначить...' },
                        ...employees.filter(e => e.is_active).map(e => ({
                          value: e.id,
                          label: e.name,
                        }))
                      ]}
                      className="w-48"
                    />
                  )}
                  {selectedTicket?.status !== 'closed' && (
                    <Button
                      variant="danger"
                      size="sm"
                      onClick={() => {
                        const resolution = prompt('Введите решение:')
                        if (resolution && selectedTicket) {
                          closeMutation.mutate({ id: selectedTicket.id, resolution })
                        }
                      }}
                    >
                      Закрыть
                    </Button>
                  )}
                </>
              )}
              {(selectedTicket?.status === 'closed' || selectedTicket?.status === 'resolved') && (
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => selectedTicket && reopenMutation.mutate(selectedTicket.id)}
                >
                  Переоткрыть
                </Button>
              )}
            </div>
            <Button variant="ghost" onClick={() => { setSelectedTicket(null); setCommentsOpen(false) }}>
              Закрыть
            </Button>
          </div>
        }
      >
        {selectedTicket && (
          <div className="space-y-6">
            {/* Ticket Info */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
              <div>
                <span className="text-gray-500">Статус:</span>
                <p className="mt-1"><StatusBadge status={selectedTicket.status} /></p>
              </div>
              <div>
                <span className="text-gray-500">Приоритет:</span>
                <p className="mt-1"><PriorityBadge priority={selectedTicket.priority} /></p>
              </div>
              <div>
                <span className="text-gray-500">Категория:</span>
                <p className="mt-1 font-medium">{categoryLabels[selectedTicket.category]}</p>
              </div>
              <div>
                <span className="text-gray-500">Создан:</span>
                <p className="mt-1 font-medium">
                  {new Date(selectedTicket.created_at).toLocaleString('ru-RU')}
                </p>
              </div>
            </div>

            <div>
              <h3 className="font-semibold text-gray-900 mb-1">{selectedTicket.title}</h3>
              <p className="text-gray-600">{selectedTicket.description}</p>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm bg-gray-50 p-4 rounded-lg">
              <div>
                <span className="text-gray-500">Адрес:</span>
                <p className="mt-1 font-medium">
                  {selectedTicket.address}{selectedTicket.apartment ? `, кв. ${selectedTicket.apartment}` : ''}
                </p>
              </div>
              <div>
                <span className="text-gray-500">Контакты:</span>
                <p className="mt-1 font-medium">{selectedTicket.contact_name}</p>
                <p className="text-gray-600">{selectedTicket.contact_phone}</p>
                {selectedTicket.contact_email && <p className="text-gray-600">{selectedTicket.contact_email}</p>}
              </div>
            </div>

            {selectedTicket.assigned_to && (
              <div className="flex items-center gap-2 text-sm bg-blue-50 p-3 rounded-lg">
                <UserIcon />
                <span className="text-blue-800">
                  Назначен на: <strong>{selectedTicket.assigned_to_name}</strong>
                </span>
              </div>
            )}

            {selectedTicket.resolution && (
              <div className="bg-green-50 p-4 rounded-lg">
                <h4 className="font-semibold text-green-800 mb-1">Решение</h4>
                <p className="text-green-700">{selectedTicket.resolution}</p>
              </div>
            )}

            {/* Comments Section */}
            <div className="border-t border-gray-200 pt-4">
              <div className="flex items-center justify-between mb-4">
                <h4 className="font-semibold flex items-center gap-2">
                  <ChatIcon />
                  Комментарии
                </h4>
                {!commentsOpen && (
                  <Button size="sm" variant="ghost" onClick={() => setCommentsOpen(true)}>
                    Показать ({comments?.length || 0})
                  </Button>
                )}
              </div>

              {commentsOpen && (
                <div className="space-y-4">
                  <div className="space-y-3 max-h-60 overflow-y-auto">
                    {comments?.map((comment) => (
                      <div
                        key={comment.id}
                        className={`p-3 rounded-lg ${
                          comment.is_internal ? 'bg-yellow-50' : 'bg-gray-50'
                        }`}
                      >
                        <div className="flex items-center justify-between mb-1">
                          <span className="font-medium text-sm">{comment.author_name}</span>
                          <span className="text-xs text-gray-500">
                            {new Date(comment.created_at).toLocaleString('ru-RU')}
                          </span>
                        </div>
                        <p className="text-sm text-gray-700">{comment.content}</p>
                        {comment.is_internal && (
                          <p className="text-xs text-yellow-600 mt-1">Внутренний комментарий</p>
                        )}
                      </div>
                    ))}
                    {(!comments || comments.length === 0) && (
                      <p className="text-center text-gray-500 py-4">Пока нет комментариев</p>
                    )}
                  </div>

                  <div className="flex gap-2">
                    <Textarea
                      id="new-comment"
                      placeholder="Написать комментарий..."
                      fullWidth
                    />
                    <Button
                      onClick={handleAddComment}
                      isLoading={addCommentMutation.isPending}
                      className="self-end"
                    >
                      <CheckIcon />
                    </Button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}
