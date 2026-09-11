package benchmark

import "sort"

type ProfileDefinition struct {
	Name        string
	Description string
	Categories  []string
	RequiresV2  bool
}

var profileDefinitions = map[string]ProfileDefinition{
	"core":             {Name: "core", Description: "V1 collaboration correctness and reliability", Categories: []string{"core"}},
	"security":         {Name: "security", Description: "authorization, isolation, and privacy gates", Categories: []string{"security"}},
	"server":           {Name: "server", Description: "V2 Server lifecycle, policy, and membership", Categories: []string{"server"}, RequiresV2: true},
	"permissions":      {Name: "permissions", Description: "roles, permissions, administration, and audit", Categories: []string{"permissions"}, RequiresV2: true},
	"groups":           {Name: "groups", Description: "server-scoped groups and membership", Categories: []string{"groups"}, RequiresV2: true},
	"forums":           {Name: "forums", Description: "posts, visibility, mentions, and notifications", Categories: []string{"forums"}, RequiresV2: true},
	"agent-efficiency": {Name: "agent-efficiency", Description: "canonical jobs and effort metrics", Categories: []string{"efficiency"}, RequiresV2: true},
	"declarative":      {Name: "declarative", Description: "manifests, batches, idempotency, and deltas", Categories: []string{"declarative"}, RequiresV2: true},
	"transport":        {Name: "transport", Description: "discovery and cross-transport equivalence", Categories: []string{"transport"}, RequiresV2: true},
	"browser":          {Name: "browser", Description: "human collaboration and administration UI", Categories: []string{"core", "browser"}},
	"reliability":      {Name: "reliability", Description: "resume, deltas, and zero-loss behavior", Categories: []string{"reliability"}},
	"load":             {Name: "load", Description: "large history and mixed concurrency", Categories: []string{"load"}},
	"full":             {Name: "full", Description: "all V1 and V2 benchmark categories", Categories: []string{"core", "security", "server", "permissions", "groups", "forums", "efficiency", "declarative", "transport", "browser", "reliability", "load"}, RequiresV2: true},
}

func Profiles() []ProfileDefinition {
	out := make([]ProfileDefinition, 0, len(profileDefinitions))
	for _, p := range profileDefinitions {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func ProfileNames() []string {
	ps := Profiles()
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Name
	}
	return out
}
func Profile(name string) (ProfileDefinition, bool) { p, ok := profileDefinitions[name]; return p, ok }
func profileAllows(name, category string) bool {
	p, ok := profileDefinitions[name]
	if !ok {
		return false
	}
	for _, x := range p.Categories {
		if x == category {
			return true
		}
	}
	return false
}
