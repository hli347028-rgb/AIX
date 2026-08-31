<template>
  <div class="children-header">
    <div class="children-header-main">
      <div class="children-header-left">
        <i class="return-button" @click="handleMenuChange ? handleMenuChange(1) : router.back()">{{ $t('common.back') }}</i>
      </div>
      <div class="children-header-center">
        <!-- 这里原本是个空的 <i class="logo">：logo 在 script 里 import 了
             但从未绑定到模板，而 childrenHeader.less 里写的是 `.logo img`，
             等着一个根本不存在的 img —— 所以这十几个旧页面的顶栏中间
             一直是空白。

             改用 /assets/logo.png（蓝色 AIX 主标），与顶栏、侧边栏统一；
             原先 import 的 src/assets/images/logo.png 是 AIX×USDT 蓝绿
             联名标，同一个应用里出现两个不同的品牌标识。 -->
        <i class="logo">
          <img src="/assets/logo.png" alt="AIX" />
        </i>
      </div>
      <div class="children-header-right">
        <i class="set-button" v-if="$route.path !== '/profile'" @click="router.push('/profile')"></i>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter, useRoute } from 'vue-router'
import { ref } from 'vue'

const router = useRouter()
const route = useRoute()
const { t: $t } = useI18n()

const props = defineProps({
  handleMenuChange: Function
})

</script>
<style scoped lang="less">
@import "./childrenHeader.less";
</style>
