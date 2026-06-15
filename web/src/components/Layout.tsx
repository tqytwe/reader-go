import { Layout as AntLayout, Input } from 'antd'
import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom'
import ThemeToggle from './ThemeToggle'
import {
  BookOutlined,
  SettingOutlined,
  HomeOutlined,
  SearchOutlined,
  GlobalOutlined,
  CompassOutlined,
  FilterOutlined,
  FolderOpenOutlined,
  CloudSyncOutlined,
} from '@ant-design/icons'

const { Header, Sider, Content } = AntLayout
const { Search: AntSearch } = Input

const menuItems = [
  { key: '/', icon: <HomeOutlined />, label: '首页' },
  { key: '/bookshelf', icon: <BookOutlined />, label: '我的书架' },
  { key: '/search', icon: <SearchOutlined />, label: '搜索' },
  { key: '/booksource', icon: <SettingOutlined />, label: '书源管理' },
  { key: '/explore', icon: <CompassOutlined />, label: '书海' },
  { key: '/rss', icon: <GlobalOutlined />, label: 'RSS 订阅' },
  { key: '/replaceRules', icon: <FilterOutlined />, label: '替换规则' },
  { key: '/localBooks', icon: <FolderOpenOutlined />, label: '本地书' },
  { key: '/sync', icon: <CloudSyncOutlined />, label: '数据同步' },
]

export default function Layout() {
  const location = useLocation()
  const navigate = useNavigate()

  const handleSearch = (value: string) => {
    if (value.trim()) {
      navigate(`/search?q=${encodeURIComponent(value)}`)
    }
  }

  return (
    <AntLayout className="min-h-screen app-shell-content">
      <Sider width={200} className="app-shell-sider">
        <div className="p-4 text-center" style={{ borderBottom: '1px solid var(--app-border)' }}>
          <h1 className="text-lg font-bold" style={{ color: 'var(--app-text)' }}>Reader Go</h1>
        </div>
        <nav className="p-2">
          {menuItems.map((item) => (
            <Link
              key={item.key}
              to={item.key}
              className={`app-nav-link flex items-center gap-2 px-3 py-2 rounded-lg mb-1 ${
                location.pathname === item.key ? 'app-nav-link-active' : ''
              }`}
            >
              {item.icon}
              <span>{item.label}</span>
            </Link>
          ))}
        </nav>
      </Sider>
      <AntLayout>
        <Header className="app-shell-header">
          <div className="app-shell-header-inner">
            <div className="flex-1" />
            <div className="flex-[2] max-w-xl w-full">
              <AntSearch
                placeholder="搜索书名、作者..."
                allowClear
                onSearch={handleSearch}
                className="w-full"
              />
            </div>
            <div className="flex-1 flex justify-end">
              <ThemeToggle />
            </div>
          </div>
        </Header>
        <Content className="p-0 app-shell-content">
          <Outlet />
        </Content>
      </AntLayout>
    </AntLayout>
  )
}
