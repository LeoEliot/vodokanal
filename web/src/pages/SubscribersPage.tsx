import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { subscribersService } from '../services/subscribers'
import { DataTable, Pagination, Button, Input, FormModal, StatusBadge } from '../components/ui'
import type { Subscriber, Column, CreateSubscriberRequest } from '../types'
import { Link } from 'react-router-dom'

const PlusIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
  </svg>
)

const SearchIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
  </svg>
)

const EditIcon = () => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
  </svg>
)

export const SubscribersPage = () => {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [search, setSearch] = useState('')
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingSubscriber, setEditingSubscriber] = useState<Subscriber | null>(null)
  const [formData, setFormData] = useState<CreateSubscriberRequest>({
    account_number: '',
    last_name: '',
    first_name: '',
    middle_name: '',
    email: '',
    phone: '',
    address: '',
    apartment: '',
  })

  const { data, isLoading } = useQuery({
    queryKey: ['subscribers', page, pageSize, search],
    queryFn: () => subscribersService.getSubscribers({ page, page_size: pageSize, search }),
  })

  const createMutation = useMutation({
    mutationFn: subscribersService.createSubscriber,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['subscribers'] })
      setIsModalOpen(false)
      resetForm()
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateSubscriberRequest> }) =>
      subscribersService.updateSubscriber(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['subscribers'] })
      setIsModalOpen(false)
      setEditingSubscriber(null)
      resetForm()
    },
  })

  const handleEdit = (subscriber: Subscriber) => {
    setEditingSubscriber(subscriber)
    setFormData({
      account_number: subscriber.account_number,
      last_name: subscriber.last_name,
      first_name: subscriber.first_name,
      middle_name: subscriber.middle_name || '',
      email: subscriber.email,
      phone: subscriber.phone,
      address: subscriber.address,
      apartment: subscriber.apartment || '',
    })
    setIsModalOpen(true)
  }

  const handleSubmit = () => {
    if (editingSubscriber) {
      updateMutation.mutate({ id: editingSubscriber.id, data: formData })
    } else {
      createMutation.mutate(formData)
    }
  }

  const resetForm = () => {
    setFormData({
      account_number: '',
      last_name: '',
      first_name: '',
      middle_name: '',
      email: '',
      phone: '',
      address: '',
      apartment: '',
    })
    setEditingSubscriber(null)
  }

  const columns: Column<Subscriber>[] = [
    {
      key: 'account_number',
      title: 'Лицевой счёт',
      sortable: true,
      render: (value, row) => (
        <Link to={`/subscribers/${row.id}`} className="text-blue-600 hover:underline font-medium">
          {value}
        </Link>
      ),
    },
    {
      key: 'last_name',
      title: 'ФИО',
      sortable: true,
      render: (_, row) => `${row.last_name} ${row.first_name} ${row.middle_name || ''}`.trim(),
    },
    {
      key: 'phone',
      title: 'Телефон',
    },
    {
      key: 'email',
      title: 'Email',
    },
    {
      key: 'address',
      title: 'Адрес',
      render: (_, row) => `${row.address}${row.apartment ? `, кв. ${row.apartment}` : ''}`,
    },
    {
      key: 'is_active',
      title: 'Статус',
      render: (value) => (
        <StatusBadge status={value ? 'active' : 'inactive'} />
      ),
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
        <button
          onClick={() => handleEdit(row)}
          className="p-2 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
        >
          <EditIcon />
        </button>
      ),
    },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Абоненты</h1>
          <p className="text-gray-500 mt-1">Управление абонентами водоканала</p>
        </div>
        <Button onClick={() => { resetForm(); setIsModalOpen(true) }}>
          <PlusIcon />
          Добавить абонента
        </Button>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
        <div className="flex flex-col sm:flex-row gap-4">
          <div className="relative flex-1">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-gray-400">
              <SearchIcon />
            </div>
            <Input
              placeholder="Поиск по лицевому счёту, ФИО, телефону или email..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              fullWidth
              className="pl-10"
            />
          </div>
        </div>
      </div>

      {/* Table */}
      <DataTable
        data={data?.data || []}
        columns={columns}
        keyField="id"
        isLoading={isLoading}
        emptyMessage={search ? 'Ничего не найдено' : 'Нет абонентов'}
      />

      {/* Pagination */}
      {data && data.total > 0 && (
        <Pagination
          currentPage={page}
          totalPages={Math.ceil(data.total / pageSize)}
          total={data.total}
          pageSize={pageSize}
          onPageChange={setPage}
          onPageSizeChange={setPageSize}
        />
      )}

      {/* Create/Edit Modal */}
      <FormModal
        isOpen={isModalOpen}
        onClose={() => { setIsModalOpen(false); setEditingSubscriber(null); resetForm() }}
        title={editingSubscriber ? 'Редактировать абонента' : 'Добавить абонента'}
        onSubmit={handleSubmit}
        isSubmitting={createMutation.isPending || updateMutation.isPending}
        submitLabel={editingSubscriber ? 'Сохранить' : 'Создать'}
      >
        <div className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Input
              label="Лицевой счёт"
              value={formData.account_number}
              onChange={(e) => setFormData({ ...formData, account_number: e.target.value })}
              required
              fullWidth
            />
            <div></div>
            <Input
              label="Фамилия"
              value={formData.last_name}
              onChange={(e) => setFormData({ ...formData, last_name: e.target.value })}
              required
              fullWidth
            />
            <Input
              label="Имя"
              value={formData.first_name}
              onChange={(e) => setFormData({ ...formData, first_name: e.target.value })}
              required
              fullWidth
            />
            <Input
              label="Отчество"
              value={formData.middle_name}
              onChange={(e) => setFormData({ ...formData, middle_name: e.target.value })}
              fullWidth
            />
            <div></div>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Input
              label="Телефон"
              type="tel"
              value={formData.phone}
              onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
              required
              placeholder="+7 (999) 123-45-67"
              fullWidth
            />
            <Input
              label="Email"
              type="email"
              value={formData.email}
              onChange={(e) => setFormData({ ...formData, email: e.target.value })}
              required
              fullWidth
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
        </div>
      </FormModal>
    </div>
  )
}
