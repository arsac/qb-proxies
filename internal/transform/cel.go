package transform

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

type Engine struct {
	env      *cel.Env
	programs map[string]cel.Program
}

type RSSItem struct {
	Title       string
	Link        string
	Description string
	GUID        string
	PubDate     string
	Enclosure   Enclosure
	// Custom fields for non-standard feeds
	InfoHash string
	Size     string
	Category string
}

type Enclosure struct {
	URL    string
	Length string
	Type   string
}

func NewEngine() (*Engine, error) {
	env, err := cel.NewEnv(
		cel.Variable("item", cel.MapType(cel.StringType, cel.AnyType)),
		cel.Variable("title", cel.StringType),
		cel.Variable("link", cel.StringType),
		cel.Variable("description", cel.StringType),
		cel.Variable("guid", cel.StringType),
		cel.Variable("pubDate", cel.StringType),
		cel.Variable("enclosureUrl", cel.StringType),
		cel.Variable("enclosureLength", cel.StringType),
		cel.Variable("enclosureType", cel.StringType),
		// Custom fields for non-standard feeds
		cel.Variable("infohash", cel.StringType),
		cel.Variable("size", cel.StringType),
		cel.Variable("category", cel.StringType),
	)
	if err != nil {
		return nil, fmt.Errorf("creating CEL env: %w", err)
	}

	return &Engine{
		env:      env,
		programs: make(map[string]cel.Program),
	}, nil
}

func (e *Engine) Compile(name, expr string) error {
	ast, issues := e.env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("compiling expression %s: %w", name, issues.Err())
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return fmt.Errorf("creating program %s: %w", name, err)
	}

	e.programs[name] = prg
	return nil
}

func (e *Engine) Eval(name string, item *RSSItem) (string, error) {
	prg, ok := e.programs[name]
	if !ok {
		return "", fmt.Errorf("program %s not found", name)
	}

	vars := map[string]interface{}{
		"item": map[string]interface{}{
			"title":           item.Title,
			"link":            item.Link,
			"description":     item.Description,
			"guid":            item.GUID,
			"pubDate":         item.PubDate,
			"enclosureUrl":    item.Enclosure.URL,
			"enclosureLength": item.Enclosure.Length,
			"enclosureType":   item.Enclosure.Type,
			"infohash":        item.InfoHash,
			"size":            item.Size,
			"category":        item.Category,
		},
		"title":           item.Title,
		"link":            item.Link,
		"description":     item.Description,
		"guid":            item.GUID,
		"pubDate":         item.PubDate,
		"enclosureUrl":    item.Enclosure.URL,
		"enclosureLength": item.Enclosure.Length,
		"enclosureType":   item.Enclosure.Type,
		"infohash":        item.InfoHash,
		"size":            item.Size,
		"category":        item.Category,
	}

	out, _, err := prg.Eval(vars)
	if err != nil {
		return "", fmt.Errorf("evaluating %s: %w", name, err)
	}

	return valToString(out), nil
}

func valToString(v ref.Val) string {
	switch v.Type() {
	case types.StringType:
		return v.Value().(string)
	default:
		return fmt.Sprintf("%v", v.Value())
	}
}
