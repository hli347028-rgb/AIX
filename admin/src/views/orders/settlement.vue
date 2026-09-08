<template>
    <PageView>
        <a-card title="每日结算">
            <a-alert
                type="info"
                show-icon
                style="margin-bottom: 16px"
                message="每日结算仅由系统在中国时区 0 点自动执行，后台无法手动触发。同一时刻会按当时全网总 AIX 锁定「今日兑换额度」（一天只算一次，与是否结算成功无关）。列表仅供查看。"
            />
            <a-row :gutter="10" class="inputGroup" style="margin-bottom: 16px">
                <a-col :xs="24" :md="10" :lg="8" :xl="6">
                    <a-button :loading="loading" @click="getListTwo">刷新列表</a-button>
                </a-col>
            </a-row>
            <a-table
                :loading="loading"
                :columns="columns"
                :dataSource="data"
                :pagination="{ total, pageSize, current }"
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

export default {
    name: 'settlement',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: 'ID',
                    dataIndex: 'id',
                    width: 80,
                },
                {
                    title: '结算日期',
                    dataIndex: 'settlementDate',
                },
                {
                    title: '状态',
                    dataIndex: 'status',
                    customRender: (v) => ({
                        running: '进行中',
                        completed: '已完成',
                        success: '已完成',
                        failed: '失败',
                    }[v] || v),
                },
                {
                    title: 'AIX价格',
                    dataIndex: 'aixPrice',
                    customRender: (v) => {
                        const n = Number(v)
                        if (!Number.isFinite(n) || n <= 0) return v || '-'
                        const text = String(v ?? '')
                        if (/^\d+(\.\d+)?$/.test(text)) {
                            const [i, f = ''] = text.split('.')
                            return `${i}.${f.padEnd(15, '0').slice(0, 15)}`
                        }
                        return n.toFixed(15)
                    },
                },
                {
                    title: '静态合计',
                    dataIndex: 'staticAmount',
                },
                {
                    title: '当日兑换额度',
                    dataIndex: 'exchangeQuotaLimit',
                    customRender: (v, row) => {
                        const limit = v != null && v !== '' ? String(v) : ''
                        if (!limit || Number(limit) === 0) return '—'
                        const base = row && row.exchangeQuotaBase != null ? String(row.exchangeQuotaBase) : ''
                        if (base && Number(base) !== 0) {
                            return `${limit}（基数 ${base}）`
                        }
                        return limit
                    },
                },
                {
                    title: '管理奖合计',
                    dataIndex: 'mgmtAmount',
                    customRender: (v) => (v && Number(v) !== 0 ? v : '—（认购即时）'),
                },
                {
                    title: '本轮释放合计',
                    dataIndex: 'releaseTotal',
                },
                {
                    title: '开始时间',
                    dataIndex: 'startedAt',
                },
                {
                    title: '结束时间',
                    dataIndex: 'finishedAt',
                },
            ],
        }
    },
    methods: {
        getList() {
            this.loading = true
            Gai.settlement_list({
                page: this.current,
                pageSize: this.pageSize,
            }).then((res) => {
                this.data = (res.list || []).map((value, key) => {
                    return { ...value, key }
                })
                this.total = parseInt(res.total || res.count || 0)
                this.loading = false
            }).catch(() => {
                this.loading = false
            })
        },
    },
}
</script>

<style scoped lang="less">
.inputGroup {
    > div {
        margin-bottom: 12px;
    }
}
</style>
