package main

import (
	"bytes"
	_ "embed"
	"os"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

//go:embed github.curl.tmpl
var github_curl string

//go:embed github.jq.export.tmpl
var github_jq_export string

//go:embed github.jq.group-by.tmpl
var github_jq_group_by string

//go:embed github.jq.select.tmpl
var github_jq_select string

//go:embed github.project.issues.graphql.tmpl
var github_project_issues_graphql string

//go:embed github.project.list.graphql.tmpl
var github_project_list_graphql string

var (
	gh_node_id      string
	gh_column_name  string
	gh_column_value string
)

func init() {
	rootCmd.AddCommand(cheatsheetCmd)
	cheatsheetCmd.AddCommand(ghCurlCmd)
	ghCurlCmd.AddCommand(ghCurlProjectListCmd)
	ghCurlCmd.AddCommand(ghCurlProjectIssuesCmd)
	ghCurlCmd.AddCommand(ghCurlProjectGroupByCmd)
	ghCurlCmd.AddCommand(ghCurlProjectSelectCmd)
	ghCurlCmd.AddCommand(ghCurlProjectExportCmd)

	ghCurlCmd.Flags().StringVarP(&gh_node_id, "node-id", "n", "", "GitHub project node ID")
	ghCurlCmd.Flags().StringVar(&gh_column_name, "column-name", "", "GitHub project column name")
	ghCurlCmd.Flags().StringVar(&gh_column_value, "column-value", "", "GitHub project column value")
}

var cheatsheetCmd = &cobra.Command{
	Use:   "cheat-sheet",
	Short: "Various command line cheat-sheets.",
	Long: `Cheat-sheets for using cURL and jq etc.

In most cases the cheat sheet output can simply be piped to sh (or bash).`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

var ghCurlCmd = &cobra.Command{
	Use:   "gh",
	Short: "Various GitHub cURL examples",
	Long:  `Cheat-sheets for using cURL and jq etc.`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

type CheatCurl struct {
	Query              string
	AuthorisationToken string
	Piped              string
}

type CheatSheet struct {
	Organisation string
	NodeID       string
	Column       string
	ColumnValue  string
}

func ghCurl(query string, pipe string) {
	// load the access token
	ghtok, err := github_access_token(root_access)
	if err != nil {
		panic(err)
	}

	// build the cURL
	curl := CheatCurl{
		Query:              query,
		AuthorisationToken: ghtok,
		Piped:              pipe,
	}
	curltmpl, err := template.New("gh-curl").Parse(github_curl)
	if err != nil {
		panic(err)
	}
	err = curltmpl.Execute(os.Stdout, curl)
	if err != nil {
		panic(err)
	}
}

func ghProjectList() string {

	// build the query
	cheat := CheatSheet{
		Organisation: root_issue_owner,
	}
	querytmpl, err := template.New("gh-query").Parse(strings.TrimRight(github_project_list_graphql, "\n\r\t "))
	if err != nil {
		panic(err)
	}
	first := bytes.Buffer{}
	err = querytmpl.Execute(&first, cheat)
	if err != nil {
		panic(err)
	}

	return first.String()
}

func ghProjectIssues() string {

	// build the query
	cheat := CheatSheet{
		Organisation: root_issue_owner,
		NodeID:       gh_node_id,
		Column:       gh_column_name,
		ColumnValue:  gh_column_value,
	}
	querytmpl, err := template.New("gh-query").Parse(strings.TrimRight(github_project_issues_graphql, "\n\r\t "))
	if err != nil {
		panic(err)
	}
	first := bytes.Buffer{}
	err = querytmpl.Execute(&first, cheat)
	if err != nil {
		panic(err)
	}

	return first.String()
}

func ghPipe(pipe string) string {

	// build the query
	cheat := CheatSheet{
		Organisation: root_issue_owner,
		NodeID:       gh_node_id,
		Column:       gh_column_name,
		ColumnValue:  gh_column_value,
	}
	querytmpl, err := template.New("gh-pipe").Parse(strings.TrimRight(pipe, "\n\r\t "))
	if err != nil {
		panic(err)
	}
	first := bytes.Buffer{}
	err = querytmpl.Execute(&first, cheat)
	if err != nil {
		panic(err)
	}

	return first.String()
}

var ghCurlProjectListCmd = &cobra.Command{
	Use:   "projects",
	Short: "List GitHub projects.",
	Long:  `Cheat-sheets for using cURL and jq etc.`,
	Run: func(cmd *cobra.Command, args []string) {
		query := ghProjectList()
		ghCurl(query, "jq .")
	},
}

var ghCurlProjectIssuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Group GitHub project issues.",
	Long:  `Cheat-sheets for using cURL and jq etc.`,
	Run: func(cmd *cobra.Command, args []string) {

		query := ghProjectIssues()
		pipe := "jq ."
		ghCurl(query, pipe)

	},
}

var ghCurlProjectGroupByCmd = &cobra.Command{
	Use:   "group-by",
	Short: "Group GitHub project issues by a given column.",
	Long: `Cheat-sheets for using cURL and jq etc.

# Example

task-graph -o resystems-io cheat-sheet gh --column-name Customer --column-value ACME -n PVT_xyz group-by | bash
`,
	Run: func(cmd *cobra.Command, args []string) {

		query := ghProjectIssues()
		pipe := ghPipe(github_jq_group_by)
		ghCurl(query, pipe)

	},
}

var ghCurlProjectSelectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select GitHub project issues by a given column.",
	Long:  `Cheat-sheets for using cURL and jq etc.`,
	Run: func(cmd *cobra.Command, args []string) {

		query := ghProjectIssues()
		pipe := ghPipe(github_jq_select)
		ghCurl(query, pipe)

	},
}

var ghCurlProjectExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export GitHub project issues, select by a given column, as a list.",
	Long:  `Cheat-sheets for using cURL and jq etc.`,
	Run: func(cmd *cobra.Command, args []string) {

		query := ghProjectIssues()
		pipe := ghPipe(github_jq_export)
		ghCurl(query, pipe)

	},
}
