import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Upload, Table, Button, message } from 'antd'
import { UploadOutlined, ReadOutlined } from '@ant-design/icons'
import { api, type LocalBook } from '../api/client'

export default function LocalBooks() {
  const navigate = useNavigate()
  const [books, setBooks] = useState<LocalBook[]>([])
  const [loading, setLoading] = useState(false)

  const load = async () => {
    setLoading(true)
    try {
      const data: unknown = await api.listLocalBooks()
      setBooks(Array.isArray(data) ? (data as LocalBook[]) : [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  const uploadProps = {
    showUploadList: false,
    beforeUpload: async (file: File) => {
      try {
        await api.uploadLocalBook(file)
        message.success('上传成功')
        load()
      } catch (e: unknown) {
        message.error(e instanceof Error ? e.message : '上传失败')
      }
      return false
    },
  }

  return (
    <div className="p-6">
      <div className="flex justify-between mb-4">
        <h1 className="text-2xl font-bold">本地书籍</h1>
        <Upload {...uploadProps}>
          <Button icon={<UploadOutlined />}>上传 TXT/EPUB/CBZ</Button>
        </Upload>
      </div>
      <Table
        loading={loading}
        rowKey="id"
        dataSource={books}
        columns={[
          { title: '书名', dataIndex: 'name' },
          { title: '作者', dataIndex: 'author' },
          { title: '格式', dataIndex: 'format' },
          {
            title: '操作',
            render: (_, r) => (
              <Button icon={<ReadOutlined />} size="small" onClick={() => navigate(`/reader/local-${r.id}`)}>
                阅读
              </Button>
            ),
          },
        ]}
      />
    </div>
  )
}
