package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thaddeusrhatcher/jirate/actions"
	"github.com/thaddeusrhatcher/jirate/processor"
)

var verbose bool
var issueNumber string
var useMarkdown bool

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
		issueId := args[0]
		switch cmd.Parent() {
		case commentCmd:
			proc := processor.NewCommentProcessor(issueId)
			comments, err := proc.Process(actions.List)
			if err != nil {
				fmt.Println("Failed to retrieve comments: ", err)
			}
			if err = proc.Render(comments); err != nil {
				fmt.Println("Failed renderring comments: ", err)
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
	commentCmd.AddCommand(getCmd)
	commentCmd.AddCommand(listCmd)
	commentCmd.AddCommand(addCmd)
	commentCmd.AddCommand(updateCmd)
	commentCmd.AddCommand(deleteCmd)

	issueCmd.AddCommand(getCmd)
	rootCmd.AddCommand(issueCmd)
	rootCmd.AddCommand(commentCmd)
	return rootCmd
}
