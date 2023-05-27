package taskgraph

import (
	"bytes"
	"context"
	"fmt"
	htm "html"
	"io"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/google/go-github/v52/github"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

// -- github traversal

var validGitHubID = regexp.MustCompile(`^(([a-zA-Z0-9-_]+)/(([a-zA-Z0-9-_]+/?)+))?#([0-9]+)$`)
var validGitHubIssue = regexp.MustCompile(`^/([a-zA-Z0-9-_]+)/(([a-zA-Z0-9-_]+/?)+)/issues/([0-9]+)$`)

var _tgStderr = os.Stderr
var _tgDiscard = io.Discard
var _tgLog = log.New(_tgDiscard, "[resys-task-graph] ", log.LstdFlags|log.Lmsgprefix)
var _tgVerboseLog = log.New(_tgDiscard, "[resys-task-graph] ", log.LstdFlags|log.Lmsgprefix)

const (
	github_closed = "closed"
	github_open = "open"
)

type IssueRef struct {
	Owner  string
	Repo   string
	Number int
}

type IssueHandle struct {
	*IssueRef
	Issue *github.Issue
}

func (is *IssueRef) String() string {
	sep := "/"
	if len(is.Owner) == 0 || len(is.Repo) == 0 {
		sep = ""
	}
	return fmt.Sprintf("%s%s%s#%d", is.Owner, sep, is.Repo, is.Number)
}

type TaskGraph struct {
	Refs  map[string]*IssueHandle
	Edges map[string][]string

	skip_closed bool
}

func (tg* TaskGraph) SkipClosed(toggle bool) bool {
	was := tg.skip_closed
	tg.skip_closed = toggle
	return was
}

func (tg* TaskGraph) Verbose(toggle bool) {
	writer := _tgDiscard
	if toggle {
		writer = _tgStderr
	}
	_tgLog = log.New(writer, "[resys-task-graph] ", log.LstdFlags|log.Lmsgprefix)
}

func unique(s []string) []string {
	in := make(map[string]bool)
	var uniq []string
	for _, str := range s {
		if _, ok := in[str]; !ok {
			in[str] = true
			uniq = append(uniq, str)
		}
	}
	return uniq
}

func (tg *TaskGraph) init() {
	if tg.Refs == nil {
		tg.Refs = make(map[string]*IssueHandle, 100)
	}
	if tg.Edges == nil {
		tg.Edges = make(map[string][]string, 10)
	}
}

func (tg *TaskGraph) Accumulate(ctx context.Context, client *github.Client, is ...*IssueRef) error {
	tg.init()

	// seed the list
	pending := make([]*IssueRef,0,len(is))
	pending = append(pending, is...)

winnow:
	for {
		if len(pending) == 0 {
			break winnow
		}

		// perform a parallel fetch with limits
		g, ctx := errgroup.WithContext(ctx)
		g.SetLimit(10)

		type Result struct {
			trigger *IssueRef
			issue   *github.Issue
			refs    []*IssueRef
		}

		results := make([]Result, len(pending))
		for i, r := range pending {
			ii, rr := i, r
			g.Go(func() error {
				nm := rr.String()
				_, ok := tg.Refs[nm]
				if !ok {
					// fetch the issue from github and parse
					issue, refs, err := tg.accumulateIssueRefs(ctx, client, rr)
					if err != nil {
						return err
					}
					results[ii] = Result{rr, issue, refs}
				} else {
					// skip because we have already visited this issue
					results[ii] = Result{nil, nil, nil}
				}
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}

		// now clear the pending list
		pending = pending[0:0]

		// and then gather all the results
		for _, res := range results {
			if res.trigger == nil {
				// we already visited this
				continue
			}
			// extend our pending list
			pending = append(pending, res.refs...)
			// update our nodes
			nm := res.trigger.String()
			h := IssueHandle{IssueRef: res.trigger, Issue: res.issue}
			tg.Refs[nm] = &h
			// update our edges
			ed, ok := tg.Edges[nm]
			if !ok {
				ed = make([]string, 0, len(res.refs))
			}
			for _, r := range res.refs {
				ed = append(ed, r.String())
			}
			ed = unique(ed)
			tg.Edges[nm] = ed
		}
	}

	return nil
}

// Consider 
// var iol github.IssueListOptions
// iol.Filter = ""
// could use the list endpoint to collect seed events
// could use the issue-comment webhook to stay in sync
// https://docs.github.com/en/webhooks-and-events/webhooks/webhook-events-and-payloads#issue_comment

var x_ratelimit_remaining string

func init() {
	x_ratelimit_remaining = strings.ToLower("X-Ratelimit-Remaining")
}

func (tg *TaskGraph) accumulateIssueRefs(ctx context.Context, client *github.Client, is *IssueRef) (*github.Issue, []*IssueRef, error) {
	_tgLog.Printf("traversing into %v\n", is)

	issues := make([]*IssueRef, 0, 10)

	issue, resp, err := client.Issues.Get(ctx, is.Owner, is.Repo, is.Number)
	if err != nil {
		return nil, nil, err
	}

	// log headers
	_tgVerboseLog.Printf("github-headers: %d\n", len(resp.Header))
	for k, v := range resp.Header {
		_tgVerboseLog.Printf("github-header: %v\n", k)
		if strings.HasPrefix(k, "X-") || strings.HasPrefix(k, "x-") {
			for i, x := range v {
				_log := _tgVerboseLog
				if strings.ToLower(k) == x_ratelimit_remaining {
					_log = _tgLog
				}
				_log.Printf("github-header: %v [%d] %v\n", k, i, x)
			}
		}
	}
	if issue == nil {
		return nil, nil, fmt.Errorf("nil issue for %v", is)
	}

	// check body
	if issue.Body == nil || len(*issue.Body) == 0 {
		_tgLog.Printf("nil or empty body for %v\n", is)
		return issue, issues, nil
	}
	_tgVerboseLog.Printf("%v\n", *issue.Body)

	// check state
	if tg.skip_closed && issue.GetState() == github_closed {
		return issue, issues, nil
	}

	// parse markdown
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)
	source := []byte(*issue.Body)
	reader := text.NewReader(source)
	rootAstNode := md.Parser().Parse(reader)
	_tgVerboseLog.Printf("%v\n", rootAstNode)

	// extract a part of the source (creates copies...)
	captureLines := func(n ast.Node) *bytes.Buffer {
		buf := bytes.Buffer{}
		l := n.Lines().Len()
		for i := 0; i < l; i++ {
			line := n.Lines().At(i)
			buf.Write(line.Value(source))
		}
		return &buf
	}

	carry := func(o string, r string, n int) IssueRef {
		issue := IssueRef{o, r, n}
		if len(issue.Owner) == 0 {
			issue.Owner = is.Owner
		}
		if len(issue.Repo) == 0 {
			issue.Repo = is.Repo
		}
		return issue
	}

	// parse the task list itself
	parseTasklist := func(tasklistSource []byte) error {
		tasklistReader := text.NewReader(tasklistSource)
		tasklistRootNode := md.Parser().Parse(tasklistReader)
		ast.Walk(tasklistRootNode, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering {
				_tgVerboseLog.Printf("tasklist ast n.Kind=%v n.Type=%v\n", n.Kind(), n.Type())
				switch n.Kind() {
				// parse text references
				case ast.KindText:
					txt := n.(*ast.Text)
					ref := (string)(txt.Text(tasklistSource))
					_tgVerboseLog.Printf("ast fenced text: %v\n", ref)
					matched := validGitHubID.FindStringSubmatch(ref)
					if len(matched) > 4 {
						_tgVerboseLog.Printf("ast fenced text: issue %v\n", ref)
						if len(matched) > 4 {
							owner := matched[2]
							repo := matched[3]
							numtxt := matched[5]
							num, _ := strconv.ParseInt(numtxt, 10, 32)
							number := int(num)
							issue := carry(owner, repo, number)
							_tgLog.Printf("next issue %v\n", &issue)
							issues = append(issues, &issue)
						}
					}
				// parse link references
				case ast.KindAutoLink:
					lnk := n.(*ast.AutoLink)
					url, err := url.Parse((string)(lnk.URL(tasklistSource)))
					if err != nil {
						_tgLog.Printf("ast fenced auto-url: bad url %v - %v\n",
							(string)(lnk.URL(tasklistSource)), err)
					}
					_tgVerboseLog.Printf("ast fenced auto-url: %v [%v] [%v]\n", url.String(), url.Host, url.Path)
					if url.Host == "github.com" && strings.Contains(url.Path, "/issues/") {
						matched := validGitHubIssue.FindStringSubmatch(url.Path)
						_tgVerboseLog.Printf("ast fenced auto-url: issue %v [%v] [%v]\n", url.String(), url.Host, url.Path)
						if len(matched) > 4 {
							owner := matched[1]
							repo := matched[2]
							numtxt := matched[4]
							num, _ := strconv.ParseInt(numtxt, 10, 32)
							number := int(num)
							issue := carry(owner, repo, number)
							_tgLog.Printf("next issue %v\n", &issue)
							issues = append(issues, &issue)
						}
					}
				}
			}
			return ast.WalkContinue, nil
		})
		return nil
	}

	// walk the ast to find all fenced code blocks with a language of type '[tasklist]'
	ast.Walk(rootAstNode, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			_tgVerboseLog.Printf("ast n.Kind=%v\n", n.Kind())
			switch n.Kind() {
			case ast.KindFencedCodeBlock:
				nk := n.(*ast.FencedCodeBlock)
				language := nk.Language(source)
				// segment := nk.Info.Segment
				// fmt.Fprintf(io.Discard, "ast fenced: %v %v\n", string(language), segment)
				_tgVerboseLog.Printf("ast fenced: %v\n", string(language))
				if string(language) == "[tasklist]" {
					buf := captureLines(nk)
					_tgVerboseLog.Printf("ast fenced block:\n%v\n", buf.String())
					parseTasklist(buf.Bytes())
				}
			}
		}
		return ast.WalkContinue, nil
	})

	return issue, issues, nil
}

func (tg *TaskGraph) ToMermaid(writer io.Writer, dir string) error {

	if dir != "TB" && dir != "LR" {
		return fmt.Errorf("bad graph direction: %s", dir)
	}

	// track node identifiers
	seq := 100
	// taskref -> nodeid
	ids := make(map[string]string, len(tg.Refs))
	next := func() string {
		s := seq
		seq = seq + 1
		id := fmt.Sprintf("tg%04d", s)
		return id
	}
	id := func(taskref string) string {
		x, ok := ids[taskref]
		if !ok {
			x = next()
			ids[taskref] = x
		}
		return x
	}

	// output header
	fmt.Fprintf(writer, `---
title: Task Graph
---

flowchart

subgraph Tasks

	direction %s
`, dir)

	// output classes
	defer func() {
		for k, ref := range tg.Refs {
			kid := id(k)
			if ref.Issue.GetState() == github_closed {
				fmt.Fprintf(writer, "\tclass %s closed;\n", kid)
			}
		}
	}()

	// output footer
	defer fmt.Fprintf(writer, `
end

classDef tasks fill:#fff
classDef projects fill:#eed

classDef closed fill:#ccc
classDef abandoned fill:#222222
classDef completed fill:#37e519
classDef review fill:#f55a00
classDef active fill:#e5b104
classDef parked fill:#b37fcd
classDef pending fill:#60a1ea
classDef staged fill:#f07ee9

class Tasks tasks;
`)

	// each repo is a separate subgraph
	subgraphs := make(map[string]string)
	for _,v := range tg.Refs {
		was, ok := subgraphs[v.Repo]
		subgraphs[v.Repo] = v.Owner
		if ok && was != v.Owner {
			panic(fmt.Errorf("duplicate repo %v with distinct owners %v != %v", v.Repo, v.Owner, was))
		}
	}

	// output subgraphs
	for s := range subgraphs {
		fmt.Fprintf(writer,"\n\tsubgraph %s\n\n", s)
		for k,v := range tg.Refs {
			if v.Repo != s {
				continue // skip for now
			}

			kid := id(k)
			escaped := htm.EscapeString(v.Issue.GetTitle())
			// not ideal... but mermaid breaks on " or &quot; or &#34;
			r := strings.NewReplacer(" &#34;", " &ldquo;", "&#34; ", "&rdquo; ", "&#34;", "'")
			escaped = r.Replace(escaped)
			fmt.Fprintf(writer, "\t\t%s[\"%s\"]\n", kid, escaped)
			fmt.Fprintf(writer, "\t\tclick %s href \"https://github.com/%s/%s/issues/%d\" \"Open %s\"\n",
				kid, v.Owner, v.Repo, v.Number, v.String())
			fmt.Fprintf(writer, "\n")
		}
		fmt.Fprintf(writer,"\n\tend\n")
	}

	// output edges
	for src,dstset := range tg.Edges {
		for _,dst := range dstset {
			srcid := id(src)
			dstid := id(dst)
			fmt.Fprintf(writer, "\t\t%s --> %s\n", srcid, dstid)
		}
	}

	return nil
}
