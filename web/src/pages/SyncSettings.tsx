import { useState } from 'react'
import { Card, Button, Upload, message, Typography, Alert, Space, Divider, Input, Modal } from 'antd'
import { CloudDownloadOutlined, CloudUploadOutlined, SyncOutlined, LockOutlined, DeleteOutlined } from '@ant-design/icons'
import { api } from '../api/client'
import type { SyncBundle } from '../types/booksource'
import { useShelfStore, type ShelfBook, useLockStore } from '../store/useStore'

const { Paragraph, Text } = Typography

export default function SyncSettings() {
  const [exporting, setExporting] = useState(false)
  const [importing, setImporting] = useState(false)
  const { isPasswordSet, setPassword, clearPassword } = useLockStore()
  const [pwdModalOpen, setPwdModalOpen] = useState(false)
  const [newPassword, setNewPassword] = useState('')
  const [confirmPwd, setConfirmPwd] = useState('')

  const handleExport = async () => {
    setExporting(true)
    try {
      const bundle = (await api.syncExport()) as unknown as SyncBundle
      const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `reader-go-sync-${new Date().toISOString().slice(0, 10)}.json`
      a.click()
      URL.revokeObjectURL(url)
      message.success('同步包已导出')
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '导出失败')
    } finally {
      setExporting(false)
    }
  }

  const handleImport = async (file: File) => {
    setImporting(true)
    const reader = new FileReader()
    reader.onload = async () => {
      try {
        const bundle = JSON.parse(reader.result as string)
        await api.syncImport(bundle)
        const shelf = (await api.getShelf()) as { books?: ShelfBook[] }
        useShelfStore.getState().setBooks(shelf?.books ?? [])
        message.success('同步包导入成功，书架与书源已更新')
      } catch (e: unknown) {
        message.error(e instanceof Error ? e.message : '导入失败，请检查 JSON 格式')
      } finally {
        setImporting(false)
      }
    }
    reader.readAsText(file)
    return false
  }

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-2">数据同步</h1>
      <Paragraph type="secondary">
        导出或导入完整数据包（书架、书源、替换规则），便于备份、迁移或手动 WebDAV 同步。
      </Paragraph>

      <Alert
        className="mb-6"
        type="info"
        showIcon
        message="同步包格式"
        description={
          <Text code>{'{ version, exportedAt, shelf, bookSources, replaceRules }'}</Text>
        }
      />

      <Card title="导出" className="mb-4">
        <Paragraph>从服务器下载当前全部数据的 JSON 快照。</Paragraph>
        <Button
          type="primary"
          icon={<CloudDownloadOutlined />}
          loading={exporting}
          onClick={handleExport}
        >
          导出同步包
        </Button>
      </Card>

      <Card title="导入">
        <Paragraph>上传此前导出的 JSON；书架条目会合并追加，书源会批量导入并热加载。</Paragraph>
        <Space>
          <Upload accept=".json" showUploadList={false} beforeUpload={handleImport}>
            <Button icon={<CloudUploadOutlined />} loading={importing}>
              选择文件导入
            </Button>
          </Upload>
        </Space>
      </Card>

      <Divider />

      <Card size="small" title={<><SyncOutlined /> 后续计划</>}>
        <Paragraph type="secondary" className="mb-0">
          WebDAV 自动同步（sidecar）尚未接入；当前可通过导出/上传 JSON 手动同步。
        </Paragraph>
      </Card>

      <Divider />

      <Card title={<><LockOutlined /> 安全设置</>} className="mb-4">
        <Paragraph type="secondary">
          {isPasswordSet
            ? '已设置访问密码。每次打开网站或关闭标签页后需要输入密码才能查看内容。'
            : '设置访问密码后，每次打开网站需要输入密码才能查看内容。密码仅保存在本地浏览器中。'}
        </Paragraph>
        <Space>
          {isPasswordSet ? (
            <>
              <Button icon={<LockOutlined />} onClick={() => setPwdModalOpen(true)}>
                修改密码
              </Button>
              <Button danger icon={<DeleteOutlined />} onClick={() => {
                Modal.confirm({
                  title: '关闭密码锁',
                  content: '确定要关闭密码保护吗？之后打开网站将不再需要密码。',
                  onOk: () => {
                    clearPassword()
                    message.success('已关闭密码保护')
                  },
                })
              }}>
                关闭密码
              </Button>
            </>
          ) : (
            <Button type="primary" icon={<LockOutlined />} onClick={() => setPwdModalOpen(true)}>
              设置密码
            </Button>
          )}
        </Space>
      </Card>

      <Modal
        title={isPasswordSet ? '修改密码' : '设置密码'}
        open={pwdModalOpen}
        onOk={async () => {
          if (!newPassword || newPassword.length < 4) {
            message.error('密码至少 4 个字符')
            return
          }
          if (newPassword !== confirmPwd) {
            message.error('两次输入的密码不一致')
            return
          }
          await setPassword(newPassword)
          message.success(isPasswordSet ? '密码已修改' : '密码已设置')
          setPwdModalOpen(false)
          setNewPassword('')
          setConfirmPwd('')
        }}
        onCancel={() => {
          setPwdModalOpen(false)
          setNewPassword('')
          setConfirmPwd('')
        }}
        okText="确认"
        cancelText="取消"
      >
        <div className="space-y-3 pt-2">
          <Input.Password
            placeholder="新密码（至少4位）"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
          />
          <Input.Password
            placeholder="确认新密码"
            value={confirmPwd}
            onChange={(e) => setConfirmPwd(e.target.value)}
          />
        </div>
      </Modal>
    </div>
  )
}
