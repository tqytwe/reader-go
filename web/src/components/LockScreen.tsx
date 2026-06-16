import { useState, useRef, useEffect } from 'react'
import { Input, Button, Typography } from 'antd'
import { LockOutlined } from '@ant-design/icons'

const { Title, Text } = Typography

interface LockScreenProps {
  onUnlock: () => void
  isPasswordSet: boolean
  onSetPassword: (password: string) => void | Promise<void>
  onVerifyPassword: (password: string) => boolean | Promise<boolean>
}

export default function LockScreen({ onUnlock, isPasswordSet, onSetPassword, onVerifyPassword }: LockScreenProps) {
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [shaking, setShaking] = useState(false)
  const [mode, setMode] = useState<'unlock' | 'setup' | 'setupConfirm'>(
    isPasswordSet ? 'unlock' : 'setup'
  )
  const inputRef = useRef<any>(null)

  useEffect(() => {
    // Auto-focus input
    const timer = setTimeout(() => {
      inputRef.current?.focus()
    }, 100)
    return () => clearTimeout(timer)
  }, [mode])

  const shake = () => {
    setShaking(true)
    setTimeout(() => setShaking(false), 500)
  }

  const handleUnlock = async () => {
    if (!password) {
      setError('请输入密码')
      shake()
      return
    }
    const success = await onVerifyPassword(password)
    if (success) {
      setError('')
      onUnlock()
    } else {
      setError('密码错误，请重试')
      setPassword('')
      shake()
    }
  }

  const handleSetup = () => {
    if (!password || password.length < 4) {
      setError('密码至少 4 个字符')
      shake()
      return
    }
    setMode('setupConfirm')
    setError('')
  }

  const handleConfirmSetup = () => {
    if (confirmPassword !== password) {
      setError('两次输入的密码不一致')
      setConfirmPassword('')
      shake()
      return
    }
    onSetPassword(password)
    onUnlock()
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      if (mode === 'unlock') handleUnlock()
      else if (mode === 'setup') handleSetup()
      else if (mode === 'setupConfirm') handleConfirmSetup()
    }
  }

  return (
    <div className="fixed inset-0 z-[9999] flex items-center justify-center"
      style={{ background: 'var(--app-bg, #f5f5f5)' }}
    >
      <div
        className={`w-full max-w-sm mx-4 p-8 rounded-2xl shadow-xl text-center transition-transform ${
          shaking ? 'animate-shake' : ''
        }`}
        style={{ background: 'var(--app-surface, #fff)' }}
        onKeyDown={handleKeyDown}
      >
        {/* Logo / Icon */}
        <div className="mb-6">
          <div
            className="w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4"
            style={{ background: 'var(--app-accent, #1677ff)', opacity: 0.9 }}
          >
            <LockOutlined style={{ fontSize: 28, color: '#fff' }} />
          </div>
          <Title level={3} style={{ color: 'var(--app-text)', marginBottom: 4 }}>
            Reader Go
          </Title>
          <Text style={{ color: 'var(--app-muted)' }}>
            {mode === 'unlock' && '输入密码以继续'}
            {mode === 'setup' && '设置访问密码'}
            {mode === 'setupConfirm' && '再次确认密码'}
          </Text>
        </div>

        {/* Password Input */}
        {mode === 'unlock' && (
          <div className="space-y-4">
            <Input.Password
              ref={inputRef}
              size="large"
              placeholder="请输入密码"
              value={password}
              onChange={(e) => { setPassword(e.target.value); setError('') }}
              status={error ? 'error' : undefined}
              autoComplete="current-password"
            />
            {error && (
              <Text type="danger" className="text-sm block">{error}</Text>
            )}
            <Button
              type="primary"
              size="large"
              block
              onClick={handleUnlock}
            >
              解锁
            </Button>
          </div>
        )}

        {mode === 'setup' && (
          <div className="space-y-4">
            <Input.Password
              ref={inputRef}
              size="large"
              placeholder="设置密码（至少4位）"
              value={password}
              onChange={(e) => { setPassword(e.target.value); setError('') }}
              status={error ? 'error' : undefined}
              autoComplete="new-password"
            />
            {error && (
              <Text type="danger" className="text-sm block">{error}</Text>
            )}
            <Button
              type="primary"
              size="large"
              block
              onClick={handleSetup}
            >
              下一步
            </Button>
            <Text className="text-xs block" style={{ color: 'var(--app-muted)' }}>
              密码保存在本地浏览器中，用于保护阅读内容不被他人查看
            </Text>
          </div>
        )}

        {mode === 'setupConfirm' && (
          <div className="space-y-4">
            <Input.Password
              ref={inputRef}
              size="large"
              placeholder="再次输入密码"
              value={confirmPassword}
              onChange={(e) => { setConfirmPassword(e.target.value); setError('') }}
              status={error ? 'error' : undefined}
              autoComplete="new-password"
            />
            {error && (
              <Text type="danger" className="text-sm block">{error}</Text>
            )}
            <Button
              type="primary"
              size="large"
              block
              onClick={handleConfirmSetup}
            >
              确认并进入
            </Button>
            <Button
              type="link"
              size="small"
              onClick={() => { setMode('setup'); setPassword(''); setConfirmPassword(''); setError('') }}
            >
              返回重设
            </Button>
          </div>
        )}
      </div>

      {/* Shake animation */}
      <style>{`
        @keyframes shake {
          0%, 100% { transform: translateX(0); }
          10%, 30%, 50%, 70%, 90% { transform: translateX(-4px); }
          20%, 40%, 60%, 80% { transform: translateX(4px); }
        }
        .animate-shake {
          animation: shake 0.5s ease-in-out;
        }
      `}</style>
    </div>
  )
}
