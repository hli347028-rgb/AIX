<template>
    <PageView>
        <a-card title="用户数据">
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
            <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="{ total, pageSize, current }"
                @change="changePagination" bordered :scroll="{ x: true }">
            </a-table>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'

export default {
    name: 'member',
    mixins: [listMixin],
    data() {
        return {
            columns: [
                {
                    title: '创建时间',
                    dataIndex: 'createdAt',
                },
                {
                    title: '地址',
                    dataIndex: 'address',
                },
                {
                    title: '用户名',
                    dataIndex: 'username',
                    customRender: (v) => v || '—',
                },
                {
                    title: '充值钱包',
                    dataIndex: 'usdt_recharge',
                },
                {
                    title: '奖励钱包',
                    dataIndex: 'usdt_reward',
                },
                {
                    title: 'AIX代币数',
                    dataIndex: 'aix_balance',
                },
                {
                    title: 'WIN提现钱包',
                    dataIndex: 'win_balance',
                    customRender: (v) => v || '0'
                },
                {
                    title: '可提U余额',
                    dataIndex: 'usdt_withdrawable',
                    customRender: (v, row) => {
                        const withdraw = parseFloat((row && row.usdt_withdrawable) || '0') || 0
                        const subsidy = parseFloat((row && row.community_subsidy_total) || '0') || 0
                        const legacyZero = parseFloat((row && row.zero_account_reward_total) || '0') || 0
                        const sum = withdraw > 0 ? withdraw : (subsidy + legacyZero)
                        if (!Number.isFinite(sum)) return '0'
                        return String(parseFloat(sum.toFixed(8)))
                    }
                },
                {
                    title: 'WIN充值钱包',
                    dataIndex: 'win_recharge_balance',
                    customRender: (v) => v || '0'
                },
                {
                    title: 'WIN-A充值余额',
                    dataIndex: 'win_a_recharge_balance',
                    customRender: (v) => v || '0'
                },
                {
                    title: '溢出奖励',
                    dataIndex: 'overflow_reward',
                    customRender: (v, row) => v || (row && row.pending_mgmt_reward) || '0'
                },
                {
                    title: 'AIX-USDT',
                    dataIndex: 'points',
                    customRender: (v) => (v === null || v === undefined || v === '') ? '0' : String(v)
                },
                {
                    title: '累计AIX-USDT',
                    dataIndex: 'points_all',
                    customRender: (v) => (v === null || v === undefined || v === '') ? '0' : String(v)
                },
                {
                    title: '静态总收益',
                    dataIndex: 'static_usdt_total',
                },
                {
                    title: '总订单',
                    dataIndex: 'amountUsdtCurrent',
                    customRender: (v) => v || '0'
                },
                {
                    title: '进行中订单',
                    dataIndex: 'amountUsdtActive',
                    customRender: (v) => v || '0'
                },
                {
                    title: '总收益',
                    dataIndex: 'totalIncome',
                    customRender: (v) => v || '0'
                },
                {
                    title: '已释放',
                    dataIndex: 'releasedAmount',
                    customRender: (v) => v || '0'
                },
                {
                    title: '待释放',
                    dataIndex: 'pendingRelease',
                    customRender: (v) => v || '0'
                },
                {
                    title: '社区等级',
                    dataIndex: 'mgmt_level',
                    customRender: (v, row) => {
                        if (row && row.vip) {
                            const raw = String(row.vip).toUpperCase().replace(/^[WAV]/g, '')
                            const n = parseInt(raw, 10)
                            return 'A' + (Number.isFinite(n) ? Math.min(Math.max(n, 0), 10) : 0)
                        }
                        const n = parseInt(v, 10) || 0
                        return 'A' + Math.min(Math.max(n, 0), 10)
                    }
                },
                {
                    title: '账户状态',
                    dataIndex: 'is_frozen',
                    customRender: (v) => v ? '已冻结' : '正常',
                },
                {
                    title: '社区补贴档位',
                    dataIndex: 'community_subsidy_rate',
                    customRender: (v, row) => {
                        if (!row || !row.is_community_subsidy) return '未开通'
                        const n = parseInt(v, 10)
                        if (Number.isFinite(n) && n > 0) return `${n}%`
                        return '—'
                    },
                },
                {
                    title: '补贴设置时间',
                    dataIndex: 'community_subsidy_set_at',
                    customRender: (v) => v || '-',
                },
				{
					title: '大区业绩',
					dataIndex: 'large_area_perf',
				},
				{
                    title: '小区业绩',
                    dataIndex: 'small_area_perf',
                },
                {
                    title: '团队业绩',
                    dataIndex: 'team_perf',
                },
                {
                    title: '直推人数',
                    dataIndex: 'invitee_count',
                },
                {
                    title: '上级地址',
                    dataIndex: 'myRecommendAddress',
                },
                {
                    title: '操作',
                    key: 'action',
                    fixed: 'right',
                    width: 220,
                    customRender: (v) => {
                        return (
                            <div>
                                <a-button-group>
                                    <a-button
                                        type="primary"
                                        onClick={() => {
                                            this.$router.push({ name: 'lookChildren', query: { userId: v.userId || v.id } })
                                        }}
                                    >
                                        查看下级
                                    </a-button>

                                    <a-dropdown>
                                        <a-button type="primary">
                                            更多
                                            <a-icon type="down" />
                                        </a-button>

                                        <a-menu slot="overlay">
                                            <a-menu-item onClick={() => this.add_account_balance(v.address)}>
                                                添加充值余额
                                            </a-menu-item>

                                            <a-menu-item onClick={() => this.vip_update(v.userId || v.id, v.vip || v.mgmt_level)}>
                                                设置级别(A0~A10)
                                            </a-menu-item>

                                            <a-menu-item onClick={() => this.set_community_subsidy(v.userId || v.id, v.community_subsidy_rate)}>
                                                设置社区补贴
                                            </a-menu-item>

                                            <a-menu-item onClick={() => this.set_exchange_enabled(v.userId || v.id, v.exchange_enabled)}>
                                                {v.exchange_enabled === false ? '开启兑换功能' : '关闭兑换功能'}
                                            </a-menu-item>

                                            <a-menu-item onClick={() => this.set_inviter(v.userId || v.id, v.address, v.myRecommendAddress)}>
                                                更改上级地址
                                            </a-menu-item>

                                            <a-menu-item onClick={() => this.change_address(v.userId || v.id, v.address)}>
                                                更换钱包地址
                                            </a-menu-item>

                                            <a-menu-item onClick={() => this.set_frozen(v.userId || v.id, v.is_frozen)}>
                                                {v.is_frozen ? '解冻账户' : '冻结账户'}
                                            </a-menu-item>
                                        </a-menu>
                                    </a-dropdown>
                                </a-button-group>
                            </div>
                        )
                    },
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
            Gai.user_list({
                page: this.current,
                pageSize: this.pageSize,
                ...this.searchData
            }).then((res) => {
                this.data = (res.users || []).map((value, key) => {
                    return { ...value, key }
                })
                this.loading = false
                this.total = parseInt(res.count || 0)
            }).catch(() => {
                this.loading = false
            })
        },
        vip_update(user_id, defaultValue) {
            let vip = String(defaultValue == null ? '0' : defaultValue).replace(/^[WAV]/gi, '')
            if (!vip) vip = '0'
            this.$confirm({
                title: `设置级别(A0~A10)`,
                content: (
                    <a-select style="width:240px" defaultValue={vip} placeholder="选择级别" onChange={(val) => {
                        vip = val;
                    }}>
                        <a-select-option value="0">A0（无级别）</a-select-option>
                        <a-select-option value="1">A1</a-select-option>
                        <a-select-option value="2">A2</a-select-option>
                        <a-select-option value="3">A3</a-select-option>
                        <a-select-option value="4">A4</a-select-option>
                        <a-select-option value="5">A5</a-select-option>
                        <a-select-option value="6">A6</a-select-option>
                        <a-select-option value="7">A7</a-select-option>
                        <a-select-option value="8">A8</a-select-option>
                        <a-select-option value="9">A9</a-select-option>
                        <a-select-option value="10">A10</a-select-option>
                    </a-select>
                ),
                centered: true,
                onOk: () => {
                    return new Promise((resolve, reject) => {
                        if (vip === undefined || vip === null || vip === '') {
                            this.$notification.warning({
                                message: '提示',
                                description: '请选择级别'
                            })
                            reject()
                            return;
                        }
                        Gai.vip_update({ user_id, vip: 'A' + vip }).then(() => {
                            this.$message.success('级别已更新')
                            resolve()
                            this.getList()
                        }).catch(() => {
                            reject()
                        })
                    })
                }
            })
        },
        add_account_balance(address) {
            let amount = ""
            this.$confirm({
                title: `添加充值余额 (usdt_recharge)`,
                content: (
                    <div>
                        <div style="margin-bottom:8px;color:#888;font-size:12px;">地址：{address}</div>
                        <a-input style="margin-top:8px;" placeholder="请输入增加的金额(USDT)" onInput={(val) => {
                            amount = val.target.value
                        }} />
                    </div>
                ),
                centered: true,
                onOk: () => {
                    const n = parseFloat(amount)
                    if (!amount || isNaN(n) || n <= 0) {
                        this.$message.warning('请填写大于 0 的金额')
                        return Promise.reject()
                    }
                    return Gai.admin_recharge({ address, amount }).then(() => {
                        this.$message.success('已添加到充值余额')
                        this.getList()
                    })
                }
            })
        },
        set_community_subsidy(user_id, currentRate) {
            let rate = String(currentRate || 0)
            this.$confirm({
                title: '设置社区补贴（级差 5%/10%/15%）',
                content: (
                    <div>
                        <div style="margin-bottom:8px;color:#888;font-size:12px;">
                            下级 USDT 充值按级差发放；下级档位会阻断上级同档或更低档收益。
                        </div>
                        <a-select style="width:240px" defaultValue={rate} onChange={(val) => { rate = val }}>
                            <a-select-option value="0">关闭</a-select-option>
                            <a-select-option value="5">5%</a-select-option>
                            <a-select-option value="10">10%</a-select-option>
                            <a-select-option value="15">15%</a-select-option>
                        </a-select>
                    </div>
                ),
                centered: true,
                onOk: () => {
                    return Gai.set_community_subsidy({ user_id, rate }).then(() => {
                        this.$message.success('社区补贴已更新')
                        this.getList()
                    })
                }
            })
        },
        set_frozen(user_id, current) {
            const willFreeze = !current
            this.$confirm({
                title: willFreeze ? '冻结账户' : '解冻账户',
                content: willFreeze
                    ? '冻结后该账户将无法登录、充值、报单和提现，确认冻结？'
                    : '解冻后该账户恢复正常使用，确认解冻？',
                centered: true,
                onOk: () => {
                    return Gai.set_frozen({ user_id, enabled: willFreeze ? '1' : '0' }).then(() => {
                        this.$message.success(willFreeze ? '账户已冻结' : '账户已解冻')
                        this.getList()
                    })
                }
            })
        },
        set_exchange_enabled(user_id, current) {
            // 默认开启；仅当明确为 false 时视为已关闭
            const currentlyEnabled = current !== false && current !== 0 && current !== '0'
            const willEnable = !currentlyEnabled
            this.$confirm({
                title: willEnable ? '开启兑换功能' : '关闭兑换功能',
                content: willEnable
                    ? '开启后该用户可在「我的资产」中进行 AIX 兑换，确认开启？'
                    : '关闭后该用户「确认兑换」按钮将不可点击，也无法提交兑换，确认关闭？',
                centered: true,
                onOk: () => {
                    return Gai.set_exchange_enabled({ user_id, enabled: willEnable ? '1' : '0' }).then(() => {
                        this.$message.success(willEnable ? '兑换功能已开启' : '兑换功能已关闭')
                        this.getList()
                    })
                }
            })
        },
        set_inviter(user_id, userAddress, currentInviter) {
            let inviter_address = currentInviter ? String(currentInviter) : ''
            this.$confirm({
                title: '更改上级地址',
                content: (
                    <div>
                        <div style="margin-bottom:8px;color:#888;font-size:12px;">用户：{userAddress}</div>
                        <div style="margin-bottom:8px;color:#888;font-size:12px;">请输入新上级的钱包地址（须已在系统中注册）</div>
                        <a-input style="margin-top:8px;" defaultValue={inviter_address} placeholder="0x..." onInput={(val) => {
                            inviter_address = val.target.value
                        }} />
                    </div>
                ),
                centered: true,
                onOk: () => {
                    const addr = String(inviter_address || '').trim()
                    if (!addr) {
                        this.$message.warning('请填写上级地址')
                        return Promise.reject()
                    }
                    return Gai.set_inviter({ user_id, inviter_address: addr }).then(() => {
                        this.$message.success('上级地址已更新')
                        this.getList()
                    })
                }
            })
        },
        change_address(user_id, oldAddress) {
            let new_address = ''
            this.$confirm({
                title: '更换钱包地址',
                content: (
                    <div>
                        <div style="margin-bottom:8px;color:#888;font-size:12px;">当前地址：{oldAddress}</div>
                        <div style="margin-bottom:8px;color:#888;font-size:12px;">请输入未在系统注册过的新钱包地址，原账户资产与关系将保留到新地址</div>
                        <a-input style="margin-top:8px;" placeholder="0x..." onInput={(val) => {
                            new_address = val.target.value
                        }} />
                    </div>
                ),
                centered: true,
                onOk: () => {
                    const addr = String(new_address || '').trim()
                    if (!addr) {
                        this.$message.warning('请填写新钱包地址')
                        return Promise.reject()
                    }
                    if (!/^0x[a-fA-F0-9]{40}$/.test(addr)) {
                        this.$message.warning('钱包地址格式无效')
                        return Promise.reject()
                    }
                    if (String(oldAddress || '').toLowerCase() === addr.toLowerCase()) {
                        this.$message.warning('新地址与当前地址相同')
                        return Promise.reject()
                    }
                    return Gai.change_address({ user_id, new_address: addr }).then(() => {
                        this.$message.success('钱包地址已更换')
                        this.getList()
                    })
                }
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
