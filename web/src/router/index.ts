import { createRouter, createWebHashHistory } from "vue-router";

/**
 * 金牛主路径路由（与后端 app_server 已对接能力对齐）。
 * 未接老页统一重定向到首页，避免深链进 404/坏接口。
 * 页面均为异步 chunk，降低首包体积。
 */
const mainRoutes = [
  { path: "/", component: () => import("@/views/index.vue") },
  { path: "/recharge", component: () => import("@/views/recharge.vue") },
  { path: "/transfer", component: () => import("@/views/transfer.vue") },
  { path: "/node", component: () => import("@/views/node.vue") },
  { path: "/community", component: () => import("@/views/community.vue") },
  { path: "/wallet", component: () => import("@/views/subpage/wallet.vue") },
  { path: "/exchange", component: () => import("@/views/exchange.vue") },
  { path: "/withdrawal", component: () => import("@/views/withdrawal/index.vue") },
  { path: "/count", component: () => import("@/views/share/count.vue") },
  { path: "/profile", component: () => import("@/views/profile/index.vue") },
  { path: "/mine", component: () => import("@/views/mine.vue") },
  { path: "/announcements", component: () => import("@/views/announcements.vue") },
  { path: "/support", component: () => import("@/views/support.vue") },
  { path: "/rules", component: () => import("@/views/rules.vue") },
  { path: "/futurefi", component: () => import("@/views/futurefi.vue") },
];

/** 旧路径 → 主路径（质押入口并到认购） */
const redirects: Array<{ path: string; redirect: string }> = [
  { path: "/pledge", redirect: "/node" },
  { path: "/home", redirect: "/" },
  { path: "/index", redirect: "/" },
  { path: "/idoDetails", redirect: "/" },
  { path: "/payment", redirect: "/" },
  { path: "/trade", redirect: "/" },
  { path: "/contact", redirect: "/support" },
  { path: "/address", redirect: "/" },
  { path: "/shop", redirect: "/" },
  { path: "/Web3Shop", redirect: "/" },
  { path: "/order", redirect: "/" },
  { path: "/order/:id", redirect: "/" },
  { path: "/powerShop", redirect: "/" },
  { path: "/stat", redirect: "/" },
  { path: "/level", redirect: "/" },
  { path: "/powerOrder", redirect: "/" },
  { path: "/withdraw/:type", redirect: "/withdrawal" },
];

const routes = [
  ...mainRoutes,
  ...redirects,
  { path: "/:pathMatch(.*)*", redirect: "/" },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;
