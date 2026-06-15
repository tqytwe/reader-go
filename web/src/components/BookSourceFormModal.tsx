import { Form, Input, Select, Switch, Tabs, InputNumber } from 'antd'
import Editor from '@monaco-editor/react'
import type { BookSourceDTO, ParseMode } from '@/types/booksource'

const { TextArea } = Input

const MODE_OPTIONS: { label: string; value: ParseMode }[] = [
  { label: '默认', value: 'default' },
  { label: 'XPath', value: 'xpath' },
  { label: 'JSONPath', value: 'jsonpath' },
  { label: 'CSS', value: 'css' },
  { label: '正则', value: 'regex' },
  { label: 'JS', value: 'js' },
]

function RuleEditor({ value, onChange }: { value?: string; onChange?: (v: string) => void }) {
  return (
    <Editor
      height="120px"
      defaultLanguage="plaintext"
      value={value ?? ''}
      onChange={(v) => onChange?.(v ?? '')}
      options={{
        minimap: { enabled: false },
        wordWrap: 'on',
        fontSize: 12,
        scrollBeyondLastLine: false,
      }}
    />
  )
}

export default function BookSourceFormModal({ form }: { form: ReturnType<typeof Form.useForm<BookSourceDTO>>[0] }) {
  return (
    <Form form={form} layout="vertical" autoComplete="off">
      <Tabs
        items={[
          {
            key: 'basic',
            label: '简易',
            children: (
              <>
                <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入书源名称' }]}>
                  <Input placeholder="例如：起点中文网" />
                </Form.Item>
                <Form.Item
                  name="baseUrl"
                  label="基础 URL"
                  rules={[{ required: true, message: '请输入基础 URL' }]}
                >
                  <Input placeholder="https://example.com" />
                </Form.Item>
                <Form.Item name="searchUrl" label="搜索 URL" rules={[{ required: true }]}>
                  <Input placeholder="支持 {{key}} / @js:" />
                </Form.Item>
                <Form.Item name="searchRule" label="搜索规则" rules={[{ required: true }]}>
                  <TextArea rows={3} placeholder="Legado 规则或 CSS/XPath" />
                </Form.Item>
                <Form.Item name="searchMode" label="搜索模式">
                  <Select options={MODE_OPTIONS} />
                </Form.Item>
                <Form.Item name="group" label="分组">
                  <Input placeholder="免费 / 付费" />
                </Form.Item>
                <Form.Item name="enabled" label="启用" valuePropName="checked">
                  <Switch checkedChildren="启用" unCheckedChildren="禁用" />
                </Form.Item>
              </>
            ),
          },
          {
            key: 'advanced',
            label: '高级',
            children: (
              <>
                <div className="grid grid-cols-2 gap-3">
                  <Form.Item name="bookInfoUrl" label="详情 URL">
                    <Input />
                  </Form.Item>
                  <Form.Item name="tocUrl" label="目录 URL">
                    <Input />
                  </Form.Item>
                  <Form.Item name="contentUrl" label="正文 URL">
                    <Input />
                  </Form.Item>
                  <Form.Item name="loginUrl" label="登录 URL">
                    <Input placeholder="需登录的书源" />
                  </Form.Item>
                </div>

                <Form.Item name="bookInfoRule" label="详情规则 (Monaco)">
                  <RuleEditor />
                </Form.Item>
                <Form.Item name="tocRule" label="目录规则 (Monaco)">
                  <RuleEditor />
                </Form.Item>
                <Form.Item name="contentRule" label="正文规则 (Monaco)">
                  <RuleEditor />
                </Form.Item>

                <div className="grid grid-cols-2 gap-3">
                  <Form.Item name="bookInfoMode" label="详情模式">
                    <Select options={MODE_OPTIONS} />
                  </Form.Item>
                  <Form.Item name="tocMode" label="目录模式">
                    <Select options={MODE_OPTIONS} />
                  </Form.Item>
                  <Form.Item name="contentMode" label="正文模式">
                    <Select options={MODE_OPTIONS} />
                  </Form.Item>
                  <Form.Item name="exploreMode" label="发现模式">
                    <Select options={MODE_OPTIONS} />
                  </Form.Item>
                </div>

                <Form.Item name="exploreUrl" label="发现 URL">
                  <TextArea rows={2} placeholder="JSON 数组或单 URL" />
                </Form.Item>
                <Form.Item name="exploreRule" label="发现规则">
                  <TextArea rows={2} />
                </Form.Item>

                <Form.Item name="headers" label="请求头 (JSON 字符串)">
                  <TextArea rows={3} placeholder='{"User-Agent":"..."}' />
                </Form.Item>
                <Form.Item name="cookie" label="Cookie">
                  <TextArea rows={2} />
                </Form.Item>
                <div className="grid grid-cols-2 gap-3">
                  <Form.Item name="userAgent" label="User-Agent">
                    <Input />
                  </Form.Item>
                  <Form.Item name="timeout" label="超时 (秒)">
                    <InputNumber min={5} max={120} className="w-full" />
                  </Form.Item>
                </div>
              </>
            ),
          },
        ]}
      />
    </Form>
  )
}
