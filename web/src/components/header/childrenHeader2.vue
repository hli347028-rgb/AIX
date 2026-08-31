<template>
  <div class="children-header">
    <div class="children-header-main">
      <div class="children-header-left">
        <i class="return-button" @click="handleMenuChange ? handleMenuChange(1) : router.back()">{{ $t('common.back') }}</i>
      </div>
      <div class="children-header-center">
        <!-- 同 childrenHeader.vue：logo 此前 import 了但未绑定到模板，
             顶栏中间一直是空白。同样统一为蓝色 AIX 主标。 -->
        <i class="logo">
          <img src="/assets/logo.png" alt="AIX" />
        </i>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ref } from 'vue'

const router = useRouter()
const { t: $t } = useI18n()

const props = defineProps({
  handleMenuChange: Function
})

</script>
<style scoped lang="less">
@import "./childrenHeader.less";
</style>
