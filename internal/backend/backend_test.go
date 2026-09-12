package backend

import (
	"bytes"
	"testing"
)

func TestTraceCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "ordinary argv", args: []string{"list", "--formula", "jq"}, want: "$ brew list --formula jq\n"},
		{name: "empty argument", args: []string{""}, want: "$ brew ''\n"},
		{name: "unsafe characters", args: []string{"with space", "$HOME;"}, want: "$ brew 'with space' '$HOME;'\n"},
		{name: "single quote", args: []string{"it's"}, want: "$ brew 'it'\"'\"'s'\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := TraceCommand(&out, "brew", test.args...); err != nil {
				t.Fatalf("TraceCommand() error = %v", err)
			}
			if got := out.String(); got != test.want {
				t.Fatalf("TraceCommand() = %q, want %q", got, test.want)
			}
		})
	}
}
