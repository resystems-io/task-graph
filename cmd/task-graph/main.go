package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	_ "github.com/google/go-github/v52/github"
	"github.com/spf13/cobra"
	_ "github.com/yuin/goldmark"
)

var (
	root_access        string
	root_verbose       bool
	root_issue         string // owner/repo#123
	root_issue_owner   string
	root_issue_repo    string
	root_issue_numbers []int
)

func init() {
	rootCmd.Flags().StringVarP(&root_access, "access-token", "a", "~/.config/task-graph/github_access_token", "file from which to load the GitHub access token.")
	rootCmd.Flags().BoolVarP(&root_verbose, "verbose", "v", false, "verbose output to stderr")
	rootCmd.Flags().StringVarP(&root_issue, "issue", "i", "", "root issue owner/repo#123")
	rootCmd.Flags().StringVarP(&root_issue_owner, "issue-owner", "o", "", "root issue owner")
	rootCmd.Flags().StringVarP(&root_issue_repo, "issue-repo", "r", "", "root issue repo")
	rootCmd.Flags().IntSliceVarP(&root_issue_numbers, "issue-number", "n", []int{1}, "root issue number (repeat for multiple roots)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func github_access_token(token_path string) (string, error) {

	// replace the user's home path
	if strings.HasPrefix(token_path, "~") {
		if dirname, err := os.UserHomeDir(); err != nil {
			return "", err
		} else {
			token_path = strings.TrimPrefix(token_path, "~")
			token_path = path.Join(dirname, token_path)
		}
	}

	// load the token
	if data, err := os.ReadFile(token_path); err != nil {
		return "", err
	} else {
		github_access_token := string(data)
		github_access_token = strings.TrimRight(github_access_token, "\n\r\t ")
		return github_access_token, nil
	}
}

var rootCmd = &cobra.Command{
	Use:   "task-graph",
	Short: "task-graph produces a graph view of issues.",
	Long: `A directed graph generator that ingests GitHub issues
and produces graph views of your tasks.
`,
	TraverseChildren: true, // note traveral can only be set on the root
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(os.Stderr, "Please select a subcommand.\n")
	},
}
