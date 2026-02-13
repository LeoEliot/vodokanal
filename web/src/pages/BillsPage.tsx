import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { billsService, paymentsService } from '../services/bills'
import { DataTable, Pagination, Button, Input, Select, StatusBadge, Badge, Modal, Card } from '../components/ui'
import type { Bill, Column } from '../types'

const CreditCardIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
  </svg>
)

const DownloadIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
  </svg>
)

const EyeIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
  </svg>
)

const SbpIcon = () => (
  <svg className="w-6 h-6" viewBox="0 0 24 24" fill="currentColor">
    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
  </svg>
)

export const BillsPage = () => {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [selectedBill, setSelectedBill] = useState<Bill | null>(null)
  const [paymentModalOpen, setPaymentModalOpen] = useState(false)
  const [paymentMethod, setPaymentMethod] = useState('card')
  const [qrCodeUrl, setQrCodeUrl] = useState('')

  const { data: billsData, isLoading: billsLoading } = useQuery({
    queryKey: ['bills', page, pageSize, statusFilter],
    queryFn: () => billsService.getBills({ page, page_size: pageSize }),
  })

  const { data: paymentMethods } = useQuery({
    queryKey: ['payment-methods'],
    queryFn: () => paymentsService.getPaymentMethods(),
  })

  const createPaymentMutation = useMutation({
    mutationFn: paymentsService.createPayment,
    onSuccess: (data) => {
      if (data.payment_url) {
        window.open(data.payment_url, '_blank')
      }
      if (data.qr_code_url) {
        setQrCodeUrl(data.qr_code_url)
      }
      queryClient.invalidateQueries({ queryKey: ['bills'] })
      queryClient.invalidateQueries({ queryKey: ['payments'] })
      setPaymentModalOpen(false)
    },
  })

  const handlePayment = (bill: Bill) => {
    setSelectedBill(bill)
    setPaymentModalOpen(true)
  }

  const submitPayment = () => {
    if (selectedBill) {
      createPaymentMutation.mutate({
        bill_id: selectedBill.id,
        amount: selectedBill.total_amount,
        method: paymentMethod as 'card' | 'yookassa' | 'sbp',
      })
    }
  }

  const formatDate = (dateStr: string) => new Date(dateStr).toLocaleDateString('ru-RU')

  const columns: Column<Bill>[] = [
    {
      key: 'bill_number',
      title: 'Номер счёта',
      sortable: true,
    },
    {
      key: 'period_start',
      title: 'Период',
      render: (_, row) => `${formatDate(row.period_start)} - ${formatDate(row.period_end)}`,
    },
    {
      key: 'subscriber_name',
      title: 'Абонент',
      render: (_, row) => row.subscriber_name || `Л/с ${row.account_number}`,
    },
    {
      key: 'total_amount',
      title: 'Сумма',
      sortable: true,
      render: (value) => (
        <span className="font-semibold text-gray-900">{value.toFixed(2)} ₽</span>
      ),
    },
    {
      key: 'due_date',
      title: 'Срок оплаты',
      render: (value) => {
        const date = new Date(value)
        const isOverdue = date < new Date()
        return (
          <span className={isOverdue ? 'text-red-600 font-medium' : ''}>
            {date.toLocaleDateString('ru-RU')}
          </span>
        )
      },
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
          <button
            onClick={() => setSelectedBill(row)}
            className="p-2 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
            title="Просмотр"
          >
            <EyeIcon />
          </button>
          <button
            className="p-2 text-gray-400 hover:text-green-600 hover:bg-green-50 rounded-lg transition-colors"
            title="Скачать PDF"
          >
            <DownloadIcon />
          </button>
          {row.status !== 'paid' && row.status !== 'cancelled' && (
            <Button
              size="sm"
              onClick={() => handlePayment(row)}
            >
              Оплатить
            </Button>
          )}
        </div>
      ),
    },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Счета</h1>
        <p className="text-gray-500 mt-1">Просмотр и оплата счетов за услуги водоснабжения</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Card className="bg-green-50 border-green-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-green-600 font-medium">Оплачено</p>
              <p className="text-2xl font-bold text-green-700 mt-1">
                {billsData?.data.filter(b => b.status === 'paid').length || 0}
              </p>
            </div>
          </div>
        </Card>
        <Card className="bg-yellow-50 border-yellow-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-yellow-600 font-medium">Ожидают оплаты</p>
              <p className="text-2xl font-bold text-yellow-700 mt-1">
                {billsData?.data.filter(b => b.status === 'issued').length || 0}
              </p>
            </div>
          </div>
        </Card>
        <Card className="bg-red-50 border-red-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-red-600 font-medium">Просрочено</p>
              <p className="text-2xl font-bold text-red-700 mt-1">
                {billsData?.data.filter(b => {
                  const dueDate = new Date(b.due_date)
                  return b.status === 'issued' && dueDate < new Date()
                }).length || 0}
              </p>
            </div>
          </div>
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
              { value: '', label: 'Все счета' },
              { value: 'issued', label: 'Неоплаченные' },
              { value: 'paid', label: 'Оплаченные' },
              { value: 'overdue', label: 'Просроченные' },
            ]}
            className="min-w-[200px]"
          />
        </div>
      </div>

      {/* Table */}
      <DataTable
        data={billsData?.data || []}
        columns={columns}
        keyField="id"
        isLoading={billsLoading}
        emptyMessage="Нет счетов"
      />

      {/* Pagination */}
      {billsData && billsData.total > 0 && (
        <Pagination
          currentPage={page}
          totalPages={Math.ceil(billsData.total / pageSize)}
          total={billsData.total}
          pageSize={pageSize}
          onPageChange={setPage}
          onPageSizeChange={setPageSize}
        />
      )}

      {/* Payment Modal */}
      <Modal
        isOpen={paymentModalOpen}
        onClose={() => setPaymentModalOpen(false)}
        title="Оплата счета"
        footer={
          <>
            <Button variant="ghost" onClick={() => setPaymentModalOpen(false)}>
              Отмена
            </Button>
            <Button onClick={submitPayment} isLoading={createPaymentMutation.isPending}>
              <CreditCardIcon />
              Оплатить
            </Button>
          </>
        }
      >
        <div className="space-y-6">
          {selectedBill && (
            <div className="bg-gray-50 rounded-lg p-4">
              <h3 className="font-semibold text-gray-900 mb-2">Счёт №{selectedBill.bill_number}</h3>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-gray-600">Период:</span>
                  <span className="font-medium">
                    {formatDate(selectedBill.period_start)} - {formatDate(selectedBill.period_end)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Горячая вода:</span>
                  <span>{selectedBill.hot_water_usage} м³ × {selectedBill.hot_water_tariff} ₽ = {selectedBill.hot_water_amount.toFixed(2)} ₽</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Холодная вода:</span>
                  <span>{selectedBill.cold_water_usage} м³ × {selectedBill.cold_water_tariff} ₽ = {selectedBill.cold_water_amount.toFixed(2)} ₽</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Водоотведение:</span>
                  <span>{selectedBill.sewage_amount.toFixed(2)} ₽</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Обслуживание:</span>
                  <span>{selectedBill.service_charge.toFixed(2)} ₽</span>
                </div>
                <div className="border-t border-gray-200 pt-2 mt-2">
                  <div className="flex justify-between text-lg font-bold">
                    <span>Итого:</span>
                    <span className="text-blue-600">{selectedBill.total_amount.toFixed(2)} ₽</span>
                  </div>
                </div>
              </div>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-3">
              Способ оплаты
            </label>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <button
                type="button"
                onClick={() => setPaymentMethod('card')}
                className={`p-4 border-2 rounded-lg text-left transition-colors ${
                  paymentMethod === 'card'
                    ? 'border-blue-500 bg-blue-50'
                    : 'border-gray-200 hover:border-gray-300'
                }`}
              >
                <div className="flex items-center gap-3">
                  <CreditCardIcon />
                  <div>
                    <p className="font-medium">Банковская карта</p>
                    <p className="text-sm text-gray-500">Visa, MasterCard, МИР</p>
                  </div>
                </div>
              </button>
              <button
                type="button"
                onClick={() => setPaymentMethod('sbp')}
                className={`p-4 border-2 rounded-lg text-left transition-colors ${
                  paymentMethod === 'sbp'
                    ? 'border-blue-500 bg-blue-50'
                    : 'border-gray-200 hover:border-gray-300'
                }`}
              >
                <div className="flex items-center gap-3">
                  <div className="text-purple-600"><SbpIcon /></div>
                  <div>
                    <p className="font-medium">СБП</p>
                    <p className="text-sm text-gray-500">Система быстрых платежей</p>
                  </div>
                </div>
              </button>
              <button
                type="button"
                onClick={() => setPaymentMethod('yookassa')}
                className={`p-4 border-2 rounded-lg text-left transition-colors ${
                  paymentMethod === 'yookassa'
                    ? 'border-blue-500 bg-blue-50'
                    : 'border-gray-200 hover:border-gray-300'
                }`}
              >
                <div className="flex items-center gap-3">
                  <div className="w-6 h-6 rounded-full bg-yellow-400 flex items-center justify-center">
                    <span className="text-xs font-bold text-yellow-900">Ю</span>
                  </div>
                  <div>
                    <p className="font-medium">ЮMoney</p>
                    <p className="text-sm text-gray-500">Из кошелька ЮMoney</p>
                  </div>
                </div>
              </button>
            </div>
          </div>
        </div>
      </Modal>

      {/* Bill Detail Modal */}
      <Modal
        isOpen={!!selectedBill && !paymentModalOpen}
        onClose={() => setSelectedBill(null)}
        title={`Счёт №${selectedBill?.bill_number}`}
        footer={
          <Button variant="ghost" onClick={() => setSelectedBill(null)}>
            Закрыть
          </Button>
        }
      >
        {selectedBill && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-gray-500">Период:</span>
                <p className="font-medium mt-1">
                  {formatDate(selectedBill.period_start)} - {formatDate(selectedBill.period_end)}
                </p>
              </div>
              <div>
                <span className="text-gray-500">Срок оплаты:</span>
                <p className="font-medium mt-1">{formatDate(selectedBill.due_date)}</p>
              </div>
              <div>
                <span className="text-gray-500">Лицевой счёт:</span>
                <p className="font-medium mt-1">{selectedBill.account_number}</p>
              </div>
              <div>
                <span className="text-gray-500">Статус:</span>
                <p className="mt-1"><StatusBadge status={selectedBill.status} /></p>
              </div>
            </div>

            <div className="border-t border-gray-200 pt-4">
              <h4 className="font-semibold mb-3">Детали начисления</h4>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between py-2 border-b border-gray-100">
                  <span className="text-gray-600">Горячая вода ({selectedBill.hot_water_usage} м³ × {selectedBill.hot_water_tariff} ₽)</span>
                  <span>{selectedBill.hot_water_amount.toFixed(2)} ₽</span>
                </div>
                <div className="flex justify-between py-2 border-b border-gray-100">
                  <span className="text-gray-600">Холодная вода ({selectedBill.cold_water_usage} м³ × {selectedBill.cold_water_tariff} ₽)</span>
                  <span>{selectedBill.cold_water_amount.toFixed(2)} ₽</span>
                </div>
                <div className="flex justify-between py-2 border-b border-gray-100">
                  <span className="text-gray-600">Водоотведение</span>
                  <span>{selectedBill.sewage_amount.toFixed(2)} ₽</span>
                </div>
                <div className="flex justify-between py-2 border-b border-gray-100">
                  <span className="text-gray-600">Обслуживание</span>
                  <span>{selectedBill.service_charge.toFixed(2)} ₽</span>
                </div>
                {selectedBill.maintenance_charge > 0 && (
                  <div className="flex justify-between py-2 border-b border-gray-100">
                    <span className="text-gray-600">Техническое обслуживание</span>
                    <span>{selectedBill.maintenance_charge.toFixed(2)} ₽</span>
                  </div>
                )}
                <div className="flex justify-between py-3 text-lg font-bold bg-gray-50 px-3 rounded-lg">
                  <span>Итого:</span>
                  <span className="text-blue-600">{selectedBill.total_amount.toFixed(2)} ₽</span>
                </div>
              </div>
            </div>

            {selectedBill.paid_at && (
              <div className="bg-green-50 rounded-lg p-3">
                <p className="text-green-800 text-sm">
                  Оплачен {formatDate(selectedBill.paid_at)}
                  {selectedBill.payment_id && ` (платёж №${selectedBill.payment_id})`}
                </p>
              </div>
            )}
          </div>
        )}
      </Modal>

      {/* QR Code Modal */}
      {qrCodeUrl && (
        <Modal
          isOpen={!!qrCodeUrl}
          onClose={() => setQrCodeUrl('')}
          title="Оплата через СБП"
          footer={
            <Button variant="ghost" onClick={() => setQrCodeUrl('')}>
              Закрыть
            </Button>
          }
        >
          <div className="text-center space-y-4">
            <p className="text-gray-600">
              Отсканируйте QR-код в мобильном банке для оплаты
            </p>
            <div className="bg-white p-4 rounded-lg inline-block border">
              <img src={qrCodeUrl} alt="QR Code для оплаты" className="w-64 h-64" />
            </div>
            <p className="text-sm text-gray-500">
              Сумма: <strong>{selectedBill?.total_amount.toFixed(2)} ₽</strong>
            </p>
          </div>
        </Modal>
      )}
    </div>
  )
}
