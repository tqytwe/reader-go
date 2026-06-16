import { useEffect, useState } from 'react'
import { RouterProvider } from 'react-router-dom'
import { ConfigProvider, theme as antTheme } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { router } from './router'
import { useAppTheme } from './hooks/useAppTheme'
import { useLockStore } from './store/useStore'
import LockScreen from './components/LockScreen'

function App() {
  const { theme } = useAppTheme()
  const { isPasswordSet, setPassword, verifyPassword, isUnlocked, unlock } = useLockStore()
  // 首次访问也需要显示锁屏（设置模式）
  const [unlocked, setUnlocked] = useState(() => isUnlocked())

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
  }, [theme])

  // 当解锁状态变化时同步
  useEffect(() => {
    if (isUnlocked()) {
      setUnlocked(true)
    }
  }, [isUnlocked])

  const handleUnlock = () => {
    unlock()
    setUnlocked(true)
  }

  const handleSetPassword = async (password: string) => {
    await setPassword(password)
    unlock()
    setUnlocked(true)
  }

  const handleVerifyPassword = async (password: string): Promise<boolean> => {
    const success = await verifyPassword(password)
    if (success) {
      unlock()
      setUnlocked(true)
    }
    return success
  }

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
      {!unlocked && (
        <LockScreen
          onUnlock={handleUnlock}
          isPasswordSet={isPasswordSet}
          onSetPassword={handleSetPassword}
          onVerifyPassword={handleVerifyPassword}
        />
      )}
      <RouterProvider router={router} />
    </ConfigProvider>
  )
}

export default App
