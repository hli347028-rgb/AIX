<template>
    <Editor :id="editorId" v-model="tinymceHtml" :init="editorInit" />
</template>

<script>
    import tinymce from 'tinymce/tinymce'
    import Editor from '@tinymce/tinymce-vue'
    // 引入富文本编辑器主题的js和css
    import 'tinymce/themes/silver/theme'
    import 'tinymce/icons/default/icons.min.js'
    // 扩展插件
    import 'tinymce/plugins/image'
    import 'tinymce/plugins/link'
    import 'tinymce/plugins/code'
    import 'tinymce/plugins/table'
    import 'tinymce/plugins/lists'
    import 'tinymce/plugins/wordcount'

    // admin 部署在 /admin 下，皮肤与语言包必须走 publicPath，否则编辑区空白无法输入
    const publicBase = String(process.env.BASE_URL || '/admin/').replace(/\/?$/, '/')

    export default {
        name: 'tinymceForm',
        components: { Editor },
        props: {
            value: {
                type: String,
                default: ''
            },
            height: {
                type: Number,
                default: 500
            }
        },
        data () {
            return {
                editorId: `tinymce-${Date.now()}-${Math.floor(Math.random() * 10000)}`,
                tinymceHtml: this.value,
            }
        },
        computed: {
            editorInit () {
                return {
                    language_url: `${publicBase}tinymce/langs/zh_CN.js`,
                    language: 'zh_CN',
                    skin_url: `${publicBase}tinymce/skins/ui/oxide`,
                    content_css: `${publicBase}tinymce/skins/content/default/content.min.css`,
                    height: this.height,
                    mobile: {
                        menubar: true,
                        toolbar_drawer: true,
                    },
                    plugins: 'link lists image code table wordcount',
                    toolbar: 'undo redo | bold italic underline strikethrough | fontsizeselect | forecolor backcolor | alignleft aligncenter alignright alignjustify | bullist numlist | outdent indent blockquote | link unlink code',
                    images_upload_handler: (blobInfo, success, failure) => {
                        this.handleImgUpload(blobInfo, success, failure)
                    },
                    statusbar: true,
                    menubar: true,
                    branding: false,
                    toolbar_drawer: true,
                    convert_urls: false,
                }
            }
        },
        watch: {
            value (v) {
                if (v !== this.tinymceHtml) {
                    this.tinymceHtml = v
                }
            },
            tinymceHtml (v) {
                this.$emit('input', v)
            }
        },
        methods: {
            handleImgUpload (blobInfo, success, failure) {
                // 图片上传接口未接入时，避免抛错导致编辑器异常；可粘贴/插入外链图片
                failure('暂不支持本地图片上传，请使用图片链接')
            }
        },
    }
</script>

<style scoped lang="less">
    @import "tinymceForm";
</style>
