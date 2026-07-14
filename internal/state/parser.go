package state

import (
	"encoding/json"
	"fmt"
	"io"

	"driftdetect/internal/model"
)

// tfState models the Terraform v4 JSON state document (serial >= 4). Earlier
// serial formats used a "modules" array which is not supported here.
type tfState struct {
	Version   int         `json:"version"`
	Serial    uint64      `json:"serial"`
	Resources []tfResource `json:"resources"`
}

type tfResource struct {
	Mode      string       `json:"mode"`
	Type      string       `json:"type"`
	Name      string       `json:"name"`
	Provider  string       `json:"provider"`
	Instances []tfInstance `json:"instances"`
}

type tfInstance struct {
	Attributes map[string]interface{} `json:"attributes"`
}

// ParseState decodes a Terraform state document and normalizes every managed
// resource instance into a model.ResourceState. Only resources in "managed"
// mode are considered (data sources are ignored).
func ParseState(r io.Reader) ([]model.ResourceState, error) {
	var st tfState
	if err := json.NewDecoder(r).Decode(&st); err != nil {
		return nil, fmt.Errorf("decode terraform state: %w", err)
	}

	var out []model.ResourceState
	for _, res := range st.Resources {
		if res.Mode != "" && res.Mode != "managed" {
			continue
		}
		for i, inst := range res.Instances {
			rs := normalizeInstance(res, inst, i)
			// Skip resources without a usable identifier — they cannot be
			// matched against cloud state.
			if rs.ID == "" {
				continue
			}
			out = append(out, rs)
		}
	}
	return out, nil
}

// normalizeInstance extracts the identifier, attributes, and tags for one
// resource instance.
func normalizeInstance(res tfResource, inst tfInstance, idx int) model.ResourceState {
	attrs := inst.Attributes
	if attrs == nil {
		attrs = map[string]interface{}{}
	}

	rs := model.ResourceState{
		Provider:   providerShortName(res.Provider),
		Type:       res.Type,
		Name:       res.Name,
		ID:         stringAttr(attrs, "id"),
		Attributes: attrs,
		Tags:       normalizeTags(attrs["tags"]),
	}

	// Some resources expose their identifier via "arn" instead of "id".
	if rs.ID == "" {
		if arn := stringAttr(attrs, "arn"); arn != "" {
			rs.ID = arn
		}
	}
	_ = idx
	return rs
}

// providerShortName reduces a Terraform provider address such as
// "provider[\"registry.terraform.io/hashicorp/aws\"]" to "aws".
func providerShortName(p string) string {
	if p == "" {
		return ""
	}
	// provider["registry.terraform.io/hashicorp/aws"] -> hashicorp/aws
	start := indexByte(p, '[')
	end := lastIndexByte(p, ']')
	if start >= 0 && end > start {
		inner := p[start+1 : end]
		inner = trimQuotes(inner)
		if i := lastIndexByte(inner, '/'); i >= 0 {
			return inner[i+1:]
		}
		return inner
	}
	return p
}

func stringAttr(attrs map[string]interface{}, key string) string {
	if v, ok := attrs[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// normalizeTags converts a raw tags value (map[string]interface{} from JSON)
// into a flat map[string]string.
func normalizeTags(raw interface{}) map[string]string {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func lastIndexByte(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func trimQuotes(s string) string {
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"') {
		return s[1 : len(s)-1]
	}
	return s
}
