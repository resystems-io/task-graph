package main

import (
	"context"
	_ "embed"
	"os"
	"strings"

	"github.com/google/go-github/v52/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"go.resystems.io/task-graph/internal/taskgraph"
)

var (
	mermaid_with_html   bool   = false
	mermaid_with_fence  bool   = false
	mermaid_dir         string = "TB"
	mermaid_skip_closed bool   = false
)

func init() {
	rootCmd.AddCommand(listMermaidCmd)

	listMermaidCmd.Flags().BoolVarP(&mermaid_with_html, "browser", "b", false, "encase in HTML for viewing in a browser")
	listMermaidCmd.Flags().BoolVarP(&mermaid_with_fence, "fence", "f", false, "encase in ```mermaid ... ``` fence")
	listMermaidCmd.Flags().StringVarP(&mermaid_dir, "dir", "d", "TB", "use TB or LR flow direction")
	listMermaidCmd.Flags().BoolVarP(&mermaid_skip_closed, "skip-closed", "c", false, "skip traversing closed issues")
}

//go:embed mermaid.head.html
var mermaid_head_html string

//go:embed mermaid.tail.html
var mermaid_tail_html string

const (
	mermaid_head_fence = "```mermaid\n"
	mermaid_tail_fence = "\n```\n"
)

var listMermaidCmd = &cobra.Command{
	Use:   "mermaid",
	Short: "generate a mermaid graph of tasks.",
	Long: `Fetch tasklists embedded in a root issue
and produce a mermaid graph thereof.

# Example

task-graph -o resystems-io -r architecture -n 2 -v mermaid -d LR -b -c > tg-2.html
`,
	Run: func(cmd *cobra.Command, args []string) {
		if root_issue != "" {
			panic("issue tags not yet supported")
		}

		// authenticate to github
		ghtok, err := github_access_token(root_access)
		if err != nil {
			panic(err)
		}
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: ghtok},
		)
		ctx := context.Background()
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)

		// accumulate linked issues
		tg := taskgraph.TaskGraph{}
		tg.Verbose(root_verbose)
		tg.SkipClosed(mermaid_skip_closed)

		rootIssues := make([]*taskgraph.IssueRef, len(root_issue_numbers))
		for i, n := range root_issue_numbers {
			rootIssue := &taskgraph.IssueRef{Owner: root_issue_owner, Repo: root_issue_repo, Number: n}
			rootIssues[i] = rootIssue
		}
		err = tg.Accumulate(ctx, client, rootIssues...)
		if err != nil {
			panic(err)
		}

		if mermaid_with_html {
			os.Stdout.WriteString(mermaid_head_html)
			defer os.Stdout.WriteString(mermaid_tail_html)
		} else if mermaid_with_fence {
			os.Stdout.WriteString(mermaid_head_fence)
			defer os.Stdout.WriteString(mermaid_tail_fence)
		}
		mermaid_dir = strings.ToUpper(mermaid_dir)
		err = tg.ToMermaid(os.Stdout, mermaid_dir)
		if err != nil {
			panic(err)
		}
	},
}
