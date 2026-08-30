<template>
    <PageView>
        <a-card title="操作记录（仅主账户可见）">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.operator" placeholder="操作账号" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.action" placeholder="操作说明" @keyup.enter="getListTwo" />
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
    name: 'operationLog',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                { title: 'ID', dataIndex: 'id' },
                { title: '操作账号', dataIndex: 'operator' },
                {
                    title: '账号类型',
                    dataIndex: 'operator_type',
                    customRender: (v) => v === 'main' ? '主账户' : '子账户',
                },
                { title: '操作', dataIndex: 'action_label' },
                { title: '时间', dataIndex: 'created_at' },
            ],
            searchData: {
                operator: '',
                action: '',
            },
        }
    },
    methods: {
        getList() {
            this.loading = true
            Gai.operation_log_list({
                page: this.current,
                pageSize: this.pageSize,
                ...this.searchData,
            }).then((res) => {
                this.data = (res.list || []).map((value, key) => ({ ...value, key }))
                this.loading = false
                this.total = parseInt(res.count || 0)
            }).catch(() => {
                this.loading = false
            })
        },
    },
}
</script>

<style scoped>
.inputGroup {
    margin-bottom: 16px;
}
</style>
