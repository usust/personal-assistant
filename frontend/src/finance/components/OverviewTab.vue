<script setup lang="ts">
import { computed } from 'vue'
import { CreditCard as BankCard, Coin, Money, TrendCharts } from '@element-plus/icons-vue'
import FinanceBarChart from './FinanceBarChart.vue'
import FinanceDonut from './FinanceDonut.vue'
import FinanceLineChart from './FinanceLineChart.vue'
import { money, percent } from '@/finance/utils/money'
import type { FinanceOverview, FinancialAccount } from '@/types/finance'

const props = defineProps<{ overview: FinanceOverview; accounts: FinancialAccount[] }>()
const emit = defineEmits<{ addAccount: []; addTransaction: [] }>()
const cashFlow = computed(() => props.overview.cashFlow.map(item => ({ label: item.month.slice(5) + '月', income: Number(item.income), expense: Number(item.expense), net: Number(item.net) })))
const netWorthTrend = computed(() => props.overview.netWorthTrend.map(item => ({ label: item.date, assets: Number(item.assets), liabilities: Number(item.liabilities), netWorth: Number(item.netWorth) })))
const statusLabels = { normal: '正常', soon: '即将到期', today: '今日到期', overdue: '已逾期' }
const accountTypeLabels: Record<string, string> = { bank: '银行卡', alipay: '支付宝', wechat: '微信', cash: '现金', savings: '储蓄', investment: '投资', other: '其他资产' }
</script>

<template>
  <div class="finance-tab-stack">
    <div class="finance-stat-grid">
      <article class="finance-stat-card"><span class="finance-stat-icon indigo"><Money /></span><div><p>总资产</p><strong>{{ money(overview.totalAssets) }}</strong><small>{{ overview.accountCount }} 个资产账户</small></div></article>
      <article class="finance-stat-card"><span class="finance-stat-icon rose"><BankCard /></span><div><p>总负债</p><strong>{{ money(overview.totalLiabilities) }}</strong><small>负债率 {{ percent(overview.debtRatio) }}</small></div></article>
      <article class="finance-stat-card"><span class="finance-stat-icon green"><TrendCharts /></span><div><p>净资产</p><strong :class="{ negative: Number(overview.netWorth) < 0 }">{{ money(overview.netWorth) }}</strong><small>资产减去全部负债</small></div></article>
      <article class="finance-stat-card cash-flow-card"><span class="finance-stat-icon amber"><Coin /></span><div><p>本月结余</p><strong :class="{ negative: Number(overview.monthBalance) < 0 }">{{ money(overview.monthBalance) }}</strong><small>收入 {{ money(overview.monthIncome) }} · 支出 {{ money(overview.monthExpense) }} · 储蓄率 {{ percent(overview.savingsRate) }}</small></div></article>
    </div>

    <div class="finance-dashboard-grid">
      <article class="finance-panel finance-wide-panel">
        <header class="finance-panel-heading"><div><p>余额快照</p><h3>资产、负债与净资产趋势</h3></div><span class="finance-chart-key"><i class="asset" />资产 <i class="liability" />负债 <i class="networth" />净资产</span></header>
        <FinanceLineChart :data="netWorthTrend" /><p class="chart-source-note">资产趋势来自余额快照；当前版本未保存负债快照，历史点中的负债按当前值展示。</p>
      </article>
      <article class="finance-panel"><header class="finance-panel-heading"><div><p>当前构成</p><h3>资产结构</h3></div></header><FinanceDonut :data="overview.assetStructure" /></article>
    </div>

    <div class="finance-dashboard-grid">
      <article class="finance-panel finance-wide-panel"><header class="finance-panel-heading"><div><p>现金流趋势</p><h3>近 6 个月收支</h3></div><span class="finance-chart-key"><i class="income" />收入 <i class="expense" />支出</span></header><FinanceBarChart :data="cashFlow" /></article>
      <article class="finance-panel">
        <header class="finance-panel-heading"><div><p>未来 30 天与逾期项目</p><h3>近期资金安排</h3></div></header>
        <div v-if="overview.upcoming.length" class="upcoming-list">
          <div v-for="item in overview.upcoming" :key="`${item.kind}-${item.id}`" class="upcoming-row"><time>{{ item.date.slice(5).replace('-', '月') }}日</time><div><strong>{{ item.name }}</strong><span>{{ item.kind === 'credit_card' ? '信用卡' : item.kind === 'mortgage' ? '房贷' : item.kind === 'loan' ? '贷款' : '周期账单' }}</span></div><b>{{ money(item.amount) }}</b><em :class="item.status">{{ statusLabels[item.status] }}</em></div>
        </div>
        <div v-else class="finance-empty compact"><Coin /><strong>未来 30 天暂无固定资金安排</strong><span>添加信用卡、贷款或周期账单后会自动汇总。</span></div>
      </article>
    </div>
    <article class="finance-panel">
        <header class="finance-panel-heading"><div><p>快速查看</p><h3>账户概览</h3></div><el-button link type="primary" @click="emit('addAccount')">添加账户</el-button></header>
        <div v-if="accounts.length" class="account-mini-list"><div v-for="item in accounts.slice(0, 5)" :key="item.id"><span><i :class="item.accountType" />{{ item.name }}<small>{{ accountTypeLabels[item.accountType] }}</small></span><b>{{ money(item.balance) }}</b></div></div>
        <div v-else class="finance-empty compact"><Money /><strong>暂无财务账户</strong><span>添加账户后即可查看资产趋势。</span><el-button type="primary" @click="emit('addAccount')">添加账户</el-button></div>
    </article>
  </div>
</template>
