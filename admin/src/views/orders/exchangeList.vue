<template>
    <PageView>
        <a-card title="AIX→可提U 兑换记录">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="钱包地址" allowClear @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.status" style="width:100%" placeholder="状态"
                        @change="getListTwo">
                        <a-select-option value="review">待审核</a-select-option>
                        <a-select-option value="completed">已完成</a-select-option>
                        <a-select-option value="rejected">已拒绝</a-select-option>
                    </a-select>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-button-group>
                        <a-button type="primary" :loading="loading" @click="getListTwo">确定筛选</a-button>
                    </a-button-group>
                </a-col>
            </a-row>
            <div class="stats-bar" v-if="stats">
                <span>待审核：<b>{{ stats.reviewCount || 0 }}</b></span>
            </div>
            <a-table :loading="loading" :columns="columns" :dataSource="data"
                :pagination="{ total, pageSize, showSizeChanger, current }" @change="changePagination" bordered
                :scroll="{ x: true }">
            </a-table>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'

const statusText = {
    review: '待审核',
    completed: '已完成',
    rejected: '已拒绝',
}

export default {
    name: 'exchangeList',
    mixins: [listMixin],
    data() {
        return {
            stats: null,
            columns: [
                {
                    title: 'ID',
                    dataIndex: 'id',
                },
                {
                    title: '地址',
                    dataIndex: 'address',
                },
                {
                    title: '兑换来源',
                    dataIndex: 'fromAsset',
                    customRender: (v) => v || 'AIX'
                },
                {
                    title: '支付数量',
                    dataIndex: 'fromAmount',
                },
                {
                    title: '获得币种',
                    dataIndex: 'toAsset',
                    customRender: (v) => v || 'USDT'
                },
                {
                    title: '到账数量',
                    dataIndex: 'toAmount',
                },
                {
                    title: '手续费',
                    dataIndex: 'feeAmount',
                    customRender: (v) => v || '0'
                },
                {
                    title: '手续费率',
                    dataIndex: 'feeRate',
                    customRender: (v) => {
                        if (!v) return '-'
                        try {
                            const pct = parseFloat(v) * 100
                            return isNaN(pct) ? v : pct.toFixed(2) + '%'
                        } catch (e) { return v }
                    }
                },
                {
                    title: '兑换价格',
                    dataIndex: 'exchangePrice',
                },
                {
                    title: '状态',
                    dataIndex: 'status',
                    customRender: (v) => {
                        const text = statusText[v] || v || '-'
                        const color = v === 'completed' ? 'green' : (v === 'review' ? 'orange' : 'red')
                        return <a-tag color={color}>{text}</a-tag>
                    }
                },
                {
                    title: '备注',
                    dataIndex: 'remark',
                    customRender: (v) => v || '-'
                },
                {
                    title: '时间',
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
                                <a-button size="small" type="primary" onClick={() => this.approveExchange(row)}>通过</a-button>
                                <a-button size="small" style="margin-left:8px" onClick={() => this.rejectExchange(row)}>拒绝</a-button>
                            </span>
                        )
                    },
                },
            ],
            searchData: {
                address: '',
                status: undefined,
            },
        }
    },
    methods: {
        buildParams() {
            const params = {
                page: this.current || 1,
                pageSize: this.pageSize || 10,
            }
            const address = (this.searchData.address || '').trim()
            if (address) params.address = address
            if (this.searchData.status) params.status = this.searchData.status
            return params
        },
        getList() {
            this.loading = true
            Gai.exchange_list(this.buildParams()).then((res) => {
                const list = (res && res.list) ? res.list : []
                this.data = list.map((value, key) => ({
                    ...value,
                    key: value.id != null ? value.id : key,
                }))
                this.total = parseInt((res && res.count) || 0, 10) || 0
                this.stats = (res && res.stats) || null
            }).catch(() => {
                this.data = []
                this.total = 0
                this.stats = null
            }).finally(() => {
                this.loading = false
            })
        },
        approveExchange(row) {
            this.$confirm({
                title: '确认通过该兑换？',
                content: `地址 ${row.address}，AIX ${row.fromAmount} → 可提U ${row.toAmount}`,
                onOk: () => Gai.exchange_pass({ id: row.id }).then(() => this.getList()),
            })
        },
        rejectExchange(row) {
            let remark = ''
            this.$confirm({
                title: '确认拒绝该兑换？',
                content: (
                    <div>
                        <p>将退回 AIX {row.fromAmount} 给用户</p>
                        <a-input placeholder="拒绝原因（可选）" onChange={(e) => { remark = e.target.value }} />
                    </div>
                ),
                onOk: () => Gai.exchange_reject({ id: row.id, remark }).then(() => this.getList()),
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
    background: #fff7e6;
    border: 1px solid #ffd591;
    border-radius: 4px;
    color: #333;

    b {
        color: #d46b08;
    }
}
</style>
