<template>
    <PageView>
        <a-card title="团队统计">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="24" :md="8" :lg="7" :xl="6">
                    <a-input v-model="form.address" placeholder="钱包地址" allowClear @keyup.enter="saveAddress" />
                </a-col>
                <a-col :xs="24" :md="6" :lg="4" :xl="3">
                    <a-button type="primary" :loading="saving" @click="saveAddress">添加</a-button>
                </a-col>
            </a-row>
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="24" :md="12" :lg="10" :xl="8">
                    <a-range-picker
                        v-model="dateRange"
                        show-time
                        format="YYYY-MM-DD HH:mm:ss"
                        style="width:100%"
                        :placeholder="['开始时间', '结束时间']"
                    />
                </a-col>
                <a-col :xs="24" :md="10" :lg="8" :xl="6">
                    <a-button type="primary" :loading="loading" @click="getList">查询</a-button>
                    <a-button style="margin-left: 8px" :loading="exporting" @click="exportList">导出表格</a-button>
                </a-col>
            </a-row>
            <p class="hint">下级 USDT 为已入账的 USDT 充值合计。下级 WIN 为已入账的 WIN 充值与划转合计。下级 AIX-USDT 提现、下级 USDT 提现为已转账的提现金额合计。人数不含本人。未选时间时，业绩为累计值；选定时间后，充值、提现、人数和业绩只统计该时间段内的记录。</p>
            <a-table
                rowKey="id"
                :loading="loading"
                :columns="columns"
                :dataSource="data"
                :pagination="false"
                bordered
                :scroll="{ x: true }"
            >
                <template slot="name" slot-scope="text, record">
                    <a-input v-model="record.display_name" size="small" :maxLength="32" style="width: 140px; margin-right: 8px" />
                    <a-button size="small" @click="saveName(record)">保存</a-button>
                </template>
                <template slot="action" slot-scope="text, record">
                    <a-popconfirm title="从列表中移除该地址？" @confirm="remove(record)">
                        <a>移除</a>
                    </a-popconfirm>
                </template>
            </a-table>
        </a-card>
    </PageView>
</template>

<script>
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'
import moment from 'moment'

export default {
    name: 'teamStats',
    mixins: [listMixin],
    data () {
        return {
            saving: false,
            exporting: false,
            dateRange: [],
            form: {
                address: '',
            },
            columns: [
                { title: '名字', scopedSlots: { customRender: 'name' }, width: 240 },
                { title: '地址', dataIndex: 'address' },
                { title: '下级USDT充值', dataIndex: 'usdt_recharge' },
                { title: '下级WIN充值', dataIndex: 'win_recharge' },
                { title: '下级AIX-USDT提现', dataIndex: 'sdt_withdraw' },
                { title: '下级USDT提现', dataIndex: 'usdt_withdraw' },
                { title: '下级人数', dataIndex: 'downline_count' },
                { title: '团队总业绩', dataIndex: 'team_perf' },
                { title: '大区业绩', dataIndex: 'large_area_perf' },
                { title: '小区业绩', dataIndex: 'small_area_perf' },
                { title: '操作', scopedSlots: { customRender: 'action' }, width: 80 },
            ],
        }
    },
    methods: {
        listParams () {
            const params = {}
            if (this.dateRange && this.dateRange.length === 2) {
                params.startTime = moment(this.dateRange[0]).format('YYYY-MM-DD HH:mm:ss')
                params.endTime = moment(this.dateRange[1]).format('YYYY-MM-DD HH:mm:ss')
            }
            return params
        },
        getList () {
            this.loading = true
            Gai.team_stats_list(this.listParams()).then(res => {
                this.data = res.data || []
                this.total = parseInt(res.count || 0, 10)
            }).finally(() => {
                this.loading = false
            })
        },
        exportList () {
            this.exporting = true
            Gai.team_stats_export(this.listParams()).then((blob) => {
                const url = window.URL.createObjectURL(new Blob([blob]))
                const link = document.createElement('a')
                link.href = url
                link.setAttribute('download', `team_stats_${moment().format('YYYYMMDD_HHmmss')}.csv`)
                document.body.appendChild(link)
                link.click()
                document.body.removeChild(link)
                window.URL.revokeObjectURL(url)
            }).finally(() => {
                this.exporting = false
            })
        },
        saveAddress () {
            const address = (this.form.address || '').trim()
            if (!address) {
                this.$message.warning('请填写地址')
                return
            }
            this.saving = true
            Gai.team_stats_save({
                address,
            }).then(() => {
                this.form.address = ''
                this.getList()
            }).finally(() => {
                this.saving = false
            })
        },
        saveName (record) {
            Gai.team_stats_save({
                address: record.address,
                name: (record.display_name || '').trim(),
            }).then(() => this.getList())
        },
        remove (record) {
            Gai.team_stats_delete({ id: record.id }).then(() => this.getList())
        },
    },
}
</script>

<style scoped>
.hint {
    color: rgba(0, 0, 0, 0.45);
    margin: 8px 0 16px;
}
</style>
