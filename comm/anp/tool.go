package anp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/costa92/llm-agent"
)

// AsAgentTool exposes Registry operations as a single agents.Tool.
// Schema actions:
//
//   - register   — args: {service: Service}
//   - unregister — args: {id: string}
//   - discover   — args: {service_type: string}
//   - get_best   — args: {service_type: string}
//   - stats      — no args
func AsAgentTool(reg *Registry) agents.Tool {
	return agents.NewFuncTool(
		"anp_registry",
		"Agent Network Protocol service registry: register/discover/get_best.",
		json.RawMessage(`{
			"type":"object",
			"properties":{
				"action":{"type":"string","enum":["register","unregister","discover","get_best","stats"]},
				"service_type":{"type":"string"},
				"id":{"type":"string"},
				"service":{"type":"object"}
			},
			"required":["action"]
		}`),
		func(_ context.Context, raw json.RawMessage) (string, error) {
			var p struct {
				Action      string   `json:"action"`
				ServiceType string   `json:"service_type"`
				ID          string   `json:"id"`
				Service     *Service `json:"service"`
			}
			if err := json.Unmarshal(raw, &p); err != nil {
				return "", fmt.Errorf("anp/tool: bad args: %w", err)
			}
			switch p.Action {
			case "register":
				if p.Service == nil {
					return "", errors.New("anp/tool: service required for register")
				}
				if err := reg.Register(p.Service); err != nil {
					return "", err
				}
				out, _ := json.Marshal(map[string]string{"registered": p.Service.ID})
				return string(out), nil
			case "unregister":
				if p.ID == "" {
					return "", errors.New("anp/tool: id required for unregister")
				}
				reg.Unregister(p.ID)
				out, _ := json.Marshal(map[string]string{"unregistered": p.ID})
				return string(out), nil
			case "discover":
				services := reg.Discover(p.ServiceType)
				out, _ := json.Marshal(services)
				return string(out), nil
			case "get_best":
				best, err := reg.GetBest(p.ServiceType, nil)
				if err != nil {
					return "", err
				}
				out, _ := json.Marshal(best)
				return string(out), nil
			case "stats":
				out, _ := json.Marshal(map[string]int{"count": reg.Stats()})
				return string(out), nil
			default:
				return "", fmt.Errorf("anp/tool: unknown action %q", p.Action)
			}
		},
	)
}
