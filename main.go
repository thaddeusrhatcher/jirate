package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/thaddeusrhatcher/jirate/jira"
)

var jiraClient *jira.Jira

type Args struct {
	issueNumber string
	comment     []string
}

func main() {
	fmt.Println("heloo")
	rawArgs := os.Args[1:]
	args := Args{
		issueNumber: rawArgs[0],
		comment:     rawArgs[1:],
	}
	fmt.Printf("Input args: \n\tissue number: %s\n\tcomment: %s\n",
		args.issueNumber,
		args.comment,
	)

	jiraClient, err := jira.NewClient()
	if err != nil {
		panic(err)
	}

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

	comments, err := jiraClient.GetComments(args.issueNumber)
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

	c, err := jiraClient.AddComment(args.issueNumber, strings.Join(args.comment, " "))
	if err != nil {
		fmt.Println("failed to create comment")
		panic(err)
	}
	fmt.Printf("Comment: \n\tID: %s\n\tName: %s\n\tAuthor Email %v\n\tCreated: %s\n\tBody: %s\n",
		c.ID,
		c.Name,
		c.Author.EmailAddress,
		c.Created,
		c.Body,
	)
}
