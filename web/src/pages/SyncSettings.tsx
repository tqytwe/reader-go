import { useState } from 'react'
import { Card, Button, Upload, message, Typography, Alert, Space, Divider } from 'antd'
import { CloudDownloadOutlined, CloudUploadOutlined, SyncOutlined } from '@ant-design/icons'
import { api } from '../api/client'
import type { SyncBundle } from '../types/booksource'
import { useShelfStore, type ShelfBook } from '../store/useStore'

const { Paragraph, Text } = Typography

export default function SyncSettings() {
  const [exporting, setExporting] = useState(false)
  const [importing, setImporting] = useState(false)

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
    </div>
  )
}
