import { axios } from '@/utils/request'
const api = `${projectUrl}/api/admin_dhb`
export default {
    getList: (parameter) => {
        return axios({
            url: `${api}/feedback_list`,
            method: 'get',
            params: parameter
        })
    },
    updateStatus: (parameter) => {
        return axios({
            url: `${api}/feedback_status`,
            method: 'post',
            data: parameter
        })
    },
}
