import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { readingsService } from '../services/readings'
import { DataTable, Pagination, Button, Input, FormModal, StatusBadge, Select, Badge } from '../components/ui'
import type { Reading, Column, CreateReadingRequest } from '../types'

const PlusIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
  </svg>
)

const CheckIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
  </svg>
)

const XIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
  </svg>
)

const HotIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17.657 18.657A8 8 0 016.343 7.343S7 9 9 10c0-2 .5-5 2.986-7C14 5 16.09 5.777 17.656 7.343A7.975 7.975 0 0120 13a7.975 7.975 0 01-2.343 5.657z" />
  </svg>
)

const ColdIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
  </svg>
)

export const ReadingsPage = () => {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [selectedReading, setSelectedReading] = useState<Reading | null>(null)
  const [formData, setFormData] = useState<CreateReadingRequest>({
    counter_id: '',
    value: 0,
    date: new Date().toISOString().split('T')[0],
  })

  const { data, isLoading } = useQuery({
    queryKey: ['readings', page, pageSize, statusFilter],
    queryFn: () => readingsService.getReadings({ page, page_size: pageSize }),
  })

  const createMutation = useMutation({
    mutationFn: readingsService.createReading,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['readings'] })
      setIsModalOpen(false)
      resetForm()
    },
  })

  const verifyMutation = useMutation({
    mutationFn: (id: string) => readingsService.verifyReading(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['readings'] })
      setSelectedReading(null)
    },
  })

  const rejectMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      readingsService.rejectReading(id, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['readings'] })
      setSelectedReading(null)
    },
  })

  const handleSubmit = () => {
    createMutation.mutate(formData)
  }

  const resetForm = () => {
    setFormData({
      counter_id: '',
      value: 0,
      date: new Date().toISOString().split('T')[0],
    })
  }

  const columns: Column<Reading>[] = [
    {
      key: 'date',
      title: 'Дата',
      sortable: true,
      render: (value) => new Date(value).toLocaleDateString('ru-RU'),
    },
    {
      key: 'account_number',
      title: 'Лицевой счёт',
      render: (_, row) => row.account_number || row.subscriber_id,
    },
    {
      key: 'subscriber_name',
      title: 'Абонент',
    },
    {
      key: 'counter_type',
      title: 'Тип',
      render: (value) => (
        <div className="flex items-center gap-2">
          {value === 'hot' ? (
            <>
              <HotIcon />
              <Badge variant="warning">Горячая</Badge>
            </>
          ) : (
            <>
              <ColdIcon />
              <Badge variant="info">Холодная</Badge>
            </>
          )}
        </div>
      ),
    },
    {
      key: 'counter_serial',
      title: 'Счётчик №',
    },
    {
      key: 'value',
      title: 'Показание',
      render: (value, row) => (
        <div>
          <span className="font-semibold">{value} м³</span>
          {row.previous_value && (
            <span className="text-sm text-gray-500 ml-2">
              (пред: {row.previous_value})
            </span>
          )}
        </div>
      ),
    },
    {
      key: 'consumption',
      title: 'Расход',
      render: (value) => value ? `${value} м³` : '-',
    },
    {
      key: 'status',
      title: 'Статус',
      render: (value) => <StatusBadge status={value} />,
    },
    {
      key: 'actions',
      title: 'Действия',
      render: (_, row) => (
        <div className="flex items-center gap-2">
          {row.status === 'submitted' && (
            <>
              <button
                onClick={() => verifyMutation.mutate(row.id)}
                className="p-2 text-green-600 hover:bg-green-50 rounded-lg transition-colors"
                title="Подтвердить"
              >
                <CheckIcon />
              </button>
              <button
                onClick={() => setSelectedReading(row)}
                className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                title="Отклонить"
              >
                <XIcon />
              </button>
            </>
          )}
        </div>
      ),
    },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Показания счётчиков</h1>
          <p className="text-gray-500 mt-1">Управление показаниями водосчётчиков</p>
        </div>
        <Button onClick={() => { resetForm(); setIsModalOpen(true) }}>
          <PlusIcon />
          Добавить показания
        </Button>
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
              { value: 'draft', label: 'Черновики' },
              { value: 'submitted', label: 'Отправлены' },
              { value: 'verified', label: 'Проверены' },
              { value: 'rejected', label: 'Отклонены' },
            ]}
            className="min-w-[200px]"
          />
        </div>
      </div>

      {/* Table */}
      <DataTable
        data={data?.data || []}
        columns={columns}
        keyField="id"
        isLoading={isLoading}
        emptyMessage="Нет показаний"
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

      {/* Add Reading Modal */}
      <FormModal
        isOpen={isModalOpen}
        onClose={() => { setIsModalOpen(false); resetForm() }}
        title="Добавить показания"
        onSubmit={handleSubmit}
        isSubmitting={createMutation.isPending}
      >
        <div className="space-y-4">
          <Input
            label="ID счётчика"
            value={formData.counter_id}
            onChange={(e) => setFormData({ ...formData, counter_id: e.target.value })}
            required
            fullWidth
          />
          <Input
            label="Показание"
            type="number"
            step="0.001"
            value={formData.value}
            onChange={(e) => setFormData({ ...formData, value: parseFloat(e.target.value) || 0 })}
            required
            fullWidth
          />
          <Input
            label="Дата"
            type="date"
            value={formData.date}
            onChange={(e) => setFormData({ ...formData, date: e.target.value })}
            required
            fullWidth
          />
        </div>
      </FormModal>

      {/* Reject Modal */}
      <FormModal
        isOpen={!!selectedReading}
        onClose={() => setSelectedReading(null)}
        title="Отклонить показания"
        onSubmit={() => {
          const reason = (document.getElementById('reject-reason') as HTMLTextAreaElement)?.value
          if (selectedReading && reason) {
            rejectMutation.mutate({ id: selectedReading.id, reason })
          }
        }}
        isSubmitting={rejectMutation.isPending}
        submitLabel="Отклонить"
        submitDisabled={false}
      >
        <div className="space-y-4">
          <p className="text-gray-600">
            Вы отклоняете показания от <strong>{selectedReading?.date}</strong>
            {selectedReading && ` на сумму ${selectedReading.value} м³`}
          </p>
          <Input
            id="reject-reason"
            label="Причина отклонения"
            component="textarea"
            rows={3}
            required
            fullWidth
          />
        </div>
      </FormModal>
    </div>
  )
}
