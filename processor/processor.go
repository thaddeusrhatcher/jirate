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
	"github.com/thaddeusrhatcher/jirate/actions"
	"github.com/thaddeusrhatcher/jirate/editor"
	myJira "github.com/thaddeusrhatcher/jirate/jira"
	"github.com/thaddeusrhatcher/jirate/renderer"
	"golang.org/x/net/html"
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

type Processor interface {
	Process() error
	Render() error
}

type issueStyles struct {
	container lipgloss.Style
	status    lipgloss.Style
}

type IssueProcessor struct {
	issueId     string
	mdConverter *md.Converter
	styles      issueStyles
	jiraClient  *myJira.Jira
	status      string
	project     string
}

func NewIssueProcessor(issueId string) IssueProcessor {
	jiraClient, err := myJira.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return IssueProcessor{
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

func NewIssueProcessorWithOptions(options ProcessorOptions) IssueProcessor {
	p := NewIssueProcessor(options.IssueId)
	if options.Status != "" {
		p.status = options.Status
	}
	if options.Project != "" {
		p.project = options.Project
	}
	return p
}

func (p IssueProcessor) Process(action actions.Action) ([]jira.Issue, error) {
	switch action {
	case actions.Get:
		issue, err := p.jiraClient.GetIssue(p.issueId)
		if err != nil {
			return nil, err
		} else if issue == nil {
			return nil, errors.New("Issues was nil")
		}
		return []jira.Issue{*issue}, nil
	case actions.List:
		options := myJira.IssueSearchOptions{
			Status: p.status,
			Project: p.project,
		}
		issues, err := p.jiraClient.GetIssues(options)
		return issues, err
	}
	return nil, nil
}

func (p IssueProcessor) Render(issues []jira.Issue) error {
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

type ProcessorOptions struct {
	Project     string
	UseMarkdown bool
	IssueId     string
	Status      string
	CommentId   string
	CommentBody string
}

type CommentProcessor struct {
	IssueId     string
	CommentId   string
	UseMarkdown bool
	CommentBody string

	mdConverter *md.Converter
	styles      commentStyles
	jiraClient  myJira.Jira
}

func NewCommentProcessor(issueId string) *CommentProcessor {
	jiraClient, err := myJira.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	p := &CommentProcessor{
		IssueId:     issueId,
		UseMarkdown: false,
		jiraClient:  jiraClient,
		mdConverter: md.NewConverter("", true, &md.Options{
			LinkReferenceStyle: "shortcut",
			LinkStyle:          "inlined",
		}),
		styles: commentStyles{
			container: lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("63")),
		},
	}

	return p
}

func NewCommentProcessorWithOptions(issueId string, options ProcessorOptions) *CommentProcessor {
	p := NewCommentProcessor(issueId)
	if options.UseMarkdown {
		p.UseMarkdown = true
	}
	if options.CommentId != "" {
		p.CommentId = options.CommentId
	}
	if options.CommentBody != "" {
		p.CommentBody = options.CommentBody
	}
	return p
}

func (p CommentProcessor) Process(action actions.Action) ([]*jira.Comment, error) {
	switch action {
	case actions.Add:
		if p.UseMarkdown {
			err := p.AddMarkdown()
			return nil, err
		}
		err := p.AddBasic(p.CommentBody)
		if err != nil {
			return nil, fmt.Errorf("Failed to add comment for issue %s:\n%s", p.IssueId, err)
		}
		return nil, nil
	case actions.List:
		comments, err := p.jiraClient.GetComments(p.IssueId)
		if err != nil {
			return nil, fmt.Errorf("Failed to retrieve comments for issue %s:\n%s", p.IssueId, err)
		}
		return comments, nil
	case actions.Delete:
		fmt.Printf("Deleting comment %s for issue %s.\n", p.CommentId, p.IssueId)
		err := p.jiraClient.DeleteComment(p.IssueId, p.CommentId)
		if err != nil {
			return nil, fmt.Errorf("Failed to delete comment %s in issue %s:\n%s", p.CommentId, p.IssueId, err)
		}
		return nil, nil
	case actions.Update:
		comment, err := p.jiraClient.GetComment(p.IssueId, p.CommentId)
		if err != nil {
			return nil, fmt.Errorf("Failed to get comment %s in issue %s:\n%s", p.CommentId, p.IssueId, err)
		}
		if err = p.UpdateMarkdown(comment); err != nil {
			return nil, fmt.Errorf(
				"Failed to update comment %s in issue %s:\n%s", p.CommentBody, p.IssueId, err)
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
	err := p.jiraClient.AddComment(p.IssueId, body)
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
	err = p.jiraClient.AddCommentCustom(p.IssueId, buffer.Bytes())
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
	err = p.jiraClient.UpdateCommentCustom(p.IssueId, comment.ID, buffer.Bytes())
	if err != nil {
		return fmt.Errorf("Failed to create md comment: %v", err)
	}
	return nil
}

func getMaxLengthHtml(node *html.Node, max int, str string) (int, string) {
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, a := range node.Attr {
			if len(a.Val) > max {
				max = len(a.Val)
				str = a.Val
			}
			break
		}
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		max, str = getMaxLengthHtml(c, max, str)
	}
	return max, str
}
