package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thaddeusrhatcher/jirate/actions"
	"github.com/thaddeusrhatcher/jirate/processor"
)

var verbose bool
var issueNumber string
var useMarkdown bool
var status string

var rootCmd = &cobra.Command{
	Use: "jirate",
}

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Commands for managing Jira issues. Currently only supported 'get' for retrieving an issue.",
	Long:  ``,
}

var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Commands for managing Jira comments.",
	Long:  ``,
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Retrieve the specified object from Jira",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueId := args[0]
		switch cmd.Parent() {
		case issueCmd:
			proc := processor.NewIssueProcessor(issueId)
			issues, err := proc.Process(actions.Get)
			if err != nil {
				panic(err)
			}
			err = proc.Render(issues)
			if err != nil {
				fmt.Println(err)
			}
		default:
			fmt.Println("Command unsupported.")
		}

	},
}

var addCmd = &cobra.Command{
	Use:  "add",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		issueId := args[0]
		useMarkdown := false
		var body string
		if args[1] == "md" {
			useMarkdown = true
		} else {
			body = strings.Join(args[1:], " ")
		}
		switch cmd.Parent() {
		case commentCmd:
			proc := processor.NewCommentProcessorWithOptions(issueId, processor.ProcessorOptions{
				UseMarkdown: useMarkdown,
				CommentBody: body,
			})
			_, err := proc.Process(actions.Add)
			if err != nil {
				panic(err)
			}
			fmt.Println("Success!")
		default:
			fmt.Println("Command unsupported")
		}
	},
}

var listCmd = &cobra.Command{
	Use: "list",
	Run: func(cmd *cobra.Command, args []string) {
		status, errS := cmd.Flags().GetString("status")
		project, errP := cmd.Flags().GetString("project")
		err := errors.Join(errS, errP)
		if err != nil {
			fmt.Println("Failed parsing flags: ", err.Error())
			return
		}
		switch cmd.Parent() {
		case commentCmd:
			issueId := args[0]
			proc := processor.NewCommentProcessor(issueId)
			comments, err := proc.Process(actions.List)
			if err != nil {
				fmt.Println("Failed to retrieve comments: ", err)
			}
			if err = proc.Render(comments); err != nil {
				fmt.Println("Failed renderring comments: ", err)
			}
		case issueCmd:
			if project == "" {
				fmt.Println("Bad command: Project must be provided.")
				return
			}
			proc := processor.NewIssueProcessorWithOptions(
				processor.ProcessorOptions{
					Status:  status,
					Project: project,
				},
			)
			issues, err := proc.Process(actions.List)
			if err != nil {
				fmt.Println("Failed to retrieve issues: ", err)
			}
			if err = proc.RenderShort(issues); err != nil {
				fmt.Println("Failed rendering issues: ", err)
			}
		default:
			fmt.Println("Command unsupported")
		}
	},
}

var deleteCmd = &cobra.Command{
	Use: "delete",
	Run: func(cmd *cobra.Command, args []string) {
		issueId := args[0]
		commentId := args[1]
		switch cmd.Parent() {
		case commentCmd:
			proc := processor.NewCommentProcessorWithOptions(
				issueId,
				processor.ProcessorOptions{
					CommentId: commentId,
				})
			_, err := proc.Process(actions.Delete)
			if err != nil {
				fmt.Println("Failed to delete comments: ", err)
				return
			}
			fmt.Println("Success!")
		default:
			fmt.Println("Command unsupported")
		}
	},
}

var updateCmd = &cobra.Command{
	Use: "update",
	Run: func(cmd *cobra.Command, args []string) {
		issueId := args[0]
		switch cmd.Parent() {
		case commentCmd:
			proc := processor.NewCommentProcessorWithOptions(
				issueId,
				processor.ProcessorOptions{
					UseMarkdown: useMarkdown,
				},
			)
			_, err := proc.Process(actions.Update)
			if err != nil {
				panic(err)
			}
			fmt.Println("Success!")
		default:
			fmt.Println("Command unsupported")
		}
	},
}

func NewRoot() *cobra.Command {
	addCmd.Flags().Bool("md", false, "Whether to use markdown editor")
	listCmd.PersistentFlags().StringP("status", "S", "In Progress", "")
	listCmd.PersistentFlags().StringP("assignee", "A", "", "")
	listCmd.PersistentFlags().StringP("project", "P", "", "")
	listCommentCmd := *listCmd
	listIssueCmd := *listCmd
	commentCmd.AddCommand(getCmd)
	commentCmd.AddCommand(&listCommentCmd)
	commentCmd.AddCommand(addCmd)
	commentCmd.AddCommand(updateCmd)
	commentCmd.AddCommand(deleteCmd)

	issueCmd.AddCommand(getCmd)
	issueCmd.AddCommand(&listIssueCmd)
	rootCmd.AddCommand(issueCmd)
	rootCmd.AddCommand(commentCmd)
	return rootCmd
}
