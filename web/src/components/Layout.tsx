import { useState, useEffect } from 'react'
import { Layout as AntLayout, Input, Button, Tooltip } from 'antd'
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
  MenuFoldOutlined,
  MenuUnfoldOutlined,
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

const SIDEBAR_COLLAPSED_KEY = 'sidebar_collapsed'

export default function Layout() {
  const location = useLocation()
  const navigate = useNavigate()
  const [collapsed, setCollapsed] = useState(() => {
    try {
      return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === '1'
    } catch {
      return false
    }
  })

  // 移动端自动折叠
  useEffect(() => {
    const handleResize = () => {
      if (window.innerWidth < 768 && !collapsed) {
        setCollapsed(true)
      }
    }
    window.addEventListener('resize', handleResize)
    // Check on mount
    handleResize()
    return () => window.removeEventListener('resize', handleResize)
  }, [collapsed])

  const toggleCollapsed = () => {
    const next = !collapsed
    setCollapsed(next)
    try {
      localStorage.setItem(SIDEBAR_COLLAPSED_KEY, next ? '1' : '0')
    } catch {
      // ignore
    }
  }

  const handleSearch = (value: string) => {
    if (value.trim()) {
      navigate(`/search?q=${encodeURIComponent(value)}`)
    }
  }

  const siderWidth = collapsed ? 60 : 200

  return (
    <AntLayout className="min-h-screen app-shell-content">
      <Sider
        width={siderWidth}
        className="app-shell-sider transition-all duration-200"
        style={{ width: siderWidth, minWidth: siderWidth, maxWidth: siderWidth }}
        collapsed={collapsed}
        collapsedWidth={60}
        trigger={null}
      >
        <div
          className="flex items-center justify-center relative"
          style={{
            height: 48,
            borderBottom: '1px solid var(--app-border)',
          }}
        >
          {!collapsed && (
            <h1 className="text-lg font-bold m-0" style={{ color: 'var(--app-text)' }}>
              Reader Go
            </h1>
          )}
          {collapsed && (
            <h1 className="text-base font-bold m-0" style={{ color: 'var(--app-text)' }}>
              RG
            </h1>
          )}
        </div>
        <nav className="p-2">
          {menuItems.map((item) => {
            const isActive = location.pathname === item.key
            const link = (
              <Link
                key={item.key}
                to={item.key}
                className={`app-nav-link flex items-center gap-2 px-3 py-2 rounded-lg mb-1 transition-colors ${
                  isActive ? 'app-nav-link-active' : ''
                }`}
                style={{
                  justifyContent: collapsed ? 'center' : 'flex-start',
                }}
              >
                <span className="text-base">{item.icon}</span>
                {!collapsed && <span className="text-sm">{item.label}</span>}
              </Link>
            )
            if (collapsed) {
              return (
                <Tooltip key={item.key} title={item.label} placement="right">
                  {link}
                </Tooltip>
              )
            }
            return link
          })}
        </nav>
      </Sider>
      <AntLayout>
        <Header className="app-shell-header">
          <div className="app-shell-header-inner">
            <div className="flex-1 flex items-center">
              <Button
                type="text"
                icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                onClick={toggleCollapsed}
                title={collapsed ? '展开侧边栏' : '折叠侧边栏'}
              />
            </div>
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
