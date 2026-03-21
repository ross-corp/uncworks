package temporal

import (
	"sort"
	"testing"
)

func TestDeriveTagsFromDiff(t *testing.T) {
	tests := []struct {
		name     string
		diffStat string
		want     []string
	}{
		{
			name:     "empty input",
			diffStat: "",
			want:     nil,
		},
		{
			name:     "single Go file",
			diffStat: " internal/server/handler.go | 42 ++++++++++++++++++++++++++++++++++++------\n 1 file changed, 36 insertions(+), 6 deletions(-)\n",
			want:     []string{"go", "scope-small"},
		},
		{
			name: "mixed Go and TypeScript",
			diffStat: ` internal/temporal/workflow.go | 10 +++++++---
 web/src/views/NewRunView.tsx  | 25 +++++++++++++++++++++++++
 web/src/types/agent-run.ts    |  3 +++
 3 files changed, 35 insertions(+), 3 deletions(-)
`,
			want: []string{"go", "scope-small", "typescript"},
		},
		{
			name: "many files — large scope",
			diffStat: ` a.go    | 1 +
 b.go    | 1 +
 c.go    | 1 +
 d.go    | 1 +
 e.go    | 1 +
 f.go    | 1 +
 g.go    | 1 +
 h.go    | 1 +
 i.go    | 1 +
 j.go    | 1 +
 k.go    | 1 +
 11 files changed, 11 insertions(+)
`,
			want: []string{"go", "scope-large"},
		},
		{
			name: "medium scope",
			diffStat: ` a.py | 1 +
 b.py | 1 +
 c.py | 1 +
 d.py | 1 +
 e.py | 1 +
 5 files changed, 5 insertions(+)
`,
			want: []string{"python", "scope-medium"},
		},
		{
			name: "docs and yaml",
			diffStat: ` README.md         | 10 ++++++++--
 config.yaml       |  5 +++++
 deploy/values.yml |  3 +++
 3 files changed, 16 insertions(+), 2 deletions(-)
`,
			want: []string{"docs", "scope-small", "yaml"},
		},
		{
			name: "proto and shell",
			diffStat: ` api/v1/service.proto | 15 ++++++++++++---
 scripts/deploy.sh   |  8 ++++++++
 2 files changed, 20 insertions(+), 3 deletions(-)
`,
			want: []string{"proto", "scope-small", "shell"},
		},
		{
			name: "unknown extensions",
			diffStat: ` data/model.pickle | Bin 0 -> 1024 bytes
 1 file changed, 0 insertions(+), 0 deletions(-)
`,
			want: []string{"scope-small"},
		},
		{
			name:     "only summary line",
			diffStat: " 0 files changed\n",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveTagsFromDiff(tt.diffStat)

			// Sort both for comparison
			sort.Strings(got)
			sortedWant := make([]string, len(tt.want))
			copy(sortedWant, tt.want)
			sort.Strings(sortedWant)

			if len(got) != len(sortedWant) {
				t.Errorf("deriveTagsFromDiff() = %v, want %v", got, sortedWant)
				return
			}
			for i := range got {
				if got[i] != sortedWant[i] {
					t.Errorf("deriveTagsFromDiff()[%d] = %q, want %q", i, got[i], sortedWant[i])
				}
			}
		})
	}
}
