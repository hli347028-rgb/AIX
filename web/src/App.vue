<template>
    <a-config-provider :theme="theme">
        <div id="app">
            <template v-if="person.isLogin">
                <div v-if="renderError" class="page-error">
                    <Header />
                    <div class="page-error__body">
                        <strong>页面加载失败</strong>
                        <span>Page failed to load</span>
                        <button type="button" @click="reloadPage">重新加载 / Reload</button>
                    </div>
                </div>
                <router-view v-else></router-view>
            </template>
            <Login v-else />
        </div>
    </a-config-provider>
</template>
<script setup lang="ts">
import userPerson from "@/pinia/person";
import userSystem from "@/pinia/system";
import { theme as antdTheme } from 'ant-design-vue'
import { onErrorCaptured, onMounted, nextTick, ref } from "vue"
import Header from '@/components/Header.vue'

const theme = {
  algorithm: antdTheme.darkAlgorithm
}

const person = userPerson();
const system = userSystem();
const renderError = ref(false)
system.initTime();

onErrorCaptured((error) => {
    console.error('[App:render]', error)
    renderError.value = true
    return false
})

const reloadPage = () => window.location.reload()

onMounted(async () => {
    await nextTick();
    await person.init().catch((error) => {
        // person store 已将错误转换为可重试状态，此处仅收口 Promise rejection。
        console.error('[App:init]', error)
    });
})
</script>
<style>
#app {
    width: 100%;
    height: 100%;
}
.page-error {
    position: fixed;
    inset: 0;
    min-height: 100vh;
    background: #020817;
    color: #f5f9ff;
}
.page-error__body {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-direction: column;
    gap: 12px;
    min-height: 100vh;
    padding: 80px 24px 24px;
    box-sizing: border-box;
}
.page-error__body strong { font-size: 18px; }
.page-error__body span { color: #9fb0c6; font-size: 12px; }
.page-error__body button {
    margin-top: 8px;
    padding: 10px 20px;
    border: 1px solid #55b4ff;
    border-radius: 20px;
    background: #087eff;
    color: #fff;
}
</style>
