<template>
    <PageView>
        <a-card title="WIN 提现记录">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="账户地址" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-button-group>
                        <a-button type="primary" :loading="loading" @click="getListTwo">确定筛选</a-button>
                    </a-button-group>
                </a-col>
            </a-row>
            <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="{ total, pageSize, current, showSizeChanger }"
                @change="changePagination" bordered :scroll="{ x: true }">
            </a-table>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'
export default {
    name: 'withdrawList',
    mixins: [listMixin],
    data() {
        return {
            pageSize: 20,
            columns: [
                {
                    title: '账户',
                    dataIndex: 'address',
                },
                {
                    title: '提现金额',
                    dataIndex: 'amount',
                },
                {
                    title: '状态',
                    dataIndex: 'status',
                    customRender: (v) => ({
                        pending: '待处理',
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
            ],
            searchData: {
                address: '',
            },
        }
    },
    methods: {
        getList() {
            this.loading = true
            Gai.withdraw_list({
                page: this.current,
                pageSize: this.pageSize,
                ...this.searchData
            }).then((res) => {
                this.data = (res.withdraw || []).map((value, key) => {
                    return { ...value, key }
                })
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
</style>
