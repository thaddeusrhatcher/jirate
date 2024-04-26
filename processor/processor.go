package processor

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/andygrunwald/go-jira"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/thaddeusrhatcher/jirate/editor"
	myJira "github.com/thaddeusrhatcher/jirate/jira"
	"github.com/thaddeusrhatcher/jirate/renderer"
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

type Action string

const (
	ActionGet    Action = "get"
	ActionAdd    Action = "add"
	ActionList   Action = "list"
	ActionDelete Action = "delete"
	ActionUpdate Action = "update"
)

type Processor interface {
	Process() error
	Render() error
}

type issueStyles struct {
	container lipgloss.Style
	status    lipgloss.Style
}

type IssueProcessor struct {
	action      Action
	issueId     string
	mdConverter *md.Converter
	styles      issueStyles
	jiraClient  *myJira.Jira
}

func NewIssueProcessor(action string, issueId string) IssueProcessor {
	jiraClient, err := myJira.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return IssueProcessor{
		action:      Action(action),
		issueId:     issueId,
		jiraClient:  &jiraClient,
		mdConverter: md.NewConverter("", true, &md.Options{LinkStyle: "referenced"}),
		styles: issueStyles{
			container: lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#F44674")),
			status: lipgloss.NewStyle().
				Foreground(lipgloss.Color("")).
				Bold(true),
		},
	}
}

func (p IssueProcessor) Process() ([]*jira.Issue, error) {
	switch p.action {
	case ActionGet:
		issue, err := p.jiraClient.GetIssue(p.issueId)
		if err != nil {
			return nil, err
		}
		return []*jira.Issue{issue}, nil
	}
	return nil, nil
}

func (p IssueProcessor) Render(issues []*jira.Issue) error {
	for _, issue := range issues {
		converter := md.NewConverter("", true, &md.Options{LinkStyle: "referenced"})
		markdown, err := converter.ConvertString(issue.RenderedFields.Description)
		assignee := "Unassigned"
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.EmailAddress
		}
		full := fmt.Sprintf(issuePrefix,
			issue.Key,
			issue.Fields.Summary,
			p.styles.status.Render(issue.Fields.Status.Name),
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

		fmt.Println(p.styles.container.Render(out))

	}
	return nil
}

type commentStyles struct {
	container lipgloss.Style
	status    lipgloss.Style
}

type CommentProcessor struct {
	issueId     string
	commentId   string
	action      Action
	useMarkdown bool
	mdConverter *md.Converter
	styles      commentStyles
	jiraClient  myJira.Jira
}

func NewCommentProcessor(action, issueId string, useMarkdown bool) CommentProcessor {
	jiraClient, err := myJira.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return CommentProcessor{
		action:      Action(action),
		issueId:     issueId,
		useMarkdown: useMarkdown,
		jiraClient:  jiraClient,
		mdConverter: md.NewConverter("", true, &md.Options{LinkStyle: "referenced"}),
		styles: commentStyles{
			container: lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("63")),
		},
	}
}

func (p CommentProcessor) Process(body string) ([]*jira.Comment, error) {
	switch p.action {
	case ActionAdd:
		if p.useMarkdown {
			err := p.AddMarkdown()
			return nil, err
		}
		err := p.AddBasic(body)
		if err != nil {
			return nil, fmt.Errorf("Failed to add comment for issue %s:\n%s", p.issueId, err)
		}
		return nil, nil
	case ActionList:
		comments, err := p.jiraClient.GetComments(p.issueId)
		if err != nil {
			return nil, fmt.Errorf("Failed to retrieve comments for issue %s:\n%s", p.issueId, err)
		}
		return comments, nil
	case ActionDelete:
		fmt.Printf("Deleting comment %s for issue %s.\n", body, p.issueId)
		err := p.jiraClient.DeleteComment(p.issueId, body)
		if err != nil {
			return nil, fmt.Errorf("Failed to delete comment %s in issue %s:\n%s", p.commentId, p.issueId, err)
		}
		return nil, nil
	case ActionUpdate:
		comment, err := p.jiraClient.GetComment(p.issueId, body)
		if err != nil {
			return nil, fmt.Errorf("Failed to get comment %s in issue %s:\n%s", body, p.issueId, err)
		}
		if err = p.UpdateMarkdown(comment); err != nil {
			return nil, fmt.Errorf(
				"Failed to update comment %s in issue %s:\n%s", body, p.issueId, err)
		}
		return nil, nil
	}
	return nil, errors.New("Action un-supported.")
}

func (p CommentProcessor) Render(comments []*jira.Comment) error {
	for _, c := range comments {
		markdown, err := p.mdConverter.ConvertString(c.Body)
		if err != nil {
			fmt.Println("Failed to conert HTML to Markdown.")
			return err
		}

		full := fmt.Sprintf(commentPrefix,
			c.ID, c.Author.EmailAddress, c.Created, markdown)
		out, err := glamour.Render(full, "dark")

		if err != nil {
			fmt.Println("Failed to render markdown with Glamour")
			return err
		}

		fmt.Println(p.styles.container.Render(out))
	}
	return nil
}

func (p CommentProcessor) AddBasic(body string) error {
	err := p.jiraClient.AddComment(p.issueId, body)
	if err != nil {
		fmt.Println("Failed to create comment.")
		return err
	}
	return nil
}

func (p CommentProcessor) AddMarkdown() error {
	program := tea.NewProgram(editor.InitialModel())
	if _, err := program.Run(); err != nil {
		return err
	}
	if editor.Quit || editor.Content == "" {
		fmt.Println("Editor cancelled or contains no content. Exiting...")
		os.Exit(1)
	}
	comment := editor.Content

	buffer := new(bytes.Buffer)
	err := renderer.Render(buffer, []byte(comment))
	if err != nil {
		fmt.Println("Failed to render ADF from content.")
		return err
	}
	err = p.jiraClient.AddCommentCustom(p.issueId, buffer.Bytes())
	if err != nil {
		fmt.Println("Failed to create md comment.")
		panic(err)
	}
	return nil
}

func (p CommentProcessor) UpdateMarkdown(comment *jira.Comment) error {
	markdown, err := p.mdConverter.ConvertString(comment.Body)
	if err != nil {
		return err
	}
	model := editor.InitialModelWithValue(markdown)
	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		panic(err)
	}
	if editor.Quit || editor.Content == "" {
		fmt.Println("Editor cancelled or contains no content. Exiting...")
		os.Exit(1)
	}
	commentBody := editor.Content

	buffer := new(bytes.Buffer)
	err = renderer.Render(buffer, []byte(commentBody))
	if err != nil {
		return fmt.Errorf("Failed to render ADF from content: %v", err)
	}
	err = p.jiraClient.UpdateCommentCustom(p.issueId, comment.ID, buffer.Bytes())
	if err != nil {
		return fmt.Errorf("Failed to create md comment: %v", err)
	}
	return nil
}
