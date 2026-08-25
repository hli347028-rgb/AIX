<template>
    <PageView>
        <a-card title="提现记录">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="账户地址" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select v-model="searchData.asset" placeholder="资产类型" allowClear style="width: 100%">
                        <a-select-option value="">全部</a-select-option>
                        <a-select-option value="WIN">WIN</a-select-option>
                        <a-select-option value="SDT">AIX-USDT</a-select-option>
                        <a-select-option value="USDT">USDT</a-select-option>
                    </a-select>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select v-model="searchData.status" placeholder="提现状态" allowClear style="width: 100%" @change="getListTwo">
                        <a-select-option value="">全部状态</a-select-option>
                        <a-select-option value="review">待审核</a-select-option>
                        <a-select-option value="pending">待打款</a-select-option>
                        <a-select-option value="doing">处理中</a-select-option>
                        <a-select-option value="completed">已转账</a-select-option>
                        <a-select-option value="rejected">已拒绝</a-select-option>
                        <a-select-option value="failed">失败</a-select-option>
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
                    </a-button-group>
                </a-col>
            </a-row>
            <div class="stats-bar" v-if="stats">
                <span>WIN提现总额：<b>{{ stats.winTotal || 0 }}</b>（{{ stats.winCount || 0 }} 笔）</span>
                <span>AIX-USDT提现总额：<b>{{ stats.sdtTotal || 0 }}</b>（{{ stats.sdtCount || 0 }} 笔）</span>
                <span>USDT提现总额：<b>{{ stats.usdtTotal || 0 }}</b>（{{ stats.usdtCount || 0 }} 笔）</span>
                <span>待审核：<b>{{ stats.reviewCount || 0 }}</b></span>
            </div>
            <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="{ total, pageSize, current, showSizeChanger }"
                @change="changePagination" bordered :scroll="{ x: true }">
            </a-table>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'
import moment from 'moment'

export default {
    name: 'withdrawList',
    mixins: [listMixin],
    data() {
        return {
            pageSize: 20,
            stats: null,
            columns: [
                {
                    title: '账户',
                    dataIndex: 'address',
                },
                {
                    title: '资产',
                    dataIndex: 'asset',
                    customRender: (v) => ({
                        WIN: 'WIN',
                        SDT: 'AIX-USDT',
                        USDT: 'USDT',
                    }[String(v || 'WIN').toUpperCase()] || v || '-'),
                },
                {
                    title: '提现金额',
                    dataIndex: 'amount',
                },
                {
                    title: '状态',
                    dataIndex: 'status',
                    customRender: (v) => ({
                        review: '待审核',
                        pending: '待打款',
                        doing: '处理中',
                        completed: '已转账',
                        success: '已转账',
                        failed: '失败',
                        rejected: '已拒绝',
                    }[v] || v || '-'),
                },
                {
                    title: '交易订单',
                    dataIndex: 'txHash',
                    customRender: (v) => v ? <a href={`https://eoeo.info/tx/${v}`} target="_blank" rel="noopener noreferrer">{v}</a> : '-',
                },
                {
                    title: '提现地址',
                    dataIndex: 'toAddress',
                },
                {
                    title: '创建时间',
                    dataIndex: 'createdAt',
                },
                {
                    title: '操作',
                    key: 'action',
                    fixed: 'right',
                    width: 150,
                    customRender: (v, row) => {
                        if (!row || row.status !== 'review') return '-'
                        return (
                            <span>
                                <a-button size="small" type="primary" onClick={() => this.approveWithdraw(row)}>通过</a-button>
                                <a-button size="small" style="margin-left:8px" onClick={() => this.rejectWithdraw(row)}>拒绝</a-button>
                            </span>
                        )
                    },
                },
            ],
            searchData: {
                address: '',
                asset: '',
                status: '',
                dateRange: [],
            },
        }
    },
    methods: {
        buildParams() {
            const params = {
                page: this.current,
                pageSize: this.pageSize,
                address: this.searchData.address || '',
                asset: this.searchData.asset || '',
                status: this.searchData.status || '',
            }
            if (this.searchData.dateRange && this.searchData.dateRange.length === 2) {
                params.startTime = moment(this.searchData.dateRange[0]).format('YYYY-MM-DD HH:mm:ss')
                params.endTime = moment(this.searchData.dateRange[1]).format('YYYY-MM-DD HH:mm:ss')
            }
            return params
        },
        getList() {
            this.loading = true
            Gai.withdraw_list(this.buildParams()).then((res) => {
                this.data = (res.withdraw || []).map((value, key) => {
                    return { ...value, key }
                })
                this.stats = (res && res.stats) || null
                this.loading = false
                this.total = parseInt(res.count || 0)
            }).catch(() => {
                this.loading = false
            })
        },
        approveWithdraw(row) {
            this.$confirm({
                title: '确认通过审核？',
                content: `用户 ${row.address} 提现 ${row.amount} ${row.asset === 'SDT' ? 'AIX-USDT' : 'WIN'}，通过后将进入自动打款队列。`,
                centered: true,
                onOk: () => Gai.withdraw_pass({ id: row.id }).then(() => {
                    this.$message.success('审核通过')
                    this.getList()
                }),
            })
        },
        rejectWithdraw(row) {
            let remark = ''
            this.$confirm({
                title: '确认拒绝提现？',
                content: (
                    <div>
                        <div style="margin-bottom:8px;">拒绝后将退回用户余额。</div>
                        <a-input placeholder="拒绝原因（可选）" onInput={(e) => { remark = e.target.value }} />
                    </div>
                ),
                centered: true,
                onOk: () => Gai.withdraw_reject({ id: row.id, remark }).then(() => {
                    this.$message.success('已拒绝并退回余额')
                    this.getList()
                }),
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
</style>
