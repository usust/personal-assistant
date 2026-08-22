<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Plus, Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getAccounts, getCategories, getCreditCards, getFinanceOverview, getLoans, getMortgages } from '@/api/finance'
import OverviewTab from '@/finance/components/OverviewTab.vue'
import AccountsTab from '@/finance/components/AccountsTab.vue'
import TransactionsTab from '@/finance/components/TransactionsTab.vue'
import DebtTab from '@/finance/components/DebtTab.vue'
import MortgageTab from '@/finance/components/MortgageTab.vue'
import AnalysisTab from '@/finance/components/AnalysisTab.vue'
import type { CreditCard, FinanceOverview, FinancialAccount, Loan, Mortgage, TransactionCategory } from '@/types/finance'

const activeTab = ref('overview')
const loading = ref(true)
const accounts = ref<FinancialAccount[]>([])
const categories = ref<TransactionCategory[]>([])
const cards = ref<CreditCard[]>([])
const loans = ref<Loan[]>([])
const mortgages = ref<Mortgage[]>([])
const overview = ref<FinanceOverview>({ totalAssets: '0.00', totalLiabilities: '0.00', netWorth: '0.00', monthIncome: '0.00', monthExpense: '0.00', monthBalance: '0.00', savingsRate: null, debtRatio: null, accountCount: 0, assetStructure: [], netWorthTrend: [], cashFlow: [], expenseCategories: [], upcoming: [] })
const accountsTab = ref<InstanceType<typeof AccountsTab>>()
const transactionsTab = ref<InstanceType<typeof TransactionsTab>>()

async function loadAll(showMessage = false) {
  loading.value = true
  try {
    const [overviewData, accountData, categoryData, cardData, loanData, mortgageData] = await Promise.all([getFinanceOverview(), getAccounts(), getCategories(), getCreditCards(), getLoans(), getMortgages()])
    overview.value = overviewData; accounts.value = accountData; categories.value = categoryData; cards.value = cardData; loans.value = loanData; mortgages.value = mortgageData
    if (showMessage) ElMessage.success('财务数据已刷新')
  } finally { loading.value = false }
}
async function refreshCategories() { categories.value = await getCategories() }
function addAccount() { activeTab.value = 'accounts'; requestAnimationFrame(() => accountsTab.value?.openCreate()) }
function addTransaction() { activeTab.value = 'transactions'; requestAnimationFrame(() => transactionsTab.value?.openCreate()) }
onMounted(loadAll)
</script>

<template>
  <div class="finance-page" v-loading="loading">
    <div class="finance-page-header"><div><p class="eyebrow">PERSONAL FINANCE</p><h1>财务管理</h1><span>掌握资产、负债和每一笔现金流</span></div><div><el-button :icon="Refresh" @click="loadAll(true)">刷新</el-button><el-button type="primary" :icon="Plus" @click="addTransaction">记录流水</el-button></div></div>
    <nav class="finance-tabs"><button v-for="tab in [{ key: 'overview', label: '总览' }, { key: 'accounts', label: '账户' }, { key: 'transactions', label: '流水' }, { key: 'debt', label: '负债' }, { key: 'mortgage', label: '房贷' }, { key: 'analysis', label: '分析' }]" :key="tab.key" :class="{ active: activeTab === tab.key }" @click="activeTab = tab.key">{{ tab.label }}</button></nav>
    <OverviewTab v-if="activeTab === 'overview'" :overview="overview" :accounts="accounts" @add-account="addAccount" @add-transaction="addTransaction" />
    <AccountsTab v-else-if="activeTab === 'accounts'" ref="accountsTab" :accounts="accounts" @changed="loadAll" />
    <TransactionsTab v-else-if="activeTab === 'transactions'" ref="transactionsTab" :accounts="accounts" :categories="categories" :cards="cards" @changed="loadAll" @categories-changed="refreshCategories" />
    <DebtTab v-else-if="activeTab === 'debt'" :cards="cards" :loans="loans" @changed="loadAll" />
    <MortgageTab v-else-if="activeTab === 'mortgage'" :mortgages="mortgages" @changed="loadAll" />
    <AnalysisTab v-else :overview="overview" :cards="cards" :loans="loans" :mortgages="mortgages" />
  </div>
</template>
