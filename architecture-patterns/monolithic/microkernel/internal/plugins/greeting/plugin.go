package greeting

type Plugin struct{}

func NewPlugin() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "greeting"
}

func (p *Plugin) Execute(input map[string]interface{}) (map[string]interface{}, error) {
	name, ok := input["name"].(string)
	if !ok || name == "" {
		name = "World"
	}
	return map[string]interface{}{
		"message": "Hello, " + name + "!",
	}, nil
}
