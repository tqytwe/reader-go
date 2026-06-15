# Web 前端项目规范

## 项目信息

- **框架**: React 18 + TypeScript
- **构建工具**: Vite 5
- **UI 组件库**: Ant Design 5
- **状态管理**: Zustand 4
- **路由**: React Router 6

## 目录结构

```
web/
├── src/
│   ├── api/              # API 客户端
│   │   └── client.ts    # Axios 实例 + API 方法
│   ├── components/      # 公共组件
│   │   └── Layout.tsx    # 布局组件
│   ├── pages/            # 页面组件
│   │   ├── Home.tsx      # 首页
│   │   ├── Bookshelf.tsx # 书架
│   │   ├── Search.tsx    # 搜索
│   │   ├── Reader.tsx    # 阅读器
│   │   ├── BookSourceManage.tsx  # 书源管理
│   │   ├── BookSourceDebug.tsx    # 书源调试
│   │   └── Rss.tsx       # RSS 订阅
│   ├── store/            # Zustand 状态管理
│   │   └── useStore.ts   # 主 store
│   ├── types/            # TypeScript 类型
│   │   └── index.ts      # 类型定义
│   ├── router.tsx        # 路由配置
│   ├── App.tsx           # 根组件
│   ├── main.tsx          # 入口文件
│   └── index.css         # 全局样式
├── package.json
├── tsconfig.json
├── vite.config.ts
├── tailwind.config.js
└── postcss.config.js
```

## 开发命令

```bash
npm install          # 安装依赖
npm run dev          # 开发服务器 (localhost:5173)
npm run build        # 生产构建
npm run preview      # 预览构建结果
npm run lint         # ESLint 检查
npm run test         # 运行测试
```

## 代码规范

### 组件规范

1. **文件命名**: 使用 PascalCase (如 `BookSourceManage.tsx`)
2. **组件导出**: 使用命名导出 `export default function ...`
3. **类型定义**: 放在 `types/index.ts` 或组件同目录的 `types.ts`

### Hooks 规范

1. 优先使用 Zustand 管理全局状态
2. 组件本地状态使用 `useState`
3. 副作用使用 `useEffect`
4. 计算值使用 `useMemo`

### API 调用规范

```typescript
// 正确
const response = await api.getBookSources()
const sources = (response as any)?.data ?? []
useStore.getState().setBookSources(Array.isArray(sources) ? sources : [])

// 错误 - 没有空值检查
const { data } = await api.getBookSources()
setBookSources(data) // data 可能是 null
```

### 数组操作规范

```typescript
// 必须进行空值检查
const safeArray = Array.isArray(books) ? books : []
safeArray.map(...)
safeArray.filter(...)
```

## 状态管理

### Store 结构

```typescript
// useStore - 主 store
{
  books: ShelfBook[]         # 书架书籍
  bookSources: BookSource[]  # 书源列表
  replaceRules: any[]         # 替换规则
  bookSourcesLoading: boolean
  bookSourceSearchKeyword: string
  bookSourceGroupFilter: string
}

// useShelfStore - 书架专用
{
  books: ShelfBook[]
  setBooks, addBook, removeBook, ...
}

// useSearchStore - 搜索专用
{
  searchHistory: SearchHistoryItem[]
  addSearchHistory, clearSearchHistory, ...
}
```

## 样式规范

- 使用 Tailwind CSS 进行样式开发
- 主题变量定义在 `index.css`
- 组件样式优先使用 CSS 变量

## 页面路由

| 路径 | 组件 | 描述 |
|------|------|------|
| `/` | Home | 首页 |
| `/bookshelf` | Bookshelf | 书架 |
| `/search` | Search | 搜索 |
| `/booksource` | BookSourceManage | 书源管理 |
| `/booksource/debug` | BookSourceDebug | 书源调试 |
| `/reader/:bookId` | Reader | 阅读器 |
| `/rss` | Rss | RSS 订阅 |