package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/andygrunwald/go-jira"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/thaddeusrhatcher/jirate/editor"
	j "github.com/thaddeusrhatcher/jirate/jira"
	"github.com/thaddeusrhatcher/jirate/renderer"
)

var jiraClient *j.Jira

type action string

const (
	GetIssue      action = "issue"
	AddComment    action = "add"
	ListComments  action = "list"
	DeleteComment action = "delete"
	UpdateComment action = "update"
)

const commentPrefix = `# Comment %s
> Author Email: %v *Created: %s*

%s
`
const issuePrefix = `# Issue %s
Summary: %s

**Status: %s**

> Author Email: %v
> Assignee Email: %v
> *Created: %s* *Updated: %s*

%s
`

const (
	purple    = lipgloss.Color("99")
	gray      = lipgloss.Color("245")
	lightGray = lipgloss.Color("241")
)

var commentStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63"))

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
	jiraClient, err := j.NewClient()
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

			full := fmt.Sprintf(commentPrefix,
				c.ID, c.Author.EmailAddress, c.Created, markdown)
			out, err := glamour.Render(full, "dark")

			if err != nil {
				fmt.Println("Failed to render markdown with Glamour")
				panic(err)
			}

			fmt.Println(commentStyle.Render(out))
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
		full := fmt.Sprintf(commentPrefix,
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
	case GetIssue:
		issue, err := jiraClient.GetIssue(args.issueNumber)

		if err != nil {
			panic(err)
		}
		RenderIssue(issue)
	}
}

func RenderIssue(issue *jira.Issue) {
	converter := md.NewConverter("", true, &md.Options{LinkStyle: "referenced"})
	markdown, err := converter.ConvertString(issue.RenderedFields.Description)
	assignee := "Unassigned"
	if issue.Fields.Assignee != nil {
		assignee = issue.Fields.Assignee.EmailAddress
	}
	full := fmt.Sprintf(issuePrefix,
		issue.Key,
		issue.Fields.Summary,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#26E092")).Bold(true).Render(issue.Fields.Status.Name),
		issue.Fields.Creator.EmailAddress,
		assignee,
		issue.RenderedFields.Created,
		issue.RenderedFields.Updated,
		markdown,
	)
	out, err := glamour.Render(full, "dark")

	if err != nil {
		fmt.Println("Failed to render markdown with Glamour")
		panic(err)
	}

	fmt.Println(commentStyle.Render(out))

}

func listMyIssues(jiraClient j.Jira) {
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
