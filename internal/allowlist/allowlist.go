package allowlist

import "strings"

// CommandType classifies a gh command.
type CommandType int

const (
	Cacheable   CommandType = iota // Safe to cache
	Passthrough                    // Not cacheable, pass directly to gh
	Mutation                       // Mutating command, triggers cache invalidation
)

// ResourceType identifies the resource namespace for cache invalidation.
type ResourceType string

const (
	ResourcePR       ResourceType = "pr"
	ResourceIssue    ResourceType = "issue"
	ResourceRun      ResourceType = "run"
	ResourceWorkflow ResourceType = "workflow"
	ResourceRelease  ResourceType = "release"
	ResourceLabel    ResourceType = "label"
	ResourceRepo     ResourceType = "repo"
	ResourceAPI      ResourceType = "api"
	ResourceSearch   ResourceType = "search"
	ResourceUnknown  ResourceType = ""
)

type Classification struct {
	Type     CommandType
	Resource ResourceType
	// CmdKey is a normalized identifier like "pr_list" for metrics/TTL overrides.
	CmdKey string
}

// cacheableCommands maps "subcommand action" to resource type.
var cacheableCommands = map[string]ResourceType{
	"pr list":        ResourcePR,
	"pr view":        ResourcePR,
	"pr status":      ResourcePR,
	"pr checks":      ResourcePR,
	"pr diff":        ResourcePR,
	"issue list":     ResourceIssue,
	"issue view":     ResourceIssue,
	"issue status":   ResourceIssue,
	"repo view":      ResourceRepo,
	"run list":       ResourceRun,
	"run view":       ResourceRun,
	"workflow list":  ResourceWorkflow,
	"workflow view":  ResourceWorkflow,
	"release list":   ResourceRelease,
	"release view":   ResourceRelease,
	"search repos":   ResourceSearch,
	"search issues":  ResourceSearch,
	"search prs":     ResourceSearch,
	"label list":     ResourceLabel,
}

// mutatingSubcommands trigger cache invalidation for their resource type.
var mutatingActions = map[string]bool{
	"create":  true,
	"edit":    true,
	"delete":  true,
	"merge":   true,
	"close":   true,
	"reopen":  true,
	"comment": true,
	"review":  true,
	"approve": true,
	"ready":   true,
	"lock":    true,
	"unlock":  true,
	"pin":     true,
	"unpin":   true,
	"transfer": true,
}

var subcommandResourceMap = map[string]ResourceType{
	"pr":       ResourcePR,
	"issue":    ResourceIssue,
	"run":      ResourceRun,
	"workflow": ResourceWorkflow,
	"release":  ResourceRelease,
	"label":    ResourceLabel,
	"repo":     ResourceRepo,
}

// neverCacheSubcommands are always passed through regardless.
var neverCacheSubcommands = map[string]bool{
	"auth":      true,
	"codespace": true,
	"config":    true,
	"ssh-key":   true,
	"gpg-key":   true,
	"secret":    true,
	"variable":  true,
	"extension": true,
}

// Classifier classifies gh commands.
type Classifier struct {
	additionalCacheable map[string]ResourceType
}

// NewClassifier creates a classifier with optional additional cacheable commands.
func NewClassifier(additional []string) *Classifier {
	ac := make(map[string]ResourceType)
	for _, cmd := range additional {
		// Strip "gh " prefix if present
		cmd = strings.TrimPrefix(cmd, "gh ")
		ac[cmd] = ResourceUnknown
	}
	return &Classifier{additionalCacheable: ac}
}

// Classify determines how to handle a gh command based on its arguments.
// args should be the arguments after "gh", e.g., ["pr", "list", "--json", "number"].
func (c *Classifier) Classify(args []string) Classification {
	if len(args) == 0 {
		return Classification{Type: Passthrough}
	}

	sub := args[0]

	// Never-cache subcommands
	if neverCacheSubcommands[sub] {
		return Classification{Type: Passthrough}
	}

	// Special handling for "api" subcommand
	if sub == "api" {
		return c.classifyAPI(args)
	}

	// Check single-word commands in additional cacheable (e.g., "status")
	if res, ok := c.additionalCacheable[sub]; ok {
		return Classification{
			Type:     Cacheable,
			Resource: res,
			CmdKey:   sub,
		}
	}

	// Need at least subcommand + action
	if len(args) < 2 {
		return Classification{Type: Passthrough, Resource: subcommandResourceMap[sub]}
	}

	action := args[1]
	key := sub + " " + action

	// Check built-in cacheable list
	if res, ok := cacheableCommands[key]; ok {
		return Classification{
			Type:     Cacheable,
			Resource: res,
			CmdKey:   strings.ReplaceAll(key, " ", "_"),
		}
	}

	// Check additional cacheable from config
	if res, ok := c.additionalCacheable[key]; ok {
		return Classification{
			Type:     Cacheable,
			Resource: res,
			CmdKey:   strings.ReplaceAll(key, " ", "_"),
		}
	}

	// Check if it's a known mutation
	if mutatingActions[action] {
		return Classification{
			Type:     Mutation,
			Resource: subcommandResourceMap[sub],
			CmdKey:   strings.ReplaceAll(key, " ", "_"),
		}
	}

	return Classification{Type: Passthrough, Resource: subcommandResourceMap[sub]}
}

func (c *Classifier) classifyAPI(args []string) Classification {
	// gh api [flags] <endpoint>
	// Only cache GET requests (default method is GET)
	method := "GET"
	for i, arg := range args {
		if (arg == "-X" || arg == "--method") && i+1 < len(args) {
			method = strings.ToUpper(args[i+1])
		}
	}

	if method != "GET" {
		return Classification{Type: Mutation, Resource: ResourceAPI}
	}

	return Classification{
		Type:     Cacheable,
		Resource: ResourceAPI,
		CmdKey:   "api_get",
	}
}
