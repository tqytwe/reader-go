import { Button, Tooltip } from 'antd'
import { MoonOutlined, SunOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { useAppTheme, APP_THEME_LABELS, type AppTheme } from '../hooks/useAppTheme'

const ICONS: Record<AppTheme, React.ReactNode> = {
  light: <SunOutlined />,
  dark: <MoonOutlined />,
  eye: <ThunderboltOutlined />,
}

export default function ThemeToggle() {
  const { theme, cycleTheme } = useAppTheme()

  return (
    <Tooltip title={`主题：${APP_THEME_LABELS[theme]}（点击切换）`}>
      <Button type="text" icon={ICONS[theme]} onClick={cycleTheme} aria-label="切换主题" />
    </Tooltip>
  )
}
