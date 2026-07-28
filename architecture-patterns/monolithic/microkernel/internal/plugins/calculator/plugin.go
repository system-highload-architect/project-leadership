package calculator

import (
	"fmt"
)

type Plugin struct{}

func NewPlugin() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "calculator"
}

func (p *Plugin) Execute(input map[string]interface{}) (map[string]interface{}, error) {
	a, okA := input["a"].(float64)
	b, okB := input["b"].(float64)
	if !okA || !okB {
		return nil, fmt.Errorf("missing 'a' or 'b' (numbers)")
	}
	op, _ := input["op"].(string)
	var result float64
	switch op {
	case "add":
		result = a + b
	case "sub":
		result = a - b
	case "mul":
		result = a * b
	case "div":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return nil, fmt.Errorf("unknown op: %s", op)
	}
	return map[string]interface{}{
		"result": result,
	}, nil
}
