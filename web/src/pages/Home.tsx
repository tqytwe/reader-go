import { Link } from 'react-router-dom'

export default function Home() {
  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="text-center">
        <h1 className="text-4xl font-bold mb-4">Reader Go</h1>
        <p className="text-gray-500">基于 Go + React 的在线阅读平台</p>
        <p className="text-gray-400 mt-2">复刻 hectorqin/reader 功能</p>

        {/* Quick Links */}
        <div className="mt-8 flex flex-wrap justify-center gap-4">
          <Link
            to="/rss"
            className="px-4 py-2 bg-orange-500 text-white rounded-lg hover:bg-orange-600 transition"
          >
            RSS 订阅管理
          </Link>
          <Link
            to="/search"
            className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition"
          >
            搜索书籍
          </Link>
          <Link
            to="/bookshelf"
            className="px-4 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 transition"
          >
            我的书架
          </Link>
        </div>
      </div>
    </div>
  )
}
