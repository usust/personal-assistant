<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { CreditCard as BankCard, Coin, Delete, Edit, Plus, Wallet } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { archiveAccount, createAccount, getAccountImpact, updateAccount } from '@/api/finance'
import { money, sumMoney } from '@/finance/utils/money'
import type { AccountType, FinancialAccount } from '@/types/finance'

const props = defineProps<{ accounts: FinancialAccount[] }>()
const emit = defineEmits<{ changed: [] }>()
const visible = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const typeLabels: Record<AccountType, string> = { bank: '银行卡', alipay: '支付宝', wechat: '微信支付', cash: '现金', savings: '储蓄', investment: '其他资产', other: '其他' }
const typeIcons: Record<AccountType, typeof Wallet> = { bank: BankCard, alipay: Wallet, wechat: Wallet, cash: Coin, savings: BankCard, investment: Coin, other: Wallet }
const draft = reactive({ name: '', accountType: 'bank' as AccountType, institution: '', maskedAccountNumber: '', balance: '0.00', availableBalance: '0.00', includeInNetWorth: true, notes: '' })
const total = computed(() => sumMoney(props.accounts.filter(item => item.includeInNetWorth).map(item => item.balance)))
const grouped = computed(() => Object.entries(typeLabels).map(([type, label]) => ({ type: type as AccountType, label, items: props.accounts.filter(item => item.accountType === type) })).filter(group => group.items.length))

function openCreate() { editingId.value = null; Object.assign(draft, { name: '', accountType: 'bank', institution: '', maskedAccountNumber: '', balance: '0.00', availableBalance: '0.00', includeInNetWorth: true, notes: '' }); visible.value = true }
function openEdit(item: FinancialAccount) { editingId.value = item.id; Object.assign(draft, { name: item.name, accountType: item.accountType, institution: item.institution, maskedAccountNumber: item.maskedAccountNumber, balance: item.balance, availableBalance: item.availableBalance, includeInNetWorth: item.includeInNetWorth, notes: item.notes }); visible.value = true }
async function save() { if (!draft.name.trim()) { ElMessage.warning('请输入账户名称'); return }; saving.value = true; try { editingId.value ? await updateAccount(editingId.value, draft) : await createAccount(draft); ElMessage.success(editingId.value ? '账户已更新，余额历史已保留' : '账户已创建'); visible.value = false; emit('changed') } finally { saving.value = false } }
async function remove(item: FinancialAccount) { const impact = await getAccountImpact(item.id); await ElMessageBox.confirm(`该账户关联 ${impact.transactions} 条流水、${impact.snapshots} 条余额历史。归档后历史数据仍会保留。`, `归档“${item.name}”`, { type: 'warning', confirmButtonText: '确认归档', cancelButtonText: '取消' }); await archiveAccount(item.id); ElMessage.success('账户已归档'); emit('changed') }
defineExpose({ openCreate })
</script>

<template>
  <div class="finance-tab-stack">
    <div class="finance-inline-summary"><div><span>总资产</span><strong>{{ money(total) }}</strong></div><div><span>账户数量</span><strong>{{ accounts.length }}</strong></div><div><span>计入净资产</span><strong>{{ accounts.filter(item => item.includeInNetWorth).length }}</strong></div><el-button type="primary" :icon="Plus" @click="openCreate">添加账户</el-button></div>
    <div v-if="accounts.length" class="account-groups">
      <section v-for="group in grouped" :key="group.type"><header><h3>{{ group.label }}</h3><span>{{ group.items.length }} 个账户</span></header><div class="account-card-grid">
        <article v-for="item in group.items" :key="item.id" class="account-card"><div class="account-card-top"><span class="account-brand-icon"><component :is="typeIcons[item.accountType]" /></span><div><strong>{{ item.name }}</strong><small>{{ item.institution || typeLabels[item.accountType] }} {{ item.maskedAccountNumber }}</small></div><el-dropdown trigger="click"><button class="account-more">•••</button><template #dropdown><el-dropdown-menu><el-dropdown-item :icon="Edit" @click="openEdit(item)">编辑</el-dropdown-item><el-dropdown-item :icon="Delete" divided @click="remove(item)">归档</el-dropdown-item></el-dropdown-menu></template></el-dropdown></div><p>当前余额</p><b>{{ money(item.balance) }}</b><footer><span>{{ item.includeInNetWorth ? '计入净资产' : '不计入净资产' }}</span><time>更新于 {{ item.updatedAt.slice(0, 10) }}</time></footer></article>
      </div></section>
    </div>
    <div v-else class="finance-panel finance-empty"><Wallet /><strong>暂无财务账户</strong><span>添加银行卡、支付宝、微信或其他账户后，即可查看资产和净资产趋势。</span><el-button type="primary" @click="openCreate">添加账户</el-button></div>

    <el-dialog v-model="visible" :title="editingId ? '编辑账户' : '添加账户'" width="560px" destroy-on-close>
      <el-form label-position="top"><div class="finance-form-grid"><el-form-item label="账户名称" required><el-input v-model="draft.name" placeholder="如：招商银行工资卡" /></el-form-item><el-form-item label="账户类型" required><el-select v-model="draft.accountType"><el-option v-for="(label, value) in typeLabels" :key="value" :label="label" :value="value" /></el-select></el-form-item><el-form-item label="银行 / 平台"><el-input v-model="draft.institution" placeholder="如：招商银行" /></el-form-item><el-form-item label="卡号尾号"><el-input v-model="draft.maskedAccountNumber" maxlength="19" placeholder="只需填写尾号，系统会自动脱敏" /></el-form-item><el-form-item label="当前余额" required><el-input v-model="draft.balance" inputmode="decimal"><template #prepend>￥</template></el-input></el-form-item><el-form-item label="可用余额"><el-input v-model="draft.availableBalance" inputmode="decimal"><template #prepend>￥</template></el-input></el-form-item></div><el-form-item><el-checkbox v-model="draft.includeInNetWorth">计入净资产统计</el-checkbox></el-form-item><el-form-item label="备注"><el-input v-model="draft.notes" type="textarea" :rows="3" /></el-form-item></el-form>
      <template #footer><el-button @click="visible = false">取消</el-button><el-button type="primary" :loading="saving" @click="save">保存</el-button></template>
    </el-dialog>
  </div>
</template>
