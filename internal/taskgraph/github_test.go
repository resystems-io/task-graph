package taskgraph

import (
	"fmt"
	"strconv"
)

func ExampleRegexp_gitHubIDMatch() {

	type Expect struct {
		id    string
		match bool
	}

	check := []Expect{
		{"resystems-io/task-graph#123", true},
		{"#123", true},
		{"resystems-io/long/path#123", true},
		{"bogus#123", false},
	}

	for k, v := range check {
		matched := validGitHubID.Copy().MatchString(v.id)
		fmt.Printf("[%d] %v : %v == %v\n", k, v.id, matched, v.match)
	}

	// Output:
	// [0] resystems-io/task-graph#123 : true == true
	// [1] #123 : true == true
	// [2] resystems-io/long/path#123 : true == true
	// [3] bogus#123 : false == false
}

func ExampleRegexp_gitHubIssueMatch() {

	type Expect struct {
		id    string
		match bool
	}

	check := []Expect{
		{"/resystems-io/task-graph/issues/123", true},
		{"/resystems-io/this/and/that/issues/123", true},
		{"resystems-io/this/and/that/issues/123", false},
	}

	for k, v := range check {
		matched := validGitHubIssue.Copy().MatchString(v.id)
		fmt.Printf("[%d] %v : %v == %v\n", k, v.id, matched, v.match)
	}

	// Output:
	// [0] /resystems-io/task-graph/issues/123 : true == true
	// [1] /resystems-io/this/and/that/issues/123 : true == true
	// [2] resystems-io/this/and/that/issues/123 : false == false
}

	type Expect struct {
		id     string
		owner  string
		repo   string
		number int
	}

func ExampleRegexp_gitHubIDParts() {

	check := []Expect{
		{"resystems-io/task-graph#123", "resystems-io", "task-graph", 123},
		{"resystems-io/this/and/that#123", "resystems-io", "this/and/that", 123},
		{"resystems-io#123", "-", "-", 0},
		{"#123", "", "", 123},
	}

	for k, v := range check {
		matched := validGitHubID.FindStringSubmatch(v.id)
		owner := "-"
		repo := "-"
		number := 0
		if len(matched) > 4 {
			owner = matched[2]
			repo = matched[3]
			numtxt := matched[5]
			num, _ := strconv.ParseInt(numtxt, 10, 32)
			number = int(num)
		}
		fmt.Printf("[%d] %v : %s %s %d :: %s %s %d\n", k, v.id, owner, repo, number, v.owner, v.repo, v.number)
		// fmt.Printf("[%d] %v : (%d) %v\n", k, v.id, len(matched), matched)
	}

	// Output:
	// [0] resystems-io/task-graph#123 : resystems-io task-graph 123 :: resystems-io task-graph 123
	// [1] resystems-io/this/and/that#123 : resystems-io this/and/that 123 :: resystems-io this/and/that 123
	// [2] resystems-io#123 : - - 0 :: - - 0
	// [3] #123 :   123 ::   123
}

func ExampleRegexp_gitHubIssueParts() {

	check := []Expect{
		{"/resystems-io/task-graph/issues/123", "resystems-io", "task-graph", 123},
		{"/resystems-io/this/and/that/issues/123", "resystems-io", "this/and/that", 123},
		{"resystems-io/this/and/that/issues/123", "-", "-", 0},
	}

	for k, v := range check {
		matched := validGitHubIssue.FindStringSubmatch(v.id)
		owner := "-"
		repo := "-"
		number := 0
		if len(matched) > 4 {
			owner = matched[1]
			repo = matched[2]
			numtxt := matched[4]
			num, _ := strconv.ParseInt(numtxt, 10, 32)
			number = int(num)
		}
		fmt.Printf("[%d] %v : %s %s %d :: %s %s %d\n", k, v.id, owner, repo, number, v.owner, v.repo, v.number)
	}

	// Output:
	// [0] /resystems-io/task-graph/issues/123 : resystems-io task-graph 123 :: resystems-io task-graph 123
	// [1] /resystems-io/this/and/that/issues/123 : resystems-io this/and/that 123 :: resystems-io this/and/that 123
	// [2] resystems-io/this/and/that/issues/123 : - - 0 :: - - 0
}
