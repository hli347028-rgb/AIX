<template>
    <PageView>
        <a-card title="订单奖励">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="账户" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.type" style="width:100%" placeholder="类型"
                        @change="getListTwo">
                        <a-select-option v-for="item in typeOptions" :key="item.value" :value="item.value">
                            {{ item.label }}
                        </a-select-option>
                    </a-select>
                </a-col>
                <a-col :xs="24" :md="12" :lg="10" :xl="8">
                    <a-range-picker
                        v-model="searchData.dateRange"
                        show-time
                        format="YYYY-MM-DD HH:mm:ss"
                        style="width:100%"
                        :placeholder="['开始时间', '结束时间']"
                    />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-button-group>
                        <a-button type="primary" :loading="loading" @click="getListTwo">确定筛选</a-button>
                        <a-button :loading="loading" @click="resetSearch">重置</a-button>
                    </a-button-group>
                </a-col>
            </a-row>
            <div class="stats-bar stats-bar-total" v-if="stats">
                <span>筛选合计：<b>{{ filteredTotalText }}</b></span>
                <span>总笔数：<b>{{ stats.totalCount || 0 }}</b></span>
            </div>
            <div class="stats-bar" v-if="stats">
                <span>静态奖(AIX)：<b>{{ stats.staticAixTotal || 0 }}</b></span>
                <span>直推奖(USDT)：<b>{{ stats.dynamicTotal || 0 }}</b></span>
                <span>管理奖(USDT)：<b>{{ stats.mgmtTotal || 0 }}</b></span>
                <span>零号账户(USDT)：<b>{{ stats.zeroAccountTotal || 0 }}</b></span>
                <span>社区补贴(USDT)：<b>{{ stats.communitySubsidyTotal || 0 }}</b></span>
            </div>
            <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="{ total, pageSize, current }"
                @change="changePagination" bordered :scroll="{ x: true }">
            </a-table>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'
import moment from 'moment'

const typeLabel = {
    static_aix: '静态奖(AIX)',
    dynamic_usdt: '直推奖(USDT)',
    direct_pool_release: '直推奖(USDT)',
    mgmt: '管理奖(USDT)',
    mgmt_pool_release: '管理奖(USDT)',
    mgmt_overflow: '管理奖(USDT)',
    exit_accel: '出局加速',
    transfer_in: '转入',
    transfer_out: '转出',
    zero_account: '零号账户(USDT)',
    community_subsidy: '社区补贴(USDT)',
}

const typeOptions = [
    { value: 'static_aix', label: '静态奖(AIX)' },
    { value: 'dynamic_usdt', label: '直推奖(USDT)' },
    { value: 'mgmt', label: '管理奖(USDT)' },
    { value: 'zero_account', label: '零号账户(USDT)' },
    { value: 'community_subsidy', label: '社区补贴(USDT)' },
    { value: 'exit_accel', label: '出局加速' },
    { value: 'transfer_in', label: '转入' },
    { value: 'transfer_out', label: '转出' },
]

export default {
    name: 'ordersList',
    mixins: [listMixin],
    data() {
        return {
            typeOptions,
            stats: null,
            columns: [
                {
                    title: '时间',
                    dataIndex: 'createdAt',
                },
                {
                    title: '类型',
                    dataIndex: 'type',
                    customRender: (v) => typeLabel[v] || v || '-',
                },
                {
                    title: '资产',
                    dataIndex: 'asset',
                },
                {
                    title: '金额',
                    dataIndex: 'amount',
                },
                {
                    title: '地址',
                    dataIndex: 'address',
                    customRender: (v) => v || '-',
                },
                {
                    title: '来源地址',
                    dataIndex: 'addressTwo',
                    customRender: (v) => v || '-',
                },
                {
                    title: '结算日',
                    dataIndex: 'settlementDate',
                    customRender: (v) => v || '-',
                },
            ],
            searchData: {
                address: '',
                type: undefined,
                dateRange: [],
            },
        }
    },
    computed: {
        filteredTotalText() {
            const list = (this.stats && this.stats.assetTotals) || []
            if (!list.length) return '0'
            return list.map((v) => `${v.total} ${v.asset}`).join('  |  ')
        },
    },
    methods: {
        buildParams() {
            const params = {
                page: this.current,
                pageSize: this.pageSize,
                address: this.searchData.address,
            }
            if (this.searchData.type) {
                params.type = this.searchData.type
            }
            if (this.searchData.dateRange && this.searchData.dateRange.length === 2) {
                params.startTime = moment(this.searchData.dateRange[0]).format('YYYY-MM-DD HH:mm:ss')
                params.endTime = moment(this.searchData.dateRange[1]).format('YYYY-MM-DD HH:mm:ss')
            }
            return params
        },
        resetSearch() {
            this.searchData.address = ''
            this.searchData.type = undefined
            this.searchData.dateRange = []
            this.getListTwo()
        },
        getList() {
            this.loading = true
            Gai.reward_list(this.buildParams()).then((res) => {
                const list = (res && (res.rewards || res.list)) || []
                this.data = list.map((value, key) => {
                    return { ...value, key }
                })
                this.stats = (res && res.stats) || null
                this.loading = false
                this.total = parseInt(res.count || 0)
            }).catch(() => {
                this.loading = false
            })
        },
    },
}
</script>

<style scoped lang="less">
.inputGroup {
    >div {
        margin-bottom: 20px;
    }
}
.stats-bar {
    display: flex;
    flex-wrap: wrap;
    gap: 16px 24px;
    margin-bottom: 16px;
    padding: 12px 16px;
    background: #fafafa;
    border: 1px solid #f0f0f0;
    border-radius: 4px;
    span {
        color: #666;
        b {
            color: #1890ff;
            font-weight: 600;
        }
    }
}
.stats-bar-total {
    margin-bottom: 8px;
    background: #e6f7ff;
    border-color: #91d5ff;
    b {
        font-size: 16px;
    }
}
</style>
