<template>
  <div class="profile-page">
    <ChildrenHeader />
    <ul class="profile">
      <li>
        <label for="profile-language">{{ $t('common.languageSwitch') }}</label>
        <select id="profile-language" v-model="selectedLanguage" class="language-select" @change="onChangeLanguage">
          <option v-for="item in languageOptions" :key="item.code" :value="item.code">{{ item.name }}</option>
        </select>
      </li>
    </ul>
    <button class="switch-user-button" @click="switchUser">{{ $t('common.switchWallet') }}</button>
  </div>
</template>
<script setup lang="ts">
import ChildrenHeader from '../../components/header/childrenHeader.vue'
import { restartCurrentApp } from '@/tools/plaocRuntime'
import { userLanguageOptions } from '@/i18n/languages'
import { useI18n } from 'vue-i18n'
import { ref } from 'vue'
const { t: $t, locale } = useI18n()

const selectedLanguage = ref(String(locale.value))
const languageOptions = userLanguageOptions

const switchUser = async () => {
  localStorage.removeItem("token");
  localStorage.removeItem("account");
  await restartCurrentApp()
}

const onChangeLanguage = () => {
  locale.value = selectedLanguage.value
  localStorage.setItem('lan', selectedLanguage.value)
}

</script>
<style scoped lang="less">
@import "./styles/index.less";

.profile-page {
  width: 100%;
  min-height: 100vh;
}

.language-select {
  min-width: 132px;
  height: 34px;
  padding: 0 30px 0 10px;
  border: 1px solid var(--hair);
  border-radius: var(--r-sm);
  /* 原为 #0d1c29（近黑深蓝）配 color: var(--text)（近黑）——
     深底压深字，这个语言下拉框此前**完全读不出来**。
     成因与全站其它几处一致：背景写死硬编码色、前景已走令牌，
     令牌翻白时背景没跟着走。改为浅灰底 + 近黑字。 */
  background: var(--surface-2);
  color: var(--text);
}
</style>
