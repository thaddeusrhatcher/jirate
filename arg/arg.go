package arg

import (
	"errors"
	"os"
	"strings"
)

type action string
type object string

const (
	ActionGet    action = "get"
	ActionAdd    action = "add"
	ActionList   action = "list"
	ActionDelete action = "delete"
	ActionUpdate action = "update"

	ObjectComment object = "comment"
	ObjectIssue   object = "issue"
)

type Args struct {
	Action      action
	Object      object
	IssueNumber string
	Comment     string
	UseMarkdown bool
	CommentId   string
}

func ParseArgs() (Args, error) {
	rawArgs := os.Args[1:]
	if len(rawArgs) < 3 {
		return Args{}, errors.New("Missing required args or command not supported.")
	}
	a := Args{}
	a.Object = object(rawArgs[0])
	a.Action = action(rawArgs[1])
	a.IssueNumber = rawArgs[2]
	switch a.Object {
	case ObjectComment:
		switch a.Action {
		case ActionAdd:
			if rawArgs[3] == "md" {
				a.UseMarkdown = true
			} else {
				a.Comment = strings.Join(rawArgs[2:], " ")
			}
		case ActionUpdate, ActionDelete:
			a.CommentId = rawArgs[3]
		}
	case ObjectIssue:
		switch a.Action {
		case ActionGet:
			if len(rawArgs) > 3 {
				return Args{}, errors.New("Bad command. Goodbye.")
			}
		default:
			return Args{}, errors.New("Command unsupported for Issues.")
		}
	default:
		return Args{}, errors.New("Command unsupported. Goodbye.")
	}

	return a, nil
}
