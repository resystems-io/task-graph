package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v52/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"go.resystems.io/task-graph/internal/taskgraph"
)

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(listPersonalReposCmd)
	rootCmd.AddCommand(listPublicReposCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list all connected issues.",
	Long: `Fetch tasklists embedded in a root issue, and walk
the graph of issue from there.
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

		// check root numbers
		if len(root_issue_numbers) == 0 {
			return
		}

		// accumulate linked issues
		rootIssue := &taskgraph.IssueRef{Owner:root_issue_owner, Repo:root_issue_repo, Number:root_issue_numbers[0]}
		tg := taskgraph.TaskGraph{}
		tg.Verbose(root_verbose)
		err = tg.Accumulate(ctx, client, rootIssue)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(os.Stdout, "nodes: %v\n", tg.Refs)
		fmt.Fprintf(os.Stdout, "edges: %v\n", tg.Edges)

		for k, h := range tg.Refs {
			fmt.Fprintf(os.Stdout, "%s %s\n", k, *h.Issue.Title)
		}
	},
}

var listPersonalReposCmd = &cobra.Command{
	Use:   "list-personal-repos",
	Short: "list all connected issues.",
	Long: `Fetch tasklists embedded in a root issue, and walk
the graph of issue from there.
`,
	Run: func(cmd *cobra.Command, args []string) {
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

		// list all repositories for the authenticated user
		opt := &github.RepositoryListOptions{
			ListOptions: github.ListOptions{PerPage: 100},
		}
		repos, _, err := client.Repositories.List(ctx, "", opt)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(os.Stderr, "len(repos)=%d\n", len(repos))
		for i, r := range repos {
			fmt.Fprintf(os.Stderr, "repo[%d] %v\n", i, *r.URL)
		}
	},
}

var listPublicReposCmd = &cobra.Command{
	Use:   "list-public",
	Short: "list all connected issues.",
	Long: `Fetch tasklists embedded in a root issue, and walk
the graph of issue from there.
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(os.Stderr, "traverse started at %s\n", root_issue)

		client := github.NewClient(nil)

		// list public repositories for org "github"
		opt := &github.RepositoryListByOrgOptions{Type: "public"}
		repos, _, err := client.Repositories.ListByOrg(context.Background(), "github", opt)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(os.Stderr, "len(repos)=%d\n", len(repos))
		for i, r := range repos {
			fmt.Fprintf(os.Stderr, "repo[%d] %v\n", i, *r.URL)
		}
	},
}
