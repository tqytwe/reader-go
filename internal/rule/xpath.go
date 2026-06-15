package rule

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
)

// XPathParser parses and executes XPath queries on HTML/XML documents.
// It supports XPath 1.0 syntax via antchfx/xmlquery and handles
// chain operators (&&, ||, %%) for compound rule expressions.
type XPathParser struct{}

// NewXPathParser creates a new XPath parser instance.
func NewXPathParser() *XPathParser {
	return &XPathParser{}
}

// ParseXPath is the main entry point. It accepts either an XPath query string
// or a compound rule expression with &&, ||, %% operators.
//
// Parameters:
//   - query: XPath expression or compound rule (e.g., "//title" or "//h1 && //h2")
//   - doc:   Parsed xmlquery.Document (can be created from HTML string via ParseHTML)
//
// Returns all matched string values as []string.
func (p *XPathParser) ParseXPath(query string, doc *xmlquery.Node) ([]string, error) {
	if query == "" || doc == nil {
		return nil, nil
	}

	// Step 1: Analyze chain operators (&&, ||, %%)
	analyzer := NewRuleAnalyzer()
	result := analyzer.Analyze(query)
	if result.Error != nil {
		return nil, fmt.Errorf("rule analysis failed: %w", result.Error)
	}

	if len(result.Segments) == 0 {
		return nil, nil
	}

	// Step 2: Execute each XPath expression and combine results
	var results []string
	var opIdx int

	for i, seg := range result.Segments {
		expr := strings.TrimSpace(seg.Selector)
		if expr == "" {
			continue
		}

		matches, err := p.execXPath(expr, doc)
		if err != nil {
			return nil, fmt.Errorf("xpath query '%s' failed: %w", expr, err)
		}

		if i == 0 {
			// First expression, just collect results
			results = append(results, matches...)
		} else if opIdx < len(result.Operators) {
			op := ParseOperatorType(result.Operators[opIdx])
			results = p.combineResults(results, matches, op)
			opIdx++
		} else {
			results = append(results, matches...)
		}
	}

	return results, nil
}

// ParseHTML parses an HTML string into an xmlquery.Document.
func (p *XPathParser) ParseHTML(htmlStr string) (*xmlquery.Node, error) {
	doc, err := xmlquery.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil, fmt.Errorf("parse html failed: %w", err)
	}
	return doc, nil
}

// execXPath executes a single XPath expression and returns matched text values.
func (p *XPathParser) execXPath(expr string, doc *xmlquery.Node) ([]string, error) {
	xpathExpr, err := xpath.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("xpath compile '%s' error: %w", expr, err)
	}

	nodes := xmlquery.QuerySelectorAll(doc, xpathExpr)
	if len(nodes) == 0 {
		node := xmlquery.QuerySelector(doc, xpathExpr)
		if node != nil {
			return []string{p.extractText(node)}, nil
		}
		return nil, nil
	}

	var results []string
	for _, node := range nodes {
		results = append(results, p.extractText(node))
	}
	return results, nil
}

// extractText extracts text content from an XML/HTML node.
func (p *XPathParser) extractText(node *xmlquery.Node) string {
	if node == nil {
		return ""
	}

	// If it's a text node, return its data
	if node.Type == xmlquery.TextNode {
		return strings.TrimSpace(node.Data)
	}

	// For element nodes, collect all descendant text
	var sb strings.Builder
	p.collectText(node, &sb)
	return strings.TrimSpace(sb.String())
}

// collectText recursively collects text from all descendant nodes.
func (p *XPathParser) collectText(node *xmlquery.Node, sb *strings.Builder) {
	if node == nil {
		return
	}

	if node.Type == xmlquery.TextNode {
		sb.WriteString(node.Data)
	} else if node.Type == xmlquery.ElementNode {
		// Add space between block-level elements
		if node.Data != "" && sb.Len() > 0 {
			lastRune := rune(0)
			if sb.Len() > 0 {
				lastRune = rune(sb.String()[sb.Len()-1])
			}
			if lastRune != ' ' && lastRune != '\n' {
				sb.WriteByte(' ')
			}
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		p.collectText(child, sb)
	}
}

// combineResults combines two result sets based on the operator.
func (p *XPathParser) combineResults(left, right []string, op OperatorType) []string {
	switch op {
	case OpAnd:
		// && : intersection — keep only values present in both
		return p.intersection(left, right)
	case OpOr:
		// || : union — combine all unique values
		return p.union(left, right)
	case OpCross:
		// %% : text content match — filter left by right's text content
		return p.modMatch(left, right)
	default:
		return right
	}
}

// intersection returns elements present in both slices (deduplicated).
func (p *XPathParser) intersection(left, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}

	rightSet := make(map[string]struct{}, len(right))
	for _, v := range right {
		rightSet[v] = struct{}{}
	}

	var result []string
	seen := make(map[string]struct{})
	for _, v := range left {
		if _, ok := rightSet[v]; ok && v != "" {
			if _, already := seen[v]; !already {
				seen[v] = struct{}{}
				result = append(result, v)
			}
		}
	}
	return result
}

// union returns all unique elements from both slices.
func (p *XPathParser) union(left, right []string) []string {
	seen := make(map[string]struct{}, len(left)+len(right))
	var result []string

	for _, v := range left {
		if v != "" {
			if _, ok := seen[v]; !ok {
				seen[v] = struct{}{}
				result = append(result, v)
			}
		}
	}
	for _, v := range right {
		if v != "" {
			if _, ok := seen[v]; !ok {
				seen[v] = struct{}{}
				result = append(result, v)
			}
		}
	}
	return result
}

// modMatch filters left results by checking if right's text content appears in each left item.
// Used for "%%" operator in book source rules.
func (p *XPathParser) modMatch(left, right []string) []string {
	if len(right) == 0 {
		return nil
	}

	// Concatenate all right results as filter patterns
	var filter strings.Builder
	for _, v := range right {
		if v != "" {
			filter.WriteString(v)
			filter.WriteByte(' ')
		}
	}
	filterStr := strings.TrimSpace(filter.String())
	if filterStr == "" {
		return nil
	}

	var result []string
	for _, v := range left {
		if v != "" && strings.Contains(v, filterStr) {
			result = append(result, v)
		}
	}
	return result
}

// --- Convenience functions for direct use ---

// QueryXPath executes an XPath query on an HTML string directly.
func QueryXPath(htmlStr, xpathExpr string) ([]string, error) {
	parser := NewXPathParser()
	doc, err := parser.ParseHTML(htmlStr)
	if err != nil {
		return nil, err
	}
	return parser.ParseXPath(xpathExpr, doc)
}

// QueryOnDoc executes an XPath query on an existing xmlquery.Document.
func QueryOnDoc(doc *xmlquery.Node, xpathExpr string) ([]string, error) {
	parser := NewXPathParser()
	return parser.ParseXPath(xpathExpr, doc)
}

// QueryXPathSelector returns the first match of an XPath expression.
func QueryXPathSelector(htmlStr, xpathExpr string) (string, error) {
	results, err := QueryXPath(htmlStr, xpathExpr)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", nil
	}
	return results[0], nil
}
