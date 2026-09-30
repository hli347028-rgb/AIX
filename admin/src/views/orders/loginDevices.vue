<template>
    <PageView>
        <a-card title="登录设备（同设备多账号排查）">
            <a-row :gutter="10" class="inputGroup">
                <a-col :xs="12" :md="6" :lg="5" :xl="4">
                    <a-input v-model="searchData.device_id" placeholder="设备 ID" @keyup.enter="getListTwo" />
                </a-col>
                <a-col :xs="12" :md="6" :lg="4" :xl="3">
                    <a-input-number
                        v-model="searchData.min_accounts"
                        :min="1"
                        :precision="0"
                        style="width: 100%"
                        placeholder="最少账号数"
                    />
                </a-col>
                <a-col :xs="12" :md="6" :lg="4" :xl="3">
                    <a-input v-model="searchData.address" placeholder="按用户地址查流水" @keyup.enter="loadUserLogs" />
                </a-col>
                <a-col :xs="24" :md="12" :lg="8" :xl="6">
                    <a-button-group>
                        <a-button type="primary" :loading="loading" @click="getListTwo">筛设备</a-button>
                        <a-button :loading="userLogLoading" @click="loadUserLogs">查用户登录</a-button>
                    </a-button-group>
                </a-col>
            </a-row>

            <a-table
                :loading="loading"
                :columns="columns"
                :dataSource="data"
                :pagination="{ total, pageSize, showSizeChanger, current }"
                @change="changePagination"
                bordered
                :scroll="{ x: true }"
                :customRow="customRow"
                rowKey="device_id"
            />
        </a-card>

        <a-card v-if="selectedDevice" :title="`设备关联账号：${selectedDevice}`" style="margin-top: 16px">
            <a-table
                :loading="usersLoading"
                :columns="userColumns"
                :dataSource="deviceUsers"
                :pagination="{
                    total: usersTotal,
                    pageSize: usersPageSize,
                    current: usersCurrent,
                    showSizeChanger: true,
                }"
                @change="changeUsersPagination"
                bordered
                :scroll="{ x: true }"
                rowKey="user_id"
            />
        </a-card>

        <a-card v-if="userLogAddress" :title="`用户登录流水：${userLogAddress}`" style="margin-top: 16px">
            <a-table
                :loading="userLogLoading"
                :columns="logColumns"
                :dataSource="userLogs"
                :pagination="{
                    total: userLogTotal,
                    pageSize: userLogPageSize,
                    current: userLogCurrent,
                    showSizeChanger: true,
                }"
                @change="changeUserLogPagination"
                bordered
                :scroll="{ x: true }"
                rowKey="id"
            />
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'

export default {
    name: 'loginDevices',
    mixins: [listMixin],
    data() {
        return {
            searchData: {
                device_id: '',
                min_accounts: 2,
                address: '',
            },
            columns: [
                { title: '设备 ID', dataIndex: 'device_id', width: 280 },
                { title: '关联账号数', dataIndex: 'account_count', width: 110 },
                { title: '登录次数', dataIndex: 'login_count', width: 100 },
                { title: '设备标签', dataIndex: 'device_label', width: 160 },
                { title: '最近 IP', dataIndex: 'last_ip', width: 140 },
                { title: '最近客户端', dataIndex: 'last_client', width: 120 },
                { title: '最近登录', dataIndex: 'last_login_at', width: 170 },
                {
                    title: '最近 UA',
                    dataIndex: 'last_ua',
                    customRender: (v) => (
                        <span style="max-width:280px;display:inline-block;word-break:break-all;">{v || '—'}</span>
                    ),
                },
                {
                    title: '操作',
                    key: 'action',
                    fixed: 'right',
                    width: 120,
                    customRender: (v, row) => (
                        <a-button type="primary" size="small" onClick={() => this.openDevice(row.device_id)}>
                            关联账号
                        </a-button>
                    ),
                },
            ],
            selectedDevice: '',
            deviceUsers: [],
            usersLoading: false,
            usersTotal: 0,
            usersPageSize: 10,
            usersCurrent: 1,
            userColumns: [
                { title: 'UID', dataIndex: 'user_id', width: 90 },
                { title: '地址', dataIndex: 'address', width: 360 },
                { title: '用户名', dataIndex: 'username', customRender: (v) => v || '—' },
                { title: '登录次数', dataIndex: 'login_count', width: 100 },
                { title: '首次登录', dataIndex: 'first_login_at', width: 170 },
                { title: '最近登录', dataIndex: 'last_login_at', width: 170 },
                { title: '最近 IP', dataIndex: 'last_ip', width: 140 },
                {
                    title: '冻结',
                    dataIndex: 'is_frozen',
                    width: 80,
                    customRender: (v) => (v ? '是' : '否'),
                },
            ],
            userLogAddress: '',
            userLogs: [],
            userLogLoading: false,
            userLogTotal: 0,
            userLogPageSize: 10,
            userLogCurrent: 1,
            logColumns: [
                { title: 'ID', dataIndex: 'id', width: 90 },
                { title: '设备 ID', dataIndex: 'device_id', width: 280 },
                { title: '设备标签', dataIndex: 'device_label', width: 160 },
                { title: 'IP', dataIndex: 'client_ip', width: 140 },
                { title: '客户端', dataIndex: 'client', width: 120 },
                { title: '时间', dataIndex: 'created_at', width: 170 },
                {
                    title: 'UA',
                    dataIndex: 'user_agent',
                    customRender: (v) => (
                        <span style="max-width:280px;display:inline-block;word-break:break-all;">{v || '—'}</span>
                    ),
                },
                {
                    title: '操作',
                    key: 'action',
                    width: 120,
                    customRender: (v, row) => (
                        <a-button type="link" size="small" onClick={() => this.openDevice(row.device_id)}>
                            查同设备
                        </a-button>
                    ),
                },
            ],
        }
    },
    activated() {
        const q = this.$route.query || {}
        if (q.device_id) {
            this.searchData.device_id = String(q.device_id)
            this.selectedDevice = String(q.device_id)
            this.loadDeviceUsers()
        }
        if (q.address) {
            this.searchData.address = String(q.address)
            this.loadUserLogs()
        }
        if (q.user_id) {
            this.loadUserLogsById(String(q.user_id))
        }
        this.getList()
    },
    methods: {
        customRow(record) {
            return {
                on: {
                    click: () => this.openDevice(record.device_id),
                },
                style: { cursor: 'pointer' },
            }
        },
        getList() {
            this.loading = true
            Gai.login_device_list({
                page: this.current,
                pageSize: this.pageSize,
                device_id: this.searchData.device_id || undefined,
                min_accounts: this.searchData.min_accounts || 2,
            })
                .then((res) => {
                    this.data = (res.list || []).map((value, key) => ({ ...value, key }))
                    this.total = parseInt(res.count || 0, 10)
                    this.loading = false
                })
                .catch(() => {
                    this.loading = false
                })
        },
        openDevice(deviceId) {
            if (!deviceId) return
            this.selectedDevice = deviceId
            this.usersCurrent = 1
            this.loadDeviceUsers()
        },
        loadDeviceUsers() {
            if (!this.selectedDevice) return
            this.usersLoading = true
            Gai.login_device_users({
                device_id: this.selectedDevice,
                page: this.usersCurrent,
                pageSize: this.usersPageSize,
            })
                .then((res) => {
                    this.deviceUsers = (res.list || []).map((value, key) => ({ ...value, key }))
                    this.usersTotal = parseInt(res.count || 0, 10)
                    this.usersLoading = false
                })
                .catch(() => {
                    this.usersLoading = false
                })
        },
        changeUsersPagination(pagination) {
            this.usersCurrent = pagination.current
            this.usersPageSize = pagination.pageSize
            this.loadDeviceUsers()
        },
        loadUserLogs() {
            const address = (this.searchData.address || '').trim()
            if (!address) {
                this.$message.warning('请输入用户地址')
                return
            }
            this.userLogAddress = address
            this.userLogCurrent = 1
            this.fetchUserLogs({ address })
        },
        loadUserLogsById(userId) {
            this.userLogAddress = `UID ${userId}`
            this.userLogCurrent = 1
            this.fetchUserLogs({ user_id: userId })
        },
        fetchUserLogs(extra) {
            this.userLogLoading = true
            Gai.user_login_devices({
                page: this.userLogCurrent,
                pageSize: this.userLogPageSize,
                ...extra,
            })
                .then((res) => {
                    this.userLogs = (res.list || []).map((value, key) => ({ ...value, key }))
                    this.userLogTotal = parseInt(res.count || 0, 10)
                    this.userLogLoading = false
                })
                .catch(() => {
                    this.userLogLoading = false
                })
        },
        changeUserLogPagination(pagination) {
            this.userLogCurrent = pagination.current
            this.userLogPageSize = pagination.pageSize
            if ((this.searchData.address || '').trim()) {
                this.fetchUserLogs({ address: this.searchData.address.trim() })
            }
        },
    },
}
</script>

<style scoped>
.inputGroup {
    margin-bottom: 16px;
}
</style>
