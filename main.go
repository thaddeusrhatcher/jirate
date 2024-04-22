package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thaddeusrhatcher/jirate/editor"
	"github.com/thaddeusrhatcher/jirate/jira"
	"github.com/thaddeusrhatcher/jirate/renderer"
)

var jiraClient *jira.Jira

type action string

const (
	AddComment action = "add"
	GetIssues  action = "get"
)

type Args struct {
	action      action
	issueNumber string
	comment     string
	useMarkdown bool
}

func parseArgs() (Args, error) {
	rawArgs := os.Args[1:]
	if len(rawArgs) < 3 {
		return Args{}, errors.New("Missing required args")
	}
	a := Args{}
	a.action = action(rawArgs[0])
	a.issueNumber = rawArgs[1]
	if rawArgs[2] == "md" {
		a.useMarkdown = true
	} else {
		a.comment = strings.Join(rawArgs[2:], " ")
	}
	return a, nil
}

func main() {
	args, err := parseArgs()
	if err != nil {
		panic(err)
	}
	jiraClient, err := jira.NewClient()
	if err != nil {
		panic(err)
	}

	switch args.action {
	case AddComment:
		if args.useMarkdown {
			p := tea.NewProgram(editor.InitialModel())
			if _, err := p.Run(); err != nil {
				panic(err)
			}
			if editor.Quit || editor.Content == "" {
				fmt.Println("Editor cancelled or contains no content. Exiting...")
				os.Exit(1)
			}
			args.comment = editor.Content

			buffer := new(bytes.Buffer)
			err = renderer.Render(buffer, []byte(args.comment))
			if err != nil {
				fmt.Println("Failed to render ADF from content.")
				panic(err)
			}
			fmt.Printf("Adding comment to story %s", args.issueNumber)
			err = jiraClient.AddCommentCustom(args.issueNumber, buffer.Bytes())
			if err != nil {
				fmt.Println("Failed to create md comment.")
				panic(err)
			}
		} else {
			err = jiraClient.AddComment(args.issueNumber, []byte(args.comment))
			if err != nil {
				fmt.Println("Failed to create comment.")
				panic(err)
			}
		}
	}

}

func listMyIssues() {
	fmt.Println("Getting your In Progress issues")
	issues, err := jiraClient.GetMyIssues()
	if err != nil {
		fmt.Println("failed to get your issues")
		panic(err)
	}

	for _, issue := range issues {
		fmt.Printf("Issue ID %s\n\tSummary: %s\n\tKey: %s\n\tAssignee: %s\n\tcomments: %v\n",
			issue.ID,
			issue.Fields.Summary,
			issue.Key,
			issue.Fields.Assignee.EmailAddress,
			issue.Fields.Comments,
		)
	}
}

func listIssueComments(issueNumber string) {
	comments, err := jiraClient.GetComments(issueNumber)
	if err != nil {
		fmt.Println("failed to get comments")
		panic(err)
	}
	for _, c := range comments {
		fmt.Printf("Comment: \n\tID: %s\n\tName: %s\n\tAuthor Email %v\n\tCreated: %s\n\tBody: %s\n",
			c.ID,
			c.Name,
			c.Author.EmailAddress,
			c.Created,
			c.Body,
		)
	}
}
