import { createBrowserRouter } from 'react-router-dom'
import Layout from './components/Layout'
import Home from './pages/Home'
import Bookshelf from './pages/Bookshelf'
import Search from './pages/Search'
import BookSourceManage from './pages/BookSourceManage'
import BookSourceDebug from './pages/BookSourceDebug'
import Reader from './pages/Reader'
import Rss from './pages/Rss'
import Explore from './pages/Explore'
import ReplaceRules from './pages/ReplaceRules'
import LocalBooks from './pages/LocalBooks'
import SyncSettings from './pages/SyncSettings'

export const router = createBrowserRouter([
  { path: '/reader/:bookId', element: <Reader /> },
  {
    path: '/',
    element: <Layout />,
    children: [
      { index: true, element: <Home /> },
      { path: 'bookshelf', element: <Bookshelf /> },
      { path: 'search', element: <Search /> },
      { path: 'booksource', element: <BookSourceManage /> },
      { path: 'booksource/debug', element: <BookSourceDebug /> },
      { path: 'rss', element: <Rss /> },
      { path: 'explore', element: <Explore /> },
      { path: 'replaceRules', element: <ReplaceRules /> },
      { path: 'localBooks', element: <LocalBooks /> },
      { path: 'sync', element: <SyncSettings /> },
    ],
  },
])
