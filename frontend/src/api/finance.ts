import { http } from './http'
import type { ApiResponse } from '@/types/api'
import type { CreditCard, FinanceOverview, FinanceTransaction, FinancialAccount, Loan, Mortgage, MortgageInput, MortgageResult, TransactionCategory, TransactionFilter } from '@/types/finance'

async function getData<T>(request: Promise<{ data: ApiResponse<T> }>) { return (await request).data.data }

export const getFinanceOverview = () => getData<FinanceOverview>(http.get('/finance/overview'))
export const getAccounts = () => getData<FinancialAccount[]>(http.get('/finance/accounts'))
export const createAccount = (payload: Partial<FinancialAccount>) => getData<FinancialAccount>(http.post('/finance/accounts', payload))
export const updateAccount = (id: number, payload: Partial<FinancialAccount>) => getData(http.patch(`/finance/accounts/${id}`, payload))
export const getAccountImpact = (id: number) => getData<{ transactions: number; snapshots: number }>(http.get(`/finance/accounts/${id}/impact`))
export const archiveAccount = (id: number) => getData<{ transactions: number; snapshots: number }>(http.delete(`/finance/accounts/${id}`))

export const getTransactions = (params: TransactionFilter = {}) => getData<FinanceTransaction[]>(http.get('/finance/transactions', { params }))
export const createTransaction = (payload: Partial<FinanceTransaction>) => getData<FinanceTransaction>(http.post('/finance/transactions', payload))
export const getCategories = () => getData<TransactionCategory[]>(http.get('/finance/categories'))
export const createCategory = (payload: Partial<TransactionCategory>) => getData<TransactionCategory>(http.post('/finance/categories', payload))

export const getCreditCards = () => getData<CreditCard[]>(http.get('/finance/credit-cards'))
export const createCreditCard = (payload: Partial<CreditCard>) => getData<CreditCard>(http.post('/finance/credit-cards', payload))
export const getLoans = () => getData<Loan[]>(http.get('/finance/loans'))
export const createLoan = (payload: Partial<Loan>) => getData<Loan>(http.post('/finance/loans', payload))
export const getMortgages = () => getData<Mortgage[]>(http.get('/finance/mortgages'))
export const createMortgage = (payload: Partial<Mortgage>) => getData<Mortgage>(http.post('/finance/mortgages', payload))
export const calculateMortgage = (payload: MortgageInput) => getData<MortgageResult>(http.post('/finance/mortgage/calculate', payload))
export const simulatePrepayment = (payload: MortgageInput & { afterPeriod: number; prepaymentAmount: string; type: 'partial' | 'settle'; strategy: 'shorten_term' | 'lower_payment' }) => getData(http.post('/finance/mortgage/prepayment', payload))
