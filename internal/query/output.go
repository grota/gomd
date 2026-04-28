package query

import (
	"encoding/json"
	"fmt"
	"strings"
)

// OutputFormat controls how query results are displayed.
type OutputFormat int

const (
	OutPlain OutputFormat = iota
	OutJSON
	OutJSONPretty
	OutJSONL
	OutMarkdown
	OutTree
)

// ParseOutputFormat parses an output format string.
func ParseOutputFormat(s string) (OutputFormat, error) {
	switch strings.ToLower(s) {
	case "plain", "":
		return OutPlain, nil
	case "json":
		return OutJSON, nil
	case "json-pretty", "jsonp", "json_pretty":
		return OutJSONPretty, nil
	case "jsonl":
		return OutJSONL, nil
	case "md", "markdown":
		return OutMarkdown, nil
	case "tree":
		return OutTree, nil
	}
	return 0, fmt.Errorf("unknown output format %q", s)
}

// FormatOutput formats query results for display.
func FormatOutput(results []Value, format OutputFormat) string {
	if len(results) == 0 {
		return ""
	}

	switch format {
	case OutJSON:
		data := valuesToJSON(results)
		b, _ := json.Marshal(data)
		return string(b)
	case OutJSONPretty:
		data := valuesToJSON(results)
		b, _ := json.MarshalIndent(data, "", "  ")
		return string(b)
	case OutJSONL:
		var lines []string
		for _, v := range results {
			data := valueToJSON(v)
			b, _ := json.Marshal(data)
			lines = append(lines, string(b))
		}
		return strings.Join(lines, "\n")
	case OutMarkdown:
		var parts []string
		for _, v := range results {
			parts = append(parts, valueToMarkdown(v))
		}
		return strings.Join(parts, "\n\n")
	case OutTree:
		var parts []string
		for _, v := range results {
			parts = append(parts, valueToTree(v, "", true))
		}
		return strings.Join(parts, "")
	default: // OutPlain
		var parts []string
		for _, v := range results {
			parts = append(parts, valuePlain(v))
		}
		return strings.Join(parts, "\n")
	}
}

func valuePlain(v Value) string {
	switch v.Kind {
	case ValHeading:
		if v.Heading != nil {
			return fmt.Sprintf("%s %s", strings.Repeat("#", int(v.Heading.Level)), v.Heading.Text)
		}
	case ValCode:
		if v.Code != nil {
			lang := v.Code.Language
			if lang != "" {
				return fmt.Sprintf("```%s\n%s\n```", lang, v.Code.Content)
			}
			return fmt.Sprintf("```\n%s\n```", v.Code.Content)
		}
	case ValLink:
		if v.Link != nil {
			return fmt.Sprintf("[%s](%s)", v.Link.Text, v.Link.URL)
		}
	case ValImage:
		if v.Image != nil {
			return fmt.Sprintf("![%s](%s)", v.Image.Alt, v.Image.Src)
		}
	case ValTable:
		if v.Table != nil {
			return renderTablePlain(v.Table)
		}
	case ValList:
		if v.List != nil {
			return renderListPlain(v.List)
		}
	case ValArray:
		var parts []string
		for _, el := range v.Array {
			parts = append(parts, valuePlain(el))
		}
		return strings.Join(parts, "\n")
	case ValObject:
		var parts []string
		for k, val := range v.Object {
			parts = append(parts, fmt.Sprintf("%s: %s", k, valuePlain(val)))
		}
		return strings.Join(parts, "\n")
	case ValDocument:
		return "[document]"
	}
	return v.ToText()
}

func valueToMarkdown(v Value) string {
	switch v.Kind {
	case ValHeading:
		if v.Heading != nil {
			return fmt.Sprintf("%s %s", strings.Repeat("#", int(v.Heading.Level)), v.Heading.Text)
		}
	case ValCode:
		if v.Code != nil {
			return fmt.Sprintf("```%s\n%s\n```", v.Code.Language, v.Code.Content)
		}
	case ValLink:
		if v.Link != nil {
			return fmt.Sprintf("[%s](%s)", v.Link.Text, v.Link.URL)
		}
	case ValArray:
		var parts []string
		for _, el := range v.Array {
			parts = append(parts, valueToMarkdown(el))
		}
		return strings.Join(parts, "\n\n")
	}
	return v.ToText()
}

func valueToTree(v Value, prefix string, isLast bool) string {
	var connector, continuation string
	if isLast {
		connector = "└─ "
		continuation = "    "
	} else {
		connector = "├─ "
		continuation = "│   "
	}

	switch v.Kind {
	case ValHeading:
		if v.Heading != nil {
			marker := strings.Repeat("#", int(v.Heading.Level))
			return fmt.Sprintf("%s%s%s %s\n", prefix, connector, marker, v.Heading.Text)
		}
	case ValArray:
		var sb strings.Builder
		for i, el := range v.Array {
			last := i == len(v.Array)-1
			sb.WriteString(valueToTree(el, prefix+continuation, last))
		}
		return sb.String()
	}

	return fmt.Sprintf("%s%s%s\n", prefix, connector, v.ToText())
}

func renderTablePlain(t *TableValue) string {
	if t == nil {
		return ""
	}
	var lines []string
	lines = append(lines, "| "+strings.Join(t.Headers, " | ")+" |")
	seps := make([]string, len(t.Headers))
	for i := range seps {
		seps[i] = "---"
	}
	lines = append(lines, "| "+strings.Join(seps, " | ")+" |")
	for _, row := range t.Rows {
		lines = append(lines, "| "+strings.Join(row, " | ")+" |")
	}
	return strings.Join(lines, "\n")
}

func renderListPlain(l *ListValue) string {
	if l == nil {
		return ""
	}
	var lines []string
	for i, item := range l.Items {
		var prefix string
		if l.Ordered {
			prefix = fmt.Sprintf("%d. ", i+1)
		} else {
			prefix = "- "
		}
		if item.Checked != nil {
			if *item.Checked {
				prefix += "[x] "
			} else {
				prefix += "[ ] "
			}
		}
		lines = append(lines, prefix+item.Content)
	}
	return strings.Join(lines, "\n")
}

// JSON serialization

func valuesToJSON(results []Value) interface{} {
	if len(results) == 1 {
		return valueToJSON(results[0])
	}
	arr := make([]interface{}, len(results))
	for i, v := range results {
		arr[i] = valueToJSON(v)
	}
	return arr
}

func valueToJSON(v Value) interface{} {
	switch v.Kind {
	case ValNull:
		return nil
	case ValBool:
		return v.Bool
	case ValNumber:
		return v.Number
	case ValString:
		return v.Str
	case ValHeading:
		if v.Heading != nil {
			return map[string]interface{}{
				"level": v.Heading.Level,
				"text":  v.Heading.Text,
				"line":  v.Heading.Line,
			}
		}
	case ValCode:
		if v.Code != nil {
			return map[string]interface{}{
				"language":   v.Code.Language,
				"content":    v.Code.Content,
				"start_line": v.Code.StartLine,
				"end_line":   v.Code.EndLine,
			}
		}
	case ValLink:
		if v.Link != nil {
			return map[string]interface{}{
				"text": v.Link.Text,
				"url":  v.Link.URL,
				"type": v.Link.LinkType,
			}
		}
	case ValImage:
		if v.Image != nil {
			return map[string]interface{}{
				"alt":   v.Image.Alt,
				"src":   v.Image.Src,
				"title": v.Image.Title,
			}
		}
	case ValTable:
		if v.Table != nil {
			return map[string]interface{}{
				"headers":    v.Table.Headers,
				"rows":       v.Table.Rows,
				"alignments": v.Table.Alignments,
			}
		}
	case ValList:
		if v.List != nil {
			items := make([]interface{}, len(v.List.Items))
			for i, item := range v.List.Items {
				m := map[string]interface{}{"content": item.Content}
				if item.Checked != nil {
					m["checked"] = *item.Checked
				}
				items[i] = m
			}
			return map[string]interface{}{
				"ordered": v.List.Ordered,
				"items":   items,
			}
		}
	case ValArray:
		arr := make([]interface{}, len(v.Array))
		for i, el := range v.Array {
			arr[i] = valueToJSON(el)
		}
		return arr
	case ValObject:
		m := make(map[string]interface{})
		for k, val := range v.Object {
			m[k] = valueToJSON(val)
		}
		return m
	case ValDocument:
		if v.Document != nil {
			return map[string]interface{}{
				"heading_count": v.Document.HeadingCount,
				"word_count":    v.Document.WordCount,
			}
		}
	}
	return v.ToText()
}
