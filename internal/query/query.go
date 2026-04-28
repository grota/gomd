package query

import mdparser "github.com/grota/gomd/internal/parser"

// Execute parses and evaluates a query string against a document.
func Execute(doc *mdparser.Document, queryStr string) ([]Value, error) {
	q, err := Parse(queryStr)
	if err != nil {
		return nil, err
	}
	engine := NewEngine(doc)
	return engine.Execute(q)
}
