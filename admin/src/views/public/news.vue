<template>
    <PageView>
        <a-card title="列表">
            <a-button type="primary" slot="extra" @click="$router.push({name:`newsEdit`})">添加公告</a-button>
            <a-table
                :loading="loading"
                :columns="columns"
                :dataSource="data"
                :pagination="{total,pageSize,showSizeChanger,current}"
                @change="changePagination"
                bordered :scroll="{x:true}">
            </a-table>
        </a-card>
    </PageView>
</template>

<script type="text/jsx">
    import Art from '../../api/Art'
    import listMixin from '../mixin/listMixin'
    export default {
        name: 'news',
        mixins: [listMixin],
        data () {
            return {
                columns: [
                    {
                        title: '排序',
                        dataIndex: 'sort_order',
                        width: 80,
                        customRender: (v) => (v === 0 || v ? v : '-')
                    },
                    {
                        title: '标题',
                        dataIndex: 'title',
                    },
                    {
                        title: '添加时间',
                        dataIndex: 'add_time',
                        customRender: (v) => this.timeOne(v)
                    },
                    {
                        title: '操作',
                        key: 'action',
                        fixed: 'right',
                        width: 200,
                        customRender: (v) => {
                            return (
                                <span>
                                    <a-button size="small" style="margin-right:6px" onClick={() => this.moveItem(v.id, 'up')}>上移</a-button>
                                    <a-button size="small" style="margin-right:6px" onClick={() => this.moveItem(v.id, 'down')}>下移</a-button>
                                    <a-dropdown>
                                        <a-menu slot="overlay">
                                            <a-menu-item onClick={() => {
                                                this.changeBanner(v.id)
                                            }}>删除
                                            </a-menu-item>
                                            <a-menu-item onClick={() => {
                                                this.$router.push({name:"newsEdit",query:{id:v.id}})
                                            }}>编辑
                                            </a-menu-item>
                                        </a-menu>
                                        <a-button size="small">更多 <a-icon type="down"/></a-button>
                                    </a-dropdown>
                                </span>
                            )
                        }
                    }
                ],
            }
        },
        methods: {
            getList () {
                this.loading = true
                Art.getArticle({
                    page: this.current,
                    num: this.pageSize,
                }).then(res => {
                    this.data = res.data.map((value, key) => {
                        return { ...value, key }
                    })
                    this.loading = false
                    this.total = parseInt(res.count)
                })
            },
            moveItem (id, direction) {
                Art.moveArticle({ id, direction }).then(() => {
                    this.getList()
                }).catch(() => {
                    this.$message.error('调整顺序失败')
                })
            },
            changeBanner (id) {
                this.$confirm({
                    title: `删除提示`,
                    content: `确定要删除此公告吗?`,
                    centered: true,
                    onOk: () => {
                        return Art.deleteArticle({id}).then(()=>{
                            this.getList()
                        })
                    }
                })
            }
        }
    }
</script>
