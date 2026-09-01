<template>
    <PageView>
        <a-card title="问题反馈">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="8" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="账户地址" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="4">
                    <a-select v-model="searchData.status" allowClear placeholder="处理状态" style="width:100%" @change="getListTwo">
                        <a-select-option value="">全部状态</a-select-option>
                        <a-select-option value="0">待处理</a-select-option>
                        <a-select-option value="1">已读</a-select-option>
                        <a-select-option value="2">已处理</a-select-option>
                    </a-select>
                </a-col>
                <a-col :xs="12" :md="8" :lg="6" :xl="4">
                    <a-button type="primary" :loading="loading" @click="getListTwo">查询</a-button>
                </a-col>
            </a-row>
            <a-table
                :loading="loading"
                :columns="columns"
                :dataSource="data"
                :pagination="{ total, pageSize, showSizeChanger, current }"
                @change="changePagination"
                bordered
                :scroll="{ x: true }">
            </a-table>
        </a-card>

        <a-modal
            title="反馈详情"
            :visible="detailVisible"
            :footer="null"
            width="640px"
            @cancel="detailVisible = false">
            <p><b>账户：</b>{{ detailRow.address || '—' }}</p>
            <p><b>状态：</b>{{ detailRow.status_text || '—' }}</p>
            <p><b>提交时间：</b>{{ detailRow.created_at || '—' }}</p>
            <p><b>反馈内容：</b></p>
            <div class="feedback-content">{{ detailRow.content || '—' }}</div>
        </a-modal>
    </PageView>
</template>

<script type="text/jsx">
import Feedback from '../../api/Feedback'
import listMixin from '../mixin/listMixin'

export default {
    name: 'feedbackList',
    mixins: [listMixin],
    data () {
        return {
            searchData: {
                address: '',
                status: '',
            },
            detailVisible: false,
            detailRow: {},
            columns: [
                {
                    title: '账户',
                    dataIndex: 'address',
                    customRender: (v) => v || '—',
                },
                {
                    title: '反馈内容',
                    dataIndex: 'content',
                    customRender: (v) => {
                        const text = v || ''
                        if (text.length <= 40) return text || '—'
                        return `${text.slice(0, 40)}...`
                    },
                },
                {
                    title: '状态',
                    dataIndex: 'status_text',
                },
                {
                    title: '提交时间',
                    dataIndex: 'add_time',
                    customRender: (v) => this.timeOne(v),
                },
                {
                    title: '操作',
                    key: 'action',
                    fixed: 'right',
                    width: 120,
                    customRender: (v) => {
                        return (
                            <a-dropdown>
                                <a-menu slot="overlay">
                                    <a-menu-item onClick={() => this.openDetail(v)}>查看</a-menu-item>
                                    {Number(v.status) === 0 ? (
                                        <a-menu-item onClick={() => this.updateStatus(v.id, 1)}>标记已读</a-menu-item>
                                    ) : null}
                                    {Number(v.status) !== 2 ? (
                                        <a-menu-item onClick={() => this.updateStatus(v.id, 2)}>标记已处理</a-menu-item>
                                    ) : null}
                                </a-menu>
                                <a-button>操作 <a-icon type="down"/></a-button>
                            </a-dropdown>
                        )
                    },
                },
            ],
        }
    },
    methods: {
        getList () {
            this.loading = true
            const params = {
                page: this.current,
                num: this.pageSize,
            }
            if (this.searchData.address) params.address = this.searchData.address
            if (this.searchData.status !== '' && this.searchData.status != null) params.status = this.searchData.status
            Feedback.getList(params).then(res => {
                this.data = (res.data || []).map((value, key) => ({ ...value, key }))
                this.total = parseInt(res.count || 0, 10)
            }).finally(() => {
                this.loading = false
            })
        },
        openDetail (row) {
            this.detailRow = row || {}
            this.detailVisible = true
            if (row && Number(row.status) === 0) {
                this.updateStatus(row.id, 1, false)
            }
        },
        updateStatus (id, status, refresh = true) {
            return Feedback.updateStatus({ id, status }).then(() => {
                if (refresh) {
                    this.$message.success('状态已更新')
                    this.getList()
                } else if (this.detailRow && this.detailRow.id === id) {
                    this.detailRow.status = status
                    this.detailRow.status_text = status === 1 ? '已读' : status === 2 ? '已处理' : '待处理'
                    const row = this.data.find(item => item.id === id)
                    if (row) {
                        row.status = status
                        row.status_text = this.detailRow.status_text
                    }
                }
            })
        },
    },
}
</script>

<style scoped lang="less">
.inputGroup {
    > div {
        margin-bottom: 20px;
    }
}

.feedback-content {
    white-space: pre-wrap;
    word-break: break-word;
    line-height: 1.7;
    padding: 12px;
    background: #fafafa;
    border-radius: 4px;
}
</style>
