<template>
    <PageView>
        <a-card :title="`交易所划转（共 ${total} 条）`">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-input v-model="searchData.address" placeholder="用户地址" allowClear @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="6" :xl="4">
                    <a-select allowClear v-model="searchData.partner_id" style="width:100%" placeholder="合作方"
                        @change="getListTwo">
                        <a-select-option v-for="p in partners" :key="p.partnerId" :value="p.partnerId">
                            {{ p.partnerId }}{{ p.enabled ? '' : '（已停用）' }}
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
            <div class="stats-bar" v-if="stats">
                <span>筛选笔数：<b>{{ stats.totalCount || 0 }}</b></span>
                <span>划转总额：<b>{{ stats.amountTotal || 0 }}</b> WIN</span>
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
import moment from 'moment'

export default {
    name: 'exchangeTransfer',
    mixins: [listMixin],
    data() {
        return {
            stats: null,
            partners: [],
            columns: [
                {
                    title: '时间',
                    dataIndex: 'createdAt',
                },
                {
                    title: '合作方',
                    dataIndex: 'partnerId',
                },
                {
                    title: '用户地址',
                    dataIndex: 'address',
                },
                {
                    title: '金额(WIN)',
                    dataIndex: 'amount',
                },
                {
                    title: '流水号',
                    dataIndex: 'aixTxnId',
                },
                {
                    title: 'Nonce',
                    dataIndex: 'nonce',
                },
            ],
            searchData: {
                address: '',
                partner_id: undefined,
                dateRange: [],
            },
            pageSize: 50,
        }
    },
    mounted() {
        this.loadPartners()
        this.getList()
    },
    methods: {
        loadPartners() {
            Gai.partner_credit_partners().then((res) => {
                this.partners = (res && res.partners) ? res.partners : []
            }).catch(() => {
                this.partners = []
            })
        },
        buildParams() {
            const params = {
                page: this.current || 1,
                pageSize: this.pageSize || 50,
            }
            const address = (this.searchData.address || '').trim()
            if (address) params.address = address
            const partnerId = this.searchData.partner_id
            if (partnerId) params.partner_id = partnerId
            if (this.searchData.dateRange && this.searchData.dateRange.length === 2) {
                params.startTime = moment(this.searchData.dateRange[0]).format('YYYY-MM-DD HH:mm:ss')
                params.endTime = moment(this.searchData.dateRange[1]).format('YYYY-MM-DD HH:mm:ss')
            }
            return params
        },
        getList() {
            this.loading = true
            Gai.partner_credit_list(this.buildParams()).then((res) => {
                const list = (res && res.list) ? res.list : []
                this.data = list.map((value, key) => ({
                    ...value,
                    id: value.id != null ? value.id : key,
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
</style>
