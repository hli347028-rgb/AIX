<template>
    <a-config-provider :theme="theme">
        <div id="app">
            <router-view v-if="person.isLogin"></router-view>
            <Login v-else />
        </div>
    </a-config-provider>
</template>
<script setup lang="ts">
import userPerson from "@/pinia/person";
import userSystem from "@/pinia/system";
import { theme as antdTheme } from 'ant-design-vue'
import { onMounted, nextTick } from "vue"

const theme = {
  algorithm: antdTheme.darkAlgorithm
}

const person = userPerson();
const system = userSystem();
system.initTime();

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
</style>
