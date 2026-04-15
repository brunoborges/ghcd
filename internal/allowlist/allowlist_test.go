package allowlist

import "testing"

func TestCacheableCommands(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		args     []string
		wantType CommandType
		wantRes  ResourceType
	}{
		{[]string{"pr", "list"}, Cacheable, ResourcePR},
		{[]string{"pr", "view", "42"}, Cacheable, ResourcePR},
		{[]string{"pr", "status"}, Cacheable, ResourcePR},
		{[]string{"issue", "list"}, Cacheable, ResourceIssue},
		{[]string{"issue", "view", "1"}, Cacheable, ResourceIssue},
		{[]string{"repo", "view"}, Cacheable, ResourceRepo},
		{[]string{"run", "list"}, Cacheable, ResourceRun},
		{[]string{"release", "list"}, Cacheable, ResourceRelease},
		{[]string{"label", "list"}, Cacheable, ResourceLabel},
		{[]string{"search", "repos", "go"}, Cacheable, ResourceSearch},
	}

	for _, tt := range tests {
		cl := c.Classify(tt.args)
		if cl.Type != tt.wantType {
			t.Errorf("Classify(%v): type=%d, want %d", tt.args, cl.Type, tt.wantType)
		}
		if cl.Resource != tt.wantRes {
			t.Errorf("Classify(%v): resource=%s, want %s", tt.args, cl.Resource, tt.wantRes)
		}
	}
}

func TestMutations(t *testing.T) {
	c := NewClassifier(nil)

	tests := []struct {
		args     []string
		wantType CommandType
	}{
		{[]string{"pr", "create"}, Mutation},
		{[]string{"pr", "merge", "42"}, Mutation},
		{[]string{"pr", "close", "42"}, Mutation},
		{[]string{"issue", "create"}, Mutation},
		{[]string{"issue", "edit", "1"}, Mutation},
		{[]string{"issue", "delete", "1"}, Mutation},
	}

	for _, tt := range tests {
		cl := c.Classify(tt.args)
		if cl.Type != tt.wantType {
			t.Errorf("Classify(%v): type=%d, want %d", tt.args, cl.Type, tt.wantType)
		}
	}
}

func TestPassthrough(t *testing.T) {
	c := NewClassifier(nil)

	tests := [][]string{
		{"auth", "login"},
		{"config", "set"},
		{"codespace", "ssh"},
		{},
	}

	for _, args := range tests {
		cl := c.Classify(args)
		if cl.Type != Passthrough {
			t.Errorf("Classify(%v): type=%d, want Passthrough", args, cl.Type)
		}
	}
}

func TestAPIClassification(t *testing.T) {
	c := NewClassifier(nil)

	// GET (default) → cacheable
	cl := c.Classify([]string{"api", "/repos/cli/cli"})
	if cl.Type != Cacheable {
		t.Errorf("GET api: type=%d, want Cacheable", cl.Type)
	}

	// POST → mutation
	cl = c.Classify([]string{"api", "-X", "POST", "/repos/cli/cli/issues"})
	if cl.Type != Mutation {
		t.Errorf("POST api: type=%d, want Mutation", cl.Type)
	}

	// --method DELETE → mutation
	cl = c.Classify([]string{"api", "--method", "DELETE", "/repos/cli/cli/issues/1"})
	if cl.Type != Mutation {
		t.Errorf("DELETE api: type=%d, want Mutation", cl.Type)
	}
}

func TestAdditionalCacheable(t *testing.T) {
	c := NewClassifier([]string{"gh status"})
	cl := c.Classify([]string{"status"})
	if cl.Type != Cacheable {
		t.Errorf("additional cacheable 'status': type=%d, want Cacheable", cl.Type)
	}
}
