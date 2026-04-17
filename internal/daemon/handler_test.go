package daemon

import "testing"

func TestSanitizeCmdKey(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "two args", args: []string{"pr", "list"}, want: "pr_list"},
		{name: "many args", args: []string{"api", "-H", "Authorization: token secret", "/repos"}, want: "api_-H"},
		{name: "single arg", args: []string{"auth"}, want: "auth"},
		{name: "empty args", args: nil, want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeCmdKey(tt.args)
			if got != tt.want {
				t.Errorf("sanitizeCmdKey(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}
