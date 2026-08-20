import { axios } from '@/utils/request'
const api = `${projectUrl}/api/admin_dhb`
export default {
    getArticle: (parameter) => {
        return axios({
            url: `${api}/announcement_list`,
            method: 'get',
            params: parameter
        })
    },
    addArticle: (parameter) => {
        return axios({
            url: `${api}/announcement_save`,
            method: 'post',
            data: parameter
        })
    },
    deleteArticle: (parameter) => {
        return axios({
            url: `${api}/announcement_delete`,
            method: 'post',
            data: parameter
        })
    },
    changeArticle: (parameter) => {
        return axios({
            url: `${api}/announcement_save`,
            method: 'post',
            data: parameter
        })
    },
    getArticleDetails: (parameter) => {
        return axios({
            url: `${api}/announcement_detail`,
            method: 'post',
            data: parameter
        })
    },
}
