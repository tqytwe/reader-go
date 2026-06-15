import { useEffect } from 'react'
import { RouterProvider } from 'react-router-dom'
import { ConfigProvider, theme as antTheme } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { router } from './router'
import { useAppTheme } from './hooks/useAppTheme'

function App() {
  const { theme } = useAppTheme()

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
  }, [theme])

  const isDark = theme === 'dark'
  const isEye = theme === 'eye'

  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm: isDark ? antTheme.darkAlgorithm : antTheme.defaultAlgorithm,
        token: isEye
          ? {
              colorBgContainer: '#f0e6d6',
              colorBgLayout: '#e8dcc8',
              colorText: '#3b3228',
              colorBorder: '#d4c4a8',
            }
          : undefined,
      }}
    >
      <RouterProvider router={router} />
    </ConfigProvider>
  )
}

export default App
