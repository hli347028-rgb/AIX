# 公告功能 — 前端文档

本文说明 AIX 项目中「公告」在管理端与用户端（H5）的前端结构、页面流程与接口约定。

---

## 1. 功能概览

| 端 | 能力 | 入口 |
|---|---|---|
| 管理端 `admin/` | 公告列表、新增、编辑、删除 | 侧边栏「公告列表」→ `/news` |
| 用户端 `web/` | 查看已发布公告列表（点击展开正文） | 侧边栏「公告」→ `/#/announcements` |

数据流：管理端写入 → 后端 `announcements` 表 → 用户端只读拉取 `status=1` 的公告。

---

## 2. 管理端（Vue 2 + Ant Design Vue）

### 2.1 相关文件

| 文件 | 说明 |
|---|---|
| `admin/src/config/myRouter.js` | 路由 `/news`、`/newsEdit` |
| `admin/src/views/public/news.vue` | 公告列表页 |
| `admin/src/views/public/newsEdit.vue` | 新增 / 编辑页（TinyMCE 富文本） |
| `admin/src/api/Art.js` | 公告 API 封装 |
| `admin/src/components/tinymceForm/tinymceForm.vue` | 富文本编辑器 |

### 2.2 路由

```js
// myRouter.js（摘录）
{ path: '/news', name: 'news', meta: { title: '公告列表' } }
{ path: '/newsEdit', name: 'newsEdit', hidden: true, meta: { title: '编辑公告' } }
```

- 列表：`/news`
- 新增：跳转 `newsEdit`（无 `id`）
- 编辑：跳转 `newsEdit?id=<公告ID>`

### 2.3 页面行为

**列表页 `news.vue`**

1. 进入页面（`activated` via `listMixin`）调用 `Art.getArticle({ page, num })`
2. 表格展示：`title`、`add_time`（经全局 `timeOne` 格式化为本地时间）
3. 操作：编辑 / 删除（确认后 `Art.deleteArticle({ id })`）
4. 「添加公告」→ `router.push({ name: 'newsEdit' })`

**编辑页 `newsEdit.vue`**

1. 有 `query.id` 时：`Art.getArticleDetails({ id })` 回填 `title`、`content`
2. 提交：`Art.addArticle({ id?, title, content })`  
   - 有 `id`：更新  
   - 无 `id`：新建  
3. 成功后 `router.back()` 回到列表

> POST 请求体经 `admin/src/utils/request.js` 转为 `application/x-www-form-urlencoded`（`qs.stringify`）。

### 2.4 API 封装（`Art.js`）

基址：`${projectUrl}/api/admin_dhb`（需管理端登录 Token：`Authorization: Bearer <token>`）

| 方法 | HTTP | 路径 | 用途 |
|---|---|---|---|
| `getArticle` | GET | `/announcement_list` | 分页列表 |
| `getArticleDetails` | POST | `/announcement_detail` | 详情 |
| `addArticle` / `changeArticle` | POST | `/announcement_save` | 新建或更新 |
| `deleteArticle` | POST | `/announcement_delete` | 删除 |

**列表请求参数**

| 参数 | 类型 | 说明 |
|---|---|---|
| `page` | number | 页码，从 1 开始 |
| `num` | number | 每页条数（对应后端 `num` / 分页） |

**列表响应（前端使用字段）**

```json
{
  "data": [
    {
      "id": 1,
      "title": "标题",
      "content": "<p>HTML 正文</p>",
      "add_time": 1720000000,
      "created_at": "2026-08-20 16:00:00",
      "status": 1
    }
  ],
  "count": 1,
  "page": 1
}
```

**保存请求字段**

| 字段 | 必填 | 说明 |
|---|---|---|
| `id` | 编辑时必填 | 有值则更新，无值则新建 |
| `title` | 是 | 标题 |
| `content` | 是 | 富文本 HTML |

**详情响应**

```json
{
  "data": {
    "id": 1,
    "title": "标题",
    "content": "<p>...</p>",
    "add_time": 1720000000
  }
}
```

---

## 3. 用户端 H5（Vue 3 + Vite + Vant）

### 3.1 相关文件

| 文件 | 说明 |
|---|---|
| `web/src/router/index.ts` | 路由 `/announcements` |
| `web/src/views/announcements.vue` | 公告列表页 |
| `web/src/components/Sidebar.vue` | 侧边栏入口 |
| `web/src/api/aix.ts` | `listAnnouncements` / `getAnnouncementDetail` |
| `web/src/i18n/lang/*.ts` | `announcement.*` 文案 |
| `web/src/components/AnnouncementModal.vue` | 旧版弹窗组件（静态 i18n 文案，**未接后端**，当前未挂载到主路径） |

### 3.2 路由与入口

```ts
// router/index.ts
{ path: "/announcements", component: () => import("@/views/announcements.vue") }
```

Hash 模式下完整路径示例：`https://<域名>/#/announcements`

侧边栏（`Sidebar.vue`）增加「公告」菜单项，点击 `go('/announcements')`。

### 3.3 页面行为（`announcements.vue`）

1. `onMounted` 调用 `listAnnouncements({ page: 1, page_size: 50 })`
2. 加载中显示 `van-loading`；无数据显示 `announcement.empty`
3. 列表默认展开第一条；点击条目切换展开/收起
4. 正文使用 `v-html` 渲染管理端富文本（需注意 XSS：内容来自可信管理端）
5. 失败时 `showToast(errMsg(...))`

### 3.4 API 封装（`aix.ts`）

基址：`VITE_API`（见 `web/.env.prod`），请求可带用户 Token（公告接口本身不强制登录）。

```ts
export interface AnnouncementItem {
  id?: number
  title?: string
  content?: string
  created_at?: string
  add_time?: number
}

listAnnouncements(params?: { page?: number; page_size?: number })
getAnnouncementDetail(id: number | string)
```

| 方法 | HTTP | 路径 | 说明 |
|---|---|---|---|
| `listAnnouncements` | GET | `/v1/announcements` | 已发布列表 |
| `getAnnouncementDetail` | GET | `/v1/announcement/detail?id=` | 单条详情（页面当前未用，可扩展详情页） |

**列表响应示例**

```json
{
  "list": [
    {
      "id": 1,
      "title": "系统升级通知",
      "content": "<p>今晚 22:00 维护</p>",
      "created_at": "2026-08-20 16:00:00",
      "add_time": 1724130000
    }
  ],
  "count": 1,
  "page": 1
}
```

### 3.5 国际化（`announcement`）

| Key | 中文（zh） | 用途 |
|---|---|---|
| `title` | 公告 | 页标题 / 侧边栏 |
| `empty` | 暂无公告 | 空列表 |
| `fetchFailed` | 获取公告失败 | 请求失败 Toast |
| `gotIt` / `noRemind` / `content` | … | 仅旧弹窗 `AnnouncementModal` 使用 |

语言文件：`zh.ts`、`en.ts`、`zh-tw.ts`、`ja.ts`、`ko.ts`、`vi.ts`。

---

## 4. 联调检查清单

管理端：

- [ ] 登录后侧边栏可见「公告列表」
- [ ] 新增公告（标题 + 富文本）提交成功并回到列表
- [ ] 编辑、删除生效
- [ ] 列表时间显示正常

用户端：

- [ ] 侧边栏可进入公告页
- [ ] 管理端发布后 H5 能看到对应标题与 HTML 内容
- [ ] 空列表显示「暂无公告」
- [ ] 多语言切换下文案正确

接口冒烟（生产示例）：

```bash
curl -sS 'https://aixai.pro/v1/announcements?page=1&page_size=10'
```

---

## 5. 注意事项

1. **内容格式**：管理端 TinyMCE 产出 HTML；H5 用 `v-html` 展示，样式已对 `img` / `p` / `a` 做基础适配。
2. **命名遗留**：管理端文件/API 仍叫 `news` / `Art` / `Article`，实际对接的是 `announcement_*` 接口，改名非必须。
3. **`AnnouncementModal.vue`**：静态文案弹窗，未接入 `/v1/announcements`；若要做「首次进入弹最新公告」，可在该组件内调用 `listAnnouncements` 取第一条。
4. **鉴权**：管理端接口需管理员 Token；用户端列表/详情为公开读接口。
5. **分页**：H5 当前一次拉最多 50 条，未做「加载更多」；需要时可扩展 `page` 翻页。

---

## 6. 目录速查

```
admin/
  src/api/Art.js
  src/config/myRouter.js
  src/views/public/news.vue
  src/views/public/newsEdit.vue

web/
  src/api/aix.ts                    # listAnnouncements / getAnnouncementDetail
  src/router/index.ts               # /announcements
  src/views/announcements.vue
  src/components/Sidebar.vue
  src/components/AnnouncementModal.vue  # 未接后端
  src/i18n/lang/*.ts
```
