import { formatAmount4 } from '../../utils/formatAmount'

export default {
    methods: {
        appendTeamQueryParams (params) {
            if (this.searchData && this.searchData.teamQuery) {
                params.teamQuery = '1'
            }
            return params
        },
        resetTeamQuerySearch () {
            if (this.searchData) {
                this.searchData.teamQuery = false
            }
        },
        formatAmount4,
        teamSummaryText (summary) {
            if (!summary) return ''
            const parts = [
                `团队人数 ${summary.memberCount || 0}`,
                `团队业绩 ${formatAmount4(summary.teamPerformance)}`,
                `小区业绩 ${formatAmount4(summary.smallAreaPerformance)}`,
                `大区业绩 ${formatAmount4(summary.largeAreaPerformance)}`,
            ]
            if (summary.communitySubsidyTotal != null && summary.communitySubsidyTotal !== '') {
                parts.push(`社区补贴累计 ${formatAmount4(summary.communitySubsidyTotal)}`)
            }
            if (summary.communitySubsidyRate > 0) {
                parts.push(`补贴档位 ${summary.communitySubsidyRate}%`)
            }
            return parts.join('  |  ')
        },
    },
}
