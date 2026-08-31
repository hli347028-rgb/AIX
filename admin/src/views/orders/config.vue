<template>
    <PageView>
        <a-card title="AIX 配置项">
            <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="false" bordered
                :scroll="{ x: true }">
            </a-table>
        </a-card>

        <a-modal
            title="设置子账户可访问模块"
            :visible="modulesModalVisible"
            :confirmLoading="modulesSaving"
            @ok="saveModules"
            @cancel="modulesModalVisible = false"
            :width="480"
            destroyOnClose
        >
            <div style="margin-bottom:8px;color:#888;font-size:12px;">
                勾选该子账户可访问的模块。不勾选任何项表示可访问全部常规模块（不含操作记录）。
            </div>
            <div style="margin-bottom:12px;">
                <a-button size="small" style="margin-right:8px;" @click="selectAllModules">全选</a-button>
                <a-button size="small" @click="clearModules">清空（=全部模块）</a-button>
            </div>
            <a-checkbox-group v-model="modulesSelected" :options="moduleOptions" class="module-check-group" />
        </a-modal>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'

const MODULE_OPTIONS = [
    { label: '数据统计', value: '/home' },
    { label: '用户数据', value: '/member' },
    { label: '充值列表', value: '/recharge' },
    { label: '提现记录', value: '/withdrawList' },
    { label: '订单列表', value: '/subscription' },
    { label: '订单奖励', value: '/ordersList' },
    { label: '配置项', value: '/config' },
    { label: '兑换记录', value: '/exchangeList' },
    { label: '划转记录', value: '/transferList' },
    { label: '交易所划转', value: '/exchangeTransfer' },
    { label: '每日结算', value: '/settlement' },
    { label: '公告列表', value: '/news' },
]

function isModulesConfigId(id) {
    return id === 102 || id === 104 || id === 106
}

function isPasswordConfigId(id) {
    return id === 101 || id === 103 || id === 105
}

function formatModulesDisplay(raw) {
    const text = String(raw || '').trim()
    if (!text) return '全部常规模块'
    const map = {}
    MODULE_OPTIONS.forEach((item) => { map[item.value] = item.label })
    return text.split(',').map((p) => {
        const path = p.trim()
        return map[path] || path
    }).filter(Boolean).join('、')
}

export default {
    name: 'config',
    mixins: [listMixin],
    data() {
        return {
            moduleOptions: MODULE_OPTIONS,
            modulesModalVisible: false,
            modulesSaving: false,
            modulesConfigId: 0,
            modulesSelected: [],
            columns: [
                {
                    title: '名称',
                    dataIndex: 'name',
                },
                {
                    title: '值',
                    dataIndex: 'value',
                    customRender: (v, row) => {
                        if (isModulesConfigId(row.id)) {
                            return formatModulesDisplay(v)
                        }
                        return v
                    },
                },
                {
                    title: '操作',
                    key: 'action',
                    fixed: 'right',
                    width: 110,
                    customRender: (v) => {
                        return <a-button type="primary" onClick={() => {
                            this.config_update(v.id);
                        }}>修改</a-button>
                    },
                },
            ],
        }
    },
    methods: {
        getList() {
            this.loading = true
            Gai.config().then((res) => {
                const list = (res && res.config) ? res.config : []
                this.data = list.map((value, key) => {
                    return { ...value, key }
                })
                this.loading = false
            }).catch(() => {
                this.loading = false
            })
        },
        selectAllModules() {
            this.modulesSelected = MODULE_OPTIONS.map((item) => item.value)
        },
        clearModules() {
            this.modulesSelected = []
        },
        openModulesModal(id, rawValue) {
            this.modulesConfigId = id
            const text = String(rawValue || '').trim()
            this.modulesSelected = text
                ? text.split(',').map((p) => p.trim()).filter(Boolean)
                : []
            this.modulesModalVisible = true
        },
        saveModules() {
            this.modulesSaving = true
            const value = (this.modulesSelected || []).join(',')
            Gai.config_update({ id: this.modulesConfigId, value }).then(() => {
                this.modulesSaving = false
                this.modulesModalVisible = false
                this.getList()
            }).catch(() => {
                this.modulesSaving = false
            })
        },
        config_update(id) {
            const row = (this.data || []).find((item) => item.id === id)
            let value = row && row.value != null ? String(row.value) : ""

            if (isModulesConfigId(id)) {
                this.openModulesModal(id, value)
                return
            }

            let hint = '静态利率填百分数如 0.5；直推/W 收益系数填小数如 0.2 表示 20%；W 晋级金额为大区与小区共同门槛（须同时达标）且必须逐级递增；出局倍数默认 4；USDT/WIN 充值最小值须 ≥ 10；提现审核阈值填 0 表示不审核；AIX兑换审核阈值填百分数如 40，表示全网当日兑换超过今日AIX的40%后后续需审核（默认100）；交易所划转限额须满足：单笔下限 ≤ 单笔上限 ≤ 单日上限'
            if (isPasswordConfigId(id)) {
                hint = '子账户登录密码，修改后立即生效。'
            }
            this.$confirm({
                title: `修改${row && row.name ? ` - ${row.name}` : ''}`,
                content: (
                    <div>
                        <div style="margin-bottom:8px;color:#888;font-size:12px;">
                            {hint}
                        </div>
                        <a-input style="margin-top:8px;" defaultValue={value} placeholder="请输入" onInput={(val) => {
                            value = val.target.value
                        }} />
                    </div>
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        Gai.config_update({ id, value }).then(() => {
                            resolve()
                            this.getList()
                        }).catch(() => {
                            reject()
                        })
                    })
                }
            })
        }
    },
}
</script>

<style scoped lang="less">
.inputGroup {
    >div {
        margin-bottom: 20px;
    }
}
.module-check-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
}
</style>
