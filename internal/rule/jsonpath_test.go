package rule

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// Test data helpers
// ============================================================

func testBookJSON() string {
	return `{
		"book": {
			"title": "三体",
			"author": "刘慈欣",
			"isbn": "978-7-5366-9293-0",
			"publisher": "重庆出版社",
			"pages": 387,
			"rating": 9.0,
			"published": true,
			"chapters": [
				{"num": 1, "title": "科学边界"},
				{"num": 2, "title": "射手与农场主"},
				{"num": 3, "title": "物理学不存在了"}
			],
			"tags": ["科幻", "硬科幻", "中国科幻"],
			"series": {
				"name": "三体系列",
				"volume": 1,
				"total": 3
			}
		}
	}`
}

func testBooksArrayJSON() string {
	return `{
		"books": [
			{"title": "三体", "author": "刘慈欣", "price": 39.5},
			{"title": "球状闪电", "author": "刘慈欣", "price": 35.0},
			{"title": "时间移民", "author": "刘慈欣", "price": 32.0},
			{"title": "三体II", "author": "刘慈欣", "price": 42.0}
		],
		"meta": {
			"total": 4,
			"source": "图书馆"
		}
	}`
}

func testComplexJSON() string {
	return `{
		"store": {
			"book": [
				{
					"category": "reference",
					"title": "Sayings of the Century",
					"author": "Ned Stin",
					"price": 8.95
				},
				{
					"category": "fiction",
					"title": "Sword of Honour",
					"author": "Evelyn Waugh",
					"price": 12.99
				},
				{
					"category": "fiction",
					"title": "Moby Dick",
					"author": "Herman Melville",
					"isbn": "0-553-21311-3",
					"price": 8.99
				},
				{
					"category": "fiction",
					"title": "The Lord of the Rings",
					"author": "J.R.R. Tolkien",
					"price": 22.99
				}
			],
			"bicycle": {
				"color": "red",
				"price": 19.95
			}
		},
		"expensive": 10
	}`
}

// ============================================================
// Basic JSONPath queries
// ============================================================

func TestJSONPathParser_BasicScalar(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testBookJSON()

	// $.book.title
	results, err := parser.ParseJSONPath("$.book.title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "三体", results[0])

	// $.book.author
	results, err = parser.ParseJSONPath("$.book.author", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "刘慈欣", results[0])

	// $.book.pages (number)
	results, err = parser.ParseJSONPath("$.book.pages", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "387", results[0])

	// $.book.rating (float)
	results, err = parser.ParseJSONPath("$.book.rating", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "9", results[0])

	// $.book.published (boolean)
	results, err = parser.ParseJSONPath("$.book.published", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "true", results[0])
}

func TestJSONPathParser_NestedPath(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testBookJSON()

	// $.book.series.name
	results, err := parser.ParseJSONPath("$.book.series.name", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "三体系列", results[0])

	// $.book.series.volume
	results, err = parser.ParseJSONPath("$.book.series.volume", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "1", results[0])
}

func TestJSONPathParser_WildcardArray(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testBookJSON()

	// $.book.chapters[*].title — 获取所有章节标题
	results, err := parser.ParseJSONPath("$.book.chapters[*].title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, "科学边界", results[0])
	assert.Equal(t, "射手与农场主", results[1])
	assert.Equal(t, "物理学不存在了", results[2])

	// $.book.chapters[*].num — 获取所有章节编号
	results, err = parser.ParseJSONPath("$.book.chapters[*].num", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, "1", results[0])
	assert.Equal(t, "2", results[1])
	assert.Equal(t, "3", results[2])
}

func TestJSONPathParser_WildcardArrayBooks(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testBooksArrayJSON()

	// $.books[*].title — 获取所有书名
	results, err := parser.ParseJSONPath("$.books[*].title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 4)
	assert.Equal(t, "三体", results[0])
	assert.Equal(t, "球状闪电", results[1])
	assert.Equal(t, "时间移民", results[2])
	assert.Equal(t, "三体II", results[3])

	// $.books[*].author — 获取所有作者
	results, err = parser.ParseJSONPath("$.books[*].author", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 4)
	for _, a := range results {
		assert.Equal(t, "刘慈欣", a)
	}

	// $.books[*].price — 获取所有价格
	results, err = parser.ParseJSONPath("$.books[*].price", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 4)
	assert.Equal(t, "39.5", results[0])
	assert.Equal(t, "35", results[1])
}

func TestJSONPathParser_ArrayIndex(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testBookJSON()

	// $.book.chapters[0].title — 第一个章节
	results, err := parser.ParseJSONPath("$.book.chapters[0].title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "科学边界", results[0])

	// $.book.chapters[2].title — 第三个章节
	results, err = parser.ParseJSONPath("$.book.chapters[2].title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "物理学不存在了", results[0])
}

func TestJSONPathParser_ArraySlice(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testBookJSON()

	// $.book.chapters[0:2].title — 前两个章节
	results, err := parser.ParseJSONPath("$.book.chapters[0:2].title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "科学边界", results[0])
	assert.Equal(t, "射手与农场主", results[1])

	// $.book.chapters[1:].title — 从第二个开始
	results, err = parser.ParseJSONPath("$.book.chapters[1:].title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "射手与农场主", results[0])
	assert.Equal(t, "物理学不存在了", results[1])
}

func TestJSONPathParser_TagArray(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testBookJSON()

	// $.book.tags[*] — 所有标签
	results, err := parser.ParseJSONPath("$.book.tags[*]", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, "科幻", results[0])
	assert.Equal(t, "硬科幻", results[1])
	assert.Equal(t, "中国科幻", results[2])
}

// ============================================================
// Store JSON (classic JSONPath test)
// ============================================================

func TestJSONPathParser_StoreAllBooks(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testComplexJSON()

	// $.store.book[*].title — 所有书名
	results, err := parser.ParseJSONPath("$.store.book[*].title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 4)
	assert.Equal(t, "Sayings of the Century", results[0])
	assert.Equal(t, "Sword of Honour", results[1])
	assert.Equal(t, "Moby Dick", results[2])
	assert.Equal(t, "The Lord of the Rings", results[3])

	// $.store.book[*].author — 所有作者
	results, err = parser.ParseJSONPath("$.store.book[*].author", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 4)
	assert.Equal(t, "Ned Stin", results[0])
	assert.Equal(t, "Evelyn Waugh", results[1])

	// $.store.book[*].price — 所有价格
	results, err = parser.ParseJSONPath("$.store.book[*].price", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 4)
	assert.Equal(t, "8.95", results[0])
	assert.Equal(t, "12.99", results[1])

	// $.store.bicycle.color
	results, err = parser.ParseJSONPath("$.store.bicycle.color", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "red", results[0])
}

// ============================================================
// Chain operators: && (intersection)
// ============================================================

func TestJSONPathParser_AndChain(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testBooksArrayJSON()

	// $.books[*].title && $.books[*].author
	// 交集：同时出现在 title 列表和 author 列表中的值
	// 这里没有交集，返回空
	results, err := parser.ParseJSONPath("$.books[*].title && $.books[*].author", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 0)

	// 另一个场景：两个查询有交集
	// 用同一本书的不同字段做测试
	jsonStr2 := `{
		"data": {
			"a": ["x", "y", "z"],
			"b": ["y", "z", "w"]
		}
	}`
	// $.data.a[*] && $.data.b[*] → ["y", "z"]
	results, err = parser.ParseJSONPath("$.data.a[*] && $.data.b[*]", jsonStr2)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Contains(t, results, "y")
	assert.Contains(t, results, "z")
}

// ============================================================
// Chain operators: || (or / short-circuit)
// ============================================================

func TestJSONPathParser_OrChain(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testBooksArrayJSON()

	// $.books[0].title || $.books[1].title
	// 短路求值：第一个非空即返回
	results, err := parser.ParseJSONPath("$.books[0].title || $.books[1].title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "三体", results[0])

	// 第一个为空时，返回第二个
	// $.nonexistent || $.books[0].title
	results, err = parser.ParseJSONPath("$.nonexistent || $.books[0].title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "三体", results[0])

	// 两个都有值，返回第一个
	results, err = parser.ParseJSONPath("$.books[0].title || $.books[1].title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "三体", results[0])
}

// ============================================================
// Mixed operators
// ============================================================

func TestJSONPathParser_MixedOperators(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testBooksArrayJSON()

	// $.books[0].title && $.books[1].title || $.books[2].title
	// 先算 && 再算 ||
	results, err := parser.ParseJSONPath("$.books[0].title && $.books[1].title || $.books[2].title", jsonStr)
	require.NoError(t, err)
	// && 交集为空，然后 || 返回第三个
	assert.Len(t, results, 1)
	assert.Equal(t, "时间移民", results[0])
}

// ============================================================
// Convenience functions
// ============================================================

func TestQuery(t *testing.T) {
	jsonStr := `{"book": {"title": "三体"}}`

	results, err := Query(jsonStr, "$.book.title")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "三体", results[0])
}

func TestQueryFirst(t *testing.T) {
	jsonStr := `{"book": {"title": "三体", "author": "刘慈欣"}}`

	result, err := QueryFirst(jsonStr, "$.book.title")
	require.NoError(t, err)
	assert.Equal(t, "三体", result)

	// 不存在的查询
	result, err = QueryFirst(jsonStr, "$.book.nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestQueryOnResult(t *testing.T) {
	parser := NewJSONPathParser()

	// 先获取 book 对象，再在其上查询
	base, err := parser.ParseJSON(testBookJSON())
	require.NoError(t, err)
	assert.True(t, base.Exists())

	results, err := QueryOnResult(base, "book.title")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "三体", results[0])

	results, err = QueryOnResult(base, "book.chapters[*].title")
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

// ============================================================
// Edge cases
// ============================================================

func TestJSONPathParser_EmptyInput(t *testing.T) {
	parser := NewJSONPathParser()

	results, err := parser.ParseJSONPath("", "{}")
	require.NoError(t, err)
	assert.Nil(t, results)

	results, err = parser.ParseJSONPath("$.book.title", "")
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestJSONPathParser_InvalidJSON(t *testing.T) {
	parser := NewJSONPathParser()

	_, err := parser.ParseJSONPath("$.book.title", "not json")
	assert.Error(t, err)
}

func TestJSONPathParser_NoMatch(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testBookJSON()

	results, err := parser.ParseJSONPath("$.book.nonexistent", jsonStr)
	require.NoError(t, err)
	assert.Nil(t, results)

	results, err = parser.ParseJSONPath("$.nonexistent.path", jsonStr)
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestJSONPathParser_PathWithoutDollar(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := `{"book": {"title": "三体"}}`

	// 不带 $ 前缀的路径应该自动补全
	results, err := parser.ParseJSONPath("book.title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "三体", results[0])
}

func TestJSONPathParser_RootObject(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := `{"title": "三体", "author": "刘慈欣"}`

	// $. 获取根对象的所有字段值
	results, err := parser.ParseJSONPath("$", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Contains(t, results, "三体")
	assert.Contains(t, results, "刘慈欣")
}

func TestJSONPathParser_ArrayRoot(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := `["三体", "球状闪电", "时间移民"]`

	// $[*] 获取数组所有元素
	results, err := parser.ParseJSONPath("$[*]", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, "三体", results[0])
	assert.Equal(t, "球状闪电", results[1])
	assert.Equal(t, "时间移民", results[2])
}

func TestJSONPathParser_Whitespace(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testBookJSON()

	// 带空格的查询
	results, err := parser.ParseJSONPath("  $.book.title  ", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "三体", results[0])
}

func TestJSONPathParser_EmptyArray(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := `{"books": []}`

	results, err := parser.ParseJSONPath("$.books[*].title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestJSONPathParser_NullValue(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := `{"book": {"title": null, "author": "刘慈欣"}}`

	// null 值应该被正确处理
	results, err := parser.ParseJSONPath("$.book.title", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "null", results[0])

	results, err = parser.ParseJSONPath("$.book.author", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "刘慈欣", results[0])
}

func TestJSONPathParser_EmbeddedObject(t *testing.T) {
	parser := NewJSONPathParser()
	jsonStr := testBookJSON()

	// 获取整个章节对象（嵌套对象）
	results, err := parser.ParseJSONPath("$.book.chapters[0]", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results[0], `"num":1`)
	assert.Contains(t, results[0], `"title":"科学边界"`)
}

// ============================================================
// Real-world book source scenarios
// ============================================================

func TestJSONPathParser_RealBookSource_Detail(t *testing.T) {
	jsonStr := `{
		"novel": {
			"name": "诡秘之主",
			"author": "爱潜水的乌贼",
			"status": "已完结",
			"wordCount": 4500000,
			"intro": "在蒸汽与机械的浪潮中，谁能触及非凡？",
			"tags": ["克苏鲁", "蒸汽朋克", "西幻"],
			"cover": "https://example.com/cover.jpg",
			"chapters": {
				"total": 1394,
				"list": [
					{"id": 1, "name": "小丑", "url": "/chap/1"},
					{"id": 2, "name": "魔药", "url": "/chap/2"},
					{"id": 3, "name": "占卜", "url": "/chap/3"}
				]
			}
		}
	}`

	parser := NewJSONPathParser()

	// 书名
	title, err := parser.ParseJSONPath("$.novel.name", jsonStr)
	require.NoError(t, err)
	assert.Equal(t, "诡秘之主", title[0])

	// 作者
	author, err := parser.ParseJSONPath("$.novel.author", jsonStr)
	require.NoError(t, err)
	assert.Equal(t, "爱潜水的乌贼", author[0])

	// 状态
	status, err := parser.ParseJSONPath("$.novel.status", jsonStr)
	require.NoError(t, err)
	assert.Equal(t, "已完结", status[0])

	// 字数
	words, err := parser.ParseJSONPath("$.novel.wordCount", jsonStr)
	require.NoError(t, err)
	assert.Equal(t, "4500000", words[0])

	// 简介
	intro, err := parser.ParseJSONPath("$.novel.intro", jsonStr)
	require.NoError(t, err)
	assert.Contains(t, intro[0], "蒸汽与机械")

	// 标签
	tags, err := parser.ParseJSONPath("$.novel.tags[*]", jsonStr)
	require.NoError(t, err)
	assert.Len(t, tags, 3)

	// 章节总数
	chapTotal, err := parser.ParseJSONPath("$.novel.chapters.total", jsonStr)
	require.NoError(t, err)
	assert.Equal(t, "1394", chapTotal[0])

	// 所有章节名
	chapNames, err := parser.ParseJSONPath("$.novel.chapters.list[*].name", jsonStr)
	require.NoError(t, err)
	assert.Len(t, chapNames, 3)
	assert.Equal(t, "小丑", chapNames[0])
	assert.Equal(t, "魔药", chapNames[1])
}

func TestJSONPathParser_RealBookSource_Search(t *testing.T) {
	jsonStr := `{
		"result": {
			"code": 0,
			"data": {
				"list": [
					{
						"bookId": "100",
						"bookName": "斗破苍穹",
						"authorName": "天蚕土豆",
						"latestChapter": "第1632章 大结局",
						"bookStatus": "已完结",
						"coverUrl": "https://img.com/100.jpg"
					},
					{
						"bookId": "101",
						"bookName": "斗罗大陆",
						"authorName": "唐家三少",
						"latestChapter": "第371章 海神缘",
					 "bookStatus": "已完结",
						"coverUrl": "https://img.com/101.jpg"
					},
					{
						"bookId": "102",
						"bookName": "凡人修仙传",
						"authorName": "忘语",
						"latestChapter": "第245章 结婴",
						"bookStatus": "已完结",
						"coverUrl": "https://img.com/102.jpg"
					}
				]
			}
		}
	}`

	parser := NewJSONPathParser()

	// 所有书名
	names, err := parser.ParseJSONPath("$.result.data.list[*].bookName", jsonStr)
	require.NoError(t, err)
	assert.Len(t, names, 3)
	assert.Equal(t, "斗破苍穹", names[0])
	assert.Equal(t, "斗罗大陆", names[1])
	assert.Equal(t, "凡人修仙传", names[2])

	// 所有作者
	authors, err := parser.ParseJSONPath("$.result.data.list[*].authorName", jsonStr)
	require.NoError(t, err)
	assert.Len(t, authors, 3)

	// 所有最新章节
	chapters, err := parser.ParseJSONPath("$.result.data.list[*].latestChapter", jsonStr)
	require.NoError(t, err)
	assert.Len(t, chapters, 3)
	assert.Contains(t, chapters[0], "大结局")

	// 所有书ID
	ids, err := parser.ParseJSONPath("$.result.data.list[*].bookId", jsonStr)
	require.NoError(t, err)
	assert.Len(t, ids, 3)
	assert.Equal(t, "100", ids[0])
}

func TestJSONPathParser_RealBookSource_ChainOr(t *testing.T) {
	// 模拟备用字段：优先取 title，没有则取 name
	jsonStr := `{
		"book": {
			"name": "备用书名",
			"info": {
				"title": "正式书名"
			}
		}
	}`

	parser := NewJSONPathParser()

	// $.book.info.title || $.book.name
	// 先尝试正式书名，没有则用备用
	results, err := parser.ParseJSONPath("$.book.info.title || $.book.name", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "正式书名", results[0])

	// 当正式书名不存在时
	jsonStr2 := `{"book": {"name": "备用书名"}}`
	results, err = parser.ParseJSONPath("$.book.info.title || $.book.name", jsonStr2)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "备用书名", results[0])
}

func TestJSONPathParser_RealBookSource_ChainAnd(t *testing.T) {
	// 取书名和作者的交集（实际场景：筛选同时出现在两个列表中的值）
	jsonStr := `{
		"source1": {"books": ["三体", "球状闪电", "时间移民"]},
		"source2": {"books": ["球状闪电", "三体II", "三体III"]}
	}`

	parser := NewJSONPathParser()

	// $.source1.books[*] && $.source2.books[*]
	// 交集：球状闪电
	results, err := parser.ParseJSONPath("$.source1.books[*] && $.source2.books[*]", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "球状闪电", results[0])
}

// ============================================================
// gjson Result integration
// ============================================================

func TestJSONPathParser_ParseJSON(t *testing.T) {
	parser := NewJSONPathParser()

	result, err := parser.ParseJSON(`{"key": "value"}`)
	require.NoError(t, err)
	assert.True(t, result.Exists())
	assert.Equal(t, "value", result.Get("key").String())

	_, err = parser.ParseJSON("invalid")
	assert.Error(t, err)
}

// ============================================================
// Complex nested structures
// ============================================================

func TestJSONPathParser_DeepNesting(t *testing.T) {
	jsonStr := `{
		"level1": {
			"level2": {
				"level3": {
					"level4": {
						"value": "deep value"
					}
				}
			}
		}
	}`

	parser := NewJSONPathParser()

	results, err := parser.ParseJSONPath("$.level1.level2.level3.level4.value", jsonStr)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "deep value", results[0])
}

func TestJSONPathParser_MultipleWildcards(t *testing.T) {
	jsonStr := `{
		"library": {
			"floors": [
				{
					"floor": 1,
					"sections": [
						{"name": "A", "books": ["book1", "book2"]},
						{"name": "B", "books": ["book3", "book4"]}
					]
				},
				{
					"floor": 2,
					"sections": [
						{"name": "C", "books": ["book5", "book6"]}
					]
				}
			]
		}
	}`

	parser := NewJSONPathParser()

	// 获取所有section名称
	sections, err := parser.ParseJSONPath("$.library.floors[*].sections[*].name", jsonStr)
	require.NoError(t, err)
	assert.Len(t, sections, 3)
	assert.Contains(t, sections, "A")
	assert.Contains(t, sections, "B")
	assert.Contains(t, sections, "C")

	// 获取所有书名
	books, err := parser.ParseJSONPath("$.library.floors[*].sections[*].books[*]", jsonStr)
	require.NoError(t, err)
	assert.Len(t, books, 6)
	assert.Equal(t, "book1", books[0])
	assert.Equal(t, "book6", books[5])
}

// ============================================================
// String values with special characters
// ============================================================

func TestJSONPathParser_SpecialCharacters(t *testing.T) {
	jsonStr := `{
		"book": {
			"title": "三体：地球往事",
			"subtitle": "The Three-Body Problem (English Edition)",
			"description": "A story about \"aliens\" and \"physics\"."
		}
	}`

	parser := NewJSONPathParser()

	results, err := parser.ParseJSONPath("$.book.title", jsonStr)
	require.NoError(t, err)
	assert.Equal(t, "三体：地球往事", results[0])

	results, err = parser.ParseJSONPath("$.book.subtitle", jsonStr)
	require.NoError(t, err)
	assert.Contains(t, results[0], "Three-Body")

	results, err = parser.ParseJSONPath("$.book.description", jsonStr)
	require.NoError(t, err)
	assert.Contains(t, results[0], "aliens")
}
