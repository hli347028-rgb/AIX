<template>
    <PageView>
        <a-card :title="`订单列表（共 ${total} 条）`">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="用户地址" allowClear @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="8" :lg="8" :xl="5">
                    <a-radio-group v-model="searchData.teamQuery" button-style="solid" @change="getListTwo">
                        <a-radio-button :value="false">查本人</a-radio-button>
                        <a-radio-button :value="true">查全团队</a-radio-button>
                    </a-radio-group>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.status" style="width:100%" placeholder="状态"
                        @change="getListTwo">
                        <a-select-option value="active">进行中</a-select-option>
                        <a-select-option value="exited">已出局</a-select-option>
                    </a-select>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.fund_source" style="width:100%" placeholder="资金来源"
                        @change="getListTwo">
                        <a-select-option v-for="opt in fundSourceOptions" :key="opt.value" :value="opt.value">
                            {{ opt.label }}
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
                <a-col :xs="24" :md="12" :lg="12" :xl="8">
                    <a-button-group>
                        <a-button type="primary" :loading="loading" @click="getListTwo">确定筛选</a-button>
                    </a-button-group>
                </a-col>
            </a-row>
            <div class="stats-bar stats-bar-team" v-if="teamSummary && searchData.teamQuery">
                <span>团队概览：<b>{{ teamSummaryText(teamSummary) }}</b></span>
            </div>
            <div class="stats-bar" v-if="stats">
                <span>筛选笔数：<b>{{ stats.totalCount || 0 }}</b></span>
                <span>报单总额：<b>{{ formatAmount4(stats.principalTotal) }}</b> USDT</span>
            </div>
            <a-table
                rowKey="id"
                :loading="loading"
                :columns="columns"
                :dataSource="data"
                :pagination="{ total, pageSize, current, showSizeChanger: true, pageSizeOptions: ['20', '50', '100', '200'] }"
                @change="changePagination"
                bordered
                :scroll="{ x: true }"
            />
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'
import teamQueryMixin from '../mixin/teamQueryMixin'
import { formatAmount4 } from '../../utils/formatAmount'
import moment from 'moment'

const statusText = {
    active: '进行中',
    exited: '已出局',
    completed: '已出局',
}

const fundSourceText = {
    recharge: '充值账本',
    reward: '奖励账本',
    win: 'WIN',
    win_a: 'WIN-A',
    'recharge+win': '充值+WIN',
    'recharge+win_a': '充值+WIN-A',
    'win+win_a': 'WIN+WIN-A',
    'win+recharge': 'WIN+充值',
    'win_a+recharge': 'WIN-A+充值',
    'win_a+win': 'WIN-A+WIN',
}

const pointsSourceText = {
    recharge: 'USDT认购',
    win: 'WIN认购',
    transfer_reinvest: '复投（上级划转，含隔代）',
}

const fundSourceOptions = Object.keys(fundSourceText).map((value) => ({
    value,
    label: fundSourceText[value],
}))

export default {
    name: 'subscription',
    mixins: [listMixin, teamQueryMixin],
    data() {
        return {
            stats: null,
            teamSummary: null,
            fundSourceOptions,
            columns: [
                {
                    title: '时间',
                    dataIndex: 'createdAt',
                },
                {
                    title: '报单本金',
                    dataIndex: 'amount',
                },
                {
                    title: '出局倍数',
                    dataIndex: 'exitAmount',
                },
                {
                    title: '出局目标',
                    dataIndex: 'money',
                },
                {
                    title: '已获收益',
                    dataIndex: 'amountGet',
                },
                {
                    title: '剩余额度',
                    dataIndex: 'amountLast',
                },
                {
                    title: '积分',
                    dataIndex: 'points',
                    customRender: (v) => v || '0',
                },
                {
                    title: 'AIX-USDT来源',
                    dataIndex: 'points_source',
                    customRender: (v, row) => {
                        if (row && row.points_source_label) return row.points_source_label
                        return pointsSourceText[v] || (v ? v : '-')
                    },
                },
                {
                    title: '资金来源',
                    dataIndex: 'fund_source',
                    customRender: (v) => fundSourceText[v] || v || '-',
                },
                {
                    title: '用户地址',
                    dataIndex: 'address',
                },
                {
                    title: '状态',
                    dataIndex: 'status',
                    customRender: (v) => statusText[v] || v || '-',
                },
            ],
            searchData: {
                address: '',
                status: undefined,
                fund_source: undefined,
                teamQuery: false,
                dateRange: [],
            },
            pageSize: 50,
        }
    },
    mounted() {
        this.getList()
    },
    methods: {
        buildParams() {
            const params = this.appendTeamQueryParams({
                page: this.current || 1,
                pageSize: this.pageSize || 50,
            })
            const address = (this.searchData.address || '').trim()
            if (address) params.address = address
            const status = this.searchData.status
            if (status) params.status = status
            const fundSource = this.searchData.fund_source
            if (fundSource) params.fund_source = fundSource
            if (this.searchData.dateRange && this.searchData.dateRange.length === 2) {
                params.startTime = moment(this.searchData.dateRange[0]).format('YYYY-MM-DD HH:mm:ss')
                params.endTime = moment(this.searchData.dateRange[1]).format('YYYY-MM-DD HH:mm:ss')
            }
            return params
        },
        getList() {
            this.loading = true
            Gai.buy_list(this.buildParams()).then((res) => {
                const list = (res && res.rewards) ? res.rewards : []
                this.data = list.map((value, key) => ({
                    ...value,
                    id: value.id != null ? value.id : key,
                    key: value.id != null ? value.id : key,
                }))
                this.total = parseInt((res && res.count) || 0, 10) || 0
                this.stats = (res && res.stats) || null
                this.teamSummary = (res && res.teamSummary) || null
            }).catch(() => {
                this.data = []
                this.total = 0
                this.stats = null
            }).finally(() => {
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
    padding: 12px 14px;
    background: #f6ffed;
    border: 1px solid #b7eb8f;
    border-radius: 4px;
    color: #333;

    b {
        color: #389e0d;
    }
}

.stats-bar-team {
    background: #fff7e6;
    border-color: #ffd591;

    b {
        color: #d46b08;
    }
}
</style>
