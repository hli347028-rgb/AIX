<template>
    <PageView>
        <a-card class="cardCon" title="AIX 数据统计" :loading="loading">
            <a-row v-for="(row, rowIndex) in statRows" :key="rowIndex" :gutter="0">
                <a-col v-for="(key, colIndex) in row" :key="`${rowIndex}-${colIndex}`" :xs="12" :md="6" :lg="6" :xl="6">
                    <a-card-grid v-if="key && statMeta[key]">
                        <a-icon :type="statMeta[key].icon" />
                        <div>
                            <a>{{ statMeta[key].label }}:</a> {{ formatValue(key) }}
                        </div>
                    </a-card-grid>
                </a-col>
            </a-row>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'

// 四列布局，与需求表格一致（按行从左到右）
const statRows = [
    ['totalUserR', 'totalUser', 'todayUserR', 'todayUser'],
    ['buyTotal', 'todayBuy', 'totalUsdtChainRecharge', 'todayUsdtChainRecharge'],
    ['totalWinChainRecharge', 'todayWinChainRecharge', 'totalWinAChainRecharge', 'todayWinAChainRecharge'],
    ['totalRewardReinvest', 'todayRewardReinvest', 'totalDynamic', 'todayDynamic'],
    ['totalStaticRelease', 'yesterdayStaticRelease', 'totalWinWithdraw', 'todayWinWithdraw'],
    ['totalSdtWithdraw', 'todaySdtWithdraw', 'totalSdtAsset', 'totalWinAsset'],
    ['totalAdminRecharge', 'todayAdminRecharge', 'totalZeroAccountReward', 'todayZeroAccountReward'],
    ['totalCommunitySubsidyReward', 'todayCommunitySubsidyReward', 'totalUsdtWithdrawable', 'totalUsdtWithdraw'],
    ['todayUsdtWithdraw'],
]

const statMeta = {
    totalUserR: { label: '注册人数', icon: 'team' },
    totalUser: { label: '报单人数', icon: 'team' },
    todayUserR: { label: '今日注册', icon: 'team' },
    todayUser: { label: '今日报单人数', icon: 'team' },
    buyTotal: { label: '报单总额', icon: 'pay-circle' },
    todayBuy: { label: '今日报单', icon: 'pay-circle' },
    totalUsdtChainRecharge: { label: '总链上充值（USDT）', icon: 'wallet' },
    todayUsdtChainRecharge: { label: '今日链上充值（USDT）', icon: 'wallet' },
    totalWinChainRecharge: { label: '总链上充值（WIN）', icon: 'wallet' },
    todayWinChainRecharge: { label: '今日链上充值（WIN）', icon: 'wallet' },
    totalWinAChainRecharge: { label: '总链上充值（WIN-A）', icon: 'wallet' },
    todayWinAChainRecharge: { label: '今日链上充值（WIN-A）', icon: 'wallet' },
    totalRewardReinvest: { label: '总奖励复投', icon: 'redo' },
    todayRewardReinvest: { label: '今日奖励复投', icon: 'redo' },
    totalDynamic: { label: '总动态', icon: 'rise' },
    todayDynamic: { label: '今日动态', icon: 'rise' },
    totalStaticRelease: { label: '总静态释放', icon: 'bar-chart' },
    yesterdayStaticRelease: { label: '昨日静态释放', icon: 'bar-chart' },
    totalWinWithdraw: { label: '总WIN提现', icon: 'export' },
    todayWinWithdraw: { label: '今日WIN提现', icon: 'export' },
    totalSdtWithdraw: { label: '总AIX-USDT提现', icon: 'export' },
    todaySdtWithdraw: { label: '今日AIX-USDT提现', icon: 'export' },
    totalSdtAsset: { label: '总AIX-USDT资产', icon: 'fund' },
    totalWinAsset: { label: '总WIN资产', icon: 'fund' },
    totalAdminRecharge: { label: '总后台手动充值', icon: 'plus-circle' },
    todayAdminRecharge: { label: '今日后台手动充值', icon: 'plus-circle' },
    totalZeroAccountReward: { label: '总零号账户累计金额', icon: 'gift' },
    todayZeroAccountReward: { label: '今日零号账户金额', icon: 'gift' },
    totalCommunitySubsidyReward: { label: '总社区补贴累计金额', icon: 'gift' },
    todayCommunitySubsidyReward: { label: '今日社区补贴金额', icon: 'gift' },
    totalUsdtWithdrawable: { label: '全网可提U', icon: 'wallet' }, // 零号+社区补贴合计
    totalUsdtWithdraw: { label: '总提现U', icon: 'export' }, // 零号+社区补贴提现合计
    todayUsdtWithdraw: { label: '今日提现U', icon: 'export' }, // 今日零号+社区补贴提现
}

export default {
    name: 'home',
    data() {
        return {
            loading: true,
            data: {},
            statRows,
            statMeta,
        }
    },
    activated() {
        this.getList()
    },
    methods: {
        getList() {
            this.loading = true
            Gai.all().then(res => {
                this.data = res || {}
                this.loading = false
            }).catch(() => {
                this.loading = false
            })
        },
        formatValue(key) {
            const value = this.data[key]
            return value == null || value === '' ? 0 : value
        },
    }
}
</script>

<style scoped lang="less">
.cardCon {
    /deep/ .ant-card-body {
        padding: 0 !important;
    }

    .ant-card-grid {
        padding: 24px 15px;
        width: 100%;
        display: flex;
        align-items: center;
        justify-content: space-between;

        i {
            font-size: 20px;
            color: #ffffff;
            padding: 10px;
            border-radius: 8px;
            background: #1890ff;
        }
    }
}
</style>
