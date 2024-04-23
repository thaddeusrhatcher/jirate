package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/thaddeusrhatcher/jirate/editor"
	"github.com/thaddeusrhatcher/jirate/jira"
	"github.com/thaddeusrhatcher/jirate/renderer"
)

var jiraClient *jira.Jira

type action string

const (
	AddComment    action = "add"
	ListComments  action = "list"
	DeleteComment action = "delete"
	UpdateComment action = "update"
)

const prefix = `# Comment %s
> Author Email: %v *Created: %s*

%s
`

type Args struct {
	action      action
	issueNumber string
	comment     string
	useMarkdown bool
	commentId   string
}

func parseArgs() (Args, error) {
	rawArgs := os.Args[1:]
	if len(rawArgs) < 2 {
		return Args{}, errors.New("Missing required args")
	}
	a := Args{}
	a.action = action(rawArgs[0])
	a.issueNumber = rawArgs[1]
	switch a.action {
	case AddComment:
		if rawArgs[2] == "md" {
			a.useMarkdown = true
		} else {
			a.comment = strings.Join(rawArgs[2:], " ")
		}
	case UpdateComment, DeleteComment:
		a.commentId = rawArgs[2]
	}
	return a, nil
}

func main() {
	args, err := parseArgs()
	if err != nil {
		fmt.Println("Missing required arguments")
		panic(err)
	}
	jiraClient, err := jira.NewClient()
	if err != nil {
		panic(err)
	}

	switch args.action {
	case AddComment:
		fmt.Printf("Adding comment to story %s\n", args.issueNumber)
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
			err = jiraClient.AddCommentCustom(args.issueNumber, buffer.Bytes())
			if err != nil {
				fmt.Println("Failed to create md comment.")
				panic(err)
			}
		} else {
			err = jiraClient.AddComment(args.issueNumber, args.comment)
			if err != nil {
				fmt.Println("Failed to create comment.")
				panic(err)
			}
		}
		fmt.Printf("Success!\n")
	case ListComments:
		comments, err := jiraClient.GetComments(args.issueNumber)
		if err != nil {
			fmt.Println("Failed to retrieve comments")
			panic(err)
		}
		converter := md.NewConverter("", true, nil)
		for _, c := range comments {
			markdown, err := converter.ConvertString(c.Body)

			if err != nil {
				fmt.Println("Failed to conert HTML to Markdown.")
				panic(err)
			}

			full := fmt.Sprintf(prefix,
				c.ID, c.Author.EmailAddress, c.Created, markdown)
			out, err := glamour.Render(full, "dark")

			if err != nil {
				fmt.Println("Failed to render markdown with Glamour")
				panic(err)
			}

			fmt.Print(out)
		}
	case DeleteComment:
		fmt.Printf("Deleting comment %s for issue %s.\n", args.commentId, args.issueNumber)
		err := jiraClient.DeleteComment(args.issueNumber, args.commentId)
		if err != nil {
			fmt.Printf(
				"Failed to delete comment %s in issue %s\n",
				args.commentId,
				args.issueNumber,
			)
			panic(err)
		}
		fmt.Println("Success!")
	case UpdateComment:
		comment, err := jiraClient.GetComment(args.issueNumber, args.commentId)
		if err != nil {
			fmt.Printf(
				"Failed to get comment %s in issue %s\n",
				args.commentId,
				args.issueNumber,
			)
			panic(err)
		}

		converter := md.NewConverter("", true, nil)
		markdown, err := converter.ConvertString(comment.Body)
		full := fmt.Sprintf(prefix,
			comment.ID, comment.Author.EmailAddress, comment.Created, markdown)
		out, err := glamour.Render(full, "dark")

		if err != nil {
			fmt.Println("Failed to render markdown with Glamour")
			panic(err)
		}

		fmt.Print(out)

		if err != nil {
			panic(err)
		}
		model := editor.InitialModelWithValue(markdown)
		p := tea.NewProgram(model)
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
		err = jiraClient.UpdateCommentCustom(args.issueNumber, args.commentId, buffer.Bytes())
		if err != nil {
			fmt.Println("Failed to create md comment.")
			panic(err)
		}
		fmt.Println("Success!")
	}
}

func listMyIssues(jiraClient jira.Jira) {
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
	fmt.Println("Listing comments for issue ", issueNumber)
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
