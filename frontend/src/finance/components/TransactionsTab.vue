<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { Plus, Search, Tickets } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { createCategory, createTransaction, getTransactions } from '@/api/finance'
import { money, sumMoney, today } from '@/finance/utils/money'
import type { CreditCard, FinanceTransaction, FinancialAccount, TransactionCategory, TransactionFilter, TransactionType } from '@/types/finance'

const props = defineProps<{ accounts: FinancialAccount[]; categories: TransactionCategory[]; cards: CreditCard[] }>()
const emit = defineEmits<{ changed: []; categoriesChanged: [] }>()
const items = ref<FinanceTransaction[]>([])
const loading = ref(false)
const visible = ref(false)
const saving = ref(false)
const categoryVisible = ref(false)
const filter = reactive<TransactionFilter>({ startDate: '', endDate: '', type: '', accountId: undefined, categoryId: undefined, minAmount: '', maxAmount: '', keyword: '' })
const draft = reactive({ accountId: 0, targetAccountId: undefined as number | undefined, targetCreditCardId: undefined as number | undefined, type: 'expense' as TransactionType, amount: '', categoryId: undefined as number | undefined, counterparty: '', transactionDate: today(), description: '', tags: '' })
const categoryDraft = reactive({ name: '', type: 'expense' as 'income' | 'expense', color: '#5658cf' })
const accountMap = computed(() => Object.fromEntries(props.accounts.map(item => [item.id, item.name])))
const categoryMap = computed(() => Object.fromEntries(props.categories.map(item => [item.id, item.name])))
const visibleCategories = computed(() => props.categories.filter(item => item.type === draft.type))
const income = computed(() => sumMoney(items.value.filter(item => item.type === 'income').map(item => item.amount)))
const expense = computed(() => sumMoney(items.value.filter(item => item.type === 'expense').map(item => item.amount)))
const net = computed(() => sumMoney([income.value, `-${expense.value}`]))

async function load() { loading.value = true; try { items.value = await getTransactions(filter) } finally { loading.value = false } }
function openCreate() { Object.assign(draft, { accountId: props.accounts[0]?.id ?? 0, targetAccountId: undefined, targetCreditCardId: undefined, type: 'expense', amount: '', categoryId: undefined, counterparty: '', transactionDate: today(), description: '', tags: '' }); visible.value = true }
async function save() { if (!draft.accountId || !draft.amount || !draft.transactionDate) { ElMessage.warning('请填写账户、金额和日期'); return }; if (draft.type === 'transfer' && !draft.targetAccountId && !draft.targetCreditCardId) { ElMessage.warning('请选择转入账户或还款信用卡'); return }; saving.value = true; try { await createTransaction({ ...draft, categoryId: draft.type === 'transfer' ? null : (draft.categoryId ?? null), targetAccountId: draft.targetAccountId ?? null, targetCreditCardId: draft.targetCreditCardId ?? null }); ElMessage.success(draft.type === 'transfer' ? '转账已记录，不计入收支' : '流水已记录'); visible.value = false; await load(); emit('changed') } finally { saving.value = false } }
async function saveCategory() { if (!categoryDraft.name.trim()) return; await createCategory(categoryDraft); ElMessage.success('分类已添加'); categoryVisible.value = false; categoryDraft.name = ''; emit('categoriesChanged') }
watch(() => draft.type, () => { draft.categoryId = undefined; draft.targetAccountId = undefined; draft.targetCreditCardId = undefined })
watch(() => props.accounts.length, () => { if (!draft.accountId) draft.accountId = props.accounts[0]?.id ?? 0 }, { immediate: true })
load()
defineExpose({ openCreate, load })
</script>

<template>
  <div class="finance-tab-stack">
    <div class="finance-filter-panel">
      <div class="finance-filters"><el-date-picker v-model="filter.startDate" value-format="YYYY-MM-DD" type="date" placeholder="开始日期" /><el-date-picker v-model="filter.endDate" value-format="YYYY-MM-DD" type="date" placeholder="结束日期" /><el-select v-model="filter.accountId" clearable placeholder="全部账户"><el-option v-for="item in accounts" :key="item.id" :label="item.name" :value="item.id" /></el-select><el-select v-model="filter.type" placeholder="全部类型"><el-option label="全部类型" value="" /><el-option label="收入" value="income" /><el-option label="支出" value="expense" /><el-option label="转账" value="transfer" /></el-select><el-select v-model="filter.categoryId" clearable placeholder="全部分类"><el-option v-for="item in categories" :key="item.id" :label="item.name" :value="item.id" /></el-select><el-input v-model="filter.minAmount" placeholder="最低金额" /><el-input v-model="filter.maxAmount" placeholder="最高金额" /><el-input v-model="filter.keyword" class="keyword-filter" clearable placeholder="商户、说明或标签"><template #prefix><Search /></template></el-input><el-button @click="load">筛选</el-button><el-button type="primary" :icon="Plus" @click="openCreate">记录流水</el-button></div>
      <div class="transaction-summary"><span>筛选结果</span><div><small>收入</small><b class="positive">+{{ money(income) }}</b></div><div><small>支出</small><b class="negative">-{{ money(expense) }}</b></div><div><small>净流入</small><b :class="Number(net) >= 0 ? 'positive' : 'negative'">{{ money(net) }}</b></div></div>
    </div>
    <article class="finance-panel finance-table-panel">
      <el-table v-if="items.length || loading" v-loading="loading" :data="items" table-layout="fixed"><el-table-column prop="transactionDate" label="日期" width="115" /><el-table-column label="账户" min-width="130"><template #default="scope">{{ accountMap[scope.row.accountId] || '已归档账户' }}</template></el-table-column><el-table-column label="类型" width="88"><template #default="scope"><span class="transaction-type" :class="scope.row.type">{{ scope.row.type === 'income' ? '收入' : scope.row.type === 'expense' ? '支出' : '转账' }}</span></template></el-table-column><el-table-column label="分类" width="110"><template #default="scope">{{ scope.row.type === 'transfer' ? '内部转账' : categoryMap[scope.row.categoryId] || '未分类' }}</template></el-table-column><el-table-column label="商户 / 对方" min-width="140"><template #default="scope">{{ scope.row.counterparty || '--' }}</template></el-table-column><el-table-column prop="description" label="说明" min-width="180" show-overflow-tooltip /><el-table-column label="金额" width="140" align="right"><template #default="scope"><strong :class="scope.row.type === 'income' ? 'positive' : scope.row.type === 'expense' ? 'negative' : ''">{{ scope.row.type === 'income' ? '+' : scope.row.type === 'expense' ? '-' : '' }}{{ money(scope.row.amount) }}</strong></template></el-table-column></el-table>
      <div v-else class="finance-empty"><Tickets /><strong>暂无收支流水</strong><span>记录第一笔收入、支出或内部转账，开始形成现金流分析。</span><el-button type="primary" @click="openCreate">记录流水</el-button></div>
    </article>

    <el-dialog v-model="visible" title="记录流水" width="580px"><el-form label-position="top"><el-form-item label="类型" required><el-radio-group v-model="draft.type"><el-radio-button value="expense">支出</el-radio-button><el-radio-button value="income">收入</el-radio-button><el-radio-button value="transfer">转账 / 信用卡还款</el-radio-button></el-radio-group></el-form-item><div class="finance-form-grid"><el-form-item :label="draft.type === 'transfer' ? '转出账户' : '账户'" required><el-select v-model="draft.accountId"><el-option v-for="item in accounts" :key="item.id" :label="`${item.name} ${item.maskedAccountNumber}`" :value="item.id" /></el-select></el-form-item><el-form-item label="金额" required><el-input v-model="draft.amount" inputmode="decimal"><template #prepend>￥</template></el-input></el-form-item><template v-if="draft.type === 'transfer'"><el-form-item label="转入账户"><el-select v-model="draft.targetAccountId" clearable :disabled="Boolean(draft.targetCreditCardId)" placeholder="普通内部转账"><el-option v-for="item in accounts.filter(item => item.id !== draft.accountId)" :key="item.id" :label="item.name" :value="item.id" /></el-select></el-form-item><el-form-item label="或信用卡还款"><el-select v-model="draft.targetCreditCardId" clearable :disabled="Boolean(draft.targetAccountId)" placeholder="选择待还信用卡"><el-option v-for="item in cards" :key="item.id" :label="item.name" :value="item.id" /></el-select></el-form-item></template><template v-else><el-form-item label="分类"><div class="category-field"><el-select v-model="draft.categoryId" clearable><el-option v-for="item in visibleCategories" :key="item.id" :label="item.name" :value="item.id" /></el-select><el-button @click="categoryDraft.type = draft.type as 'income' | 'expense'; categoryVisible = true">自定义</el-button></div></el-form-item><el-form-item label="商户 / 对方"><el-input v-model="draft.counterparty" /></el-form-item></template><el-form-item label="交易日期" required><el-date-picker v-model="draft.transactionDate" value-format="YYYY-MM-DD" type="date" /></el-form-item><el-form-item label="标签"><el-input v-model="draft.tags" placeholder="多个标签用逗号分隔" /></el-form-item></div><el-form-item label="说明"><el-input v-model="draft.description" type="textarea" :rows="2" /></el-form-item><el-alert v-if="draft.type === 'transfer'" title="内部转账和信用卡还款只改变账户余额，不计入收入或支出。" type="info" :closable="false" show-icon /></el-form><template #footer><el-button @click="visible = false">取消</el-button><el-button type="primary" :loading="saving" @click="save">保存</el-button></template></el-dialog>
    <el-dialog v-model="categoryVisible" title="添加自定义分类" width="420px"><el-form label-position="top"><el-form-item label="分类名称"><el-input v-model="categoryDraft.name" /></el-form-item><el-form-item label="类型"><el-radio-group v-model="categoryDraft.type"><el-radio value="income">收入</el-radio><el-radio value="expense">支出</el-radio></el-radio-group></el-form-item></el-form><template #footer><el-button @click="categoryVisible = false">取消</el-button><el-button type="primary" @click="saveCategory">添加</el-button></template></el-dialog>
  </div>
</template>
