<template>
    <PageView>
        <a-card title="划转记录">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="转出/转入地址" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="5" :xl="4">
                    <a-select v-model="searchData.type" allowClear placeholder="划转类型" style="width:100%" @change="getListTwo">
                        <a-select-option value="">全部类型</a-select-option>
                        <a-select-option value="user">用户互转</a-select-option>
                        <a-select-option value="self">充值钱包→奖励钱包</a-select-option>
                    </a-select>
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-button-group>
                        <a-button type="primary" :loading="loading" @click="getListTwo">确定筛选</a-button>
                    </a-button-group>
                </a-col>
            </a-row>
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

export default {
    name: 'transferList',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: 'ID',
                    dataIndex: 'id',
                },
                {
                    title: '类型',
                    dataIndex: 'typeLabel',
                    customRender: (v, row) => {
                        if (row && row.type === 'self') return <a-tag color="blue">{v || '充值钱包→奖励钱包'}</a-tag>
                        return <a-tag color="green">{v || '用户互转'}</a-tag>
                    }
                },
                {
                    title: '转出地址',
                    dataIndex: 'fromAddress',
                    customRender: (v) => v || '-'
                },
                {
                    title: '转出钱包',
                    dataIndex: 'fromWallet',
                    customRender: (v) => v || '-'
                },
                {
                    title: '转入地址',
                    dataIndex: 'toAddress',
                    customRender: (v) => v || '-'
                },
                {
                    title: '转入钱包',
                    dataIndex: 'toWallet',
                    customRender: (v) => v || '-'
                },
                {
                    title: '币种',
                    dataIndex: 'asset',
                    customRender: (v) => v || 'USDT'
                },
                {
                    title: '数量',
                    dataIndex: 'amount',
                },
                {
                    title: '时间',
                    dataIndex: 'createdAt',
                },
            ],
            searchData: {
                address: '',
                type: '',
            },
        }
    },
    methods: {
        getList() {
            this.loading = true
            const params = {
                page: this.current,
                pageSize: this.pageSize,
            }
            if (this.searchData.address) params.address = this.searchData.address
            if (this.searchData.type) params.type = this.searchData.type
            Gai.transfer_list(params).then((res) => {
                const payload = res && res.data && (res.data.list || res.data.count != null) ? res.data : res
                const list = (payload && (payload.list || payload.transfers)) || []
                this.data = list.map((value, key) => {
                    return { ...value, key }
                })
                this.loading = false
                this.total = parseInt((payload && payload.count) || list.length || 0)
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
