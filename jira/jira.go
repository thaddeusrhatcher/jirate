package jira

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/thaddeusrhatcher/jirate/config"
)

type Config struct {
	Auth jira.BasicAuthTransport
	Url  string
}

type Jira struct {
	client *jira.Client
	Config Config
}

func (c *Config) loadConfig() error {
	file, err := config.GetConfigFile()

	if err != nil {
		return errors.New(`Failed to open the config file.
			Please verify the following exists: $HOME/.config/jirate/config.txt.
		`)
	}

	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)
	vals := make(map[string]string)

	for fileScanner.Scan() {
		line := fileScanner.Text()
		l := strings.Split(line, ":")
		vals[l[0]] = l[1]
	}

	username, ok := vals["username"]
	if !ok {
		return errors.New("Missing username in config.txt")
	}
	password, ok := vals["password"]
	if !ok {
		return errors.New("Missing password in config.txt")
	}
	url, ok := vals["url"]
	if !ok {
		return errors.New("Missing url in config.txt")
	}
	c.Auth = jira.BasicAuthTransport{
		Username: username,
		Password: password,
	}
	c.Url = "https://" + url
	return nil
}

func NewClient() (Jira, error) {
	config := Config{}
	err := config.loadConfig()
	if err != nil {
		return Jira{}, err
	}
	var j Jira
	j.client, err = jira.NewClient(config.Auth.Client(), config.Url)
	if err != nil {
		return Jira{}, nil
	}
	return j, nil
}

func (j Jira) GetIssue(issueNumber string) (*jira.Issue, error) {
	issue, _, err := j.client.Issue.Get(issueNumber, &jira.GetQueryOptions{
		Expand: "renderedFields",
	})

	if err != nil {
		return nil, nil
	}

	return issue, nil
}

func (j Jira) GetMyIssues(status string) ([]jira.Issue, error) {
	user, err := j.GetMyAccount()
	if err != nil {
		return []jira.Issue{}, err
	}
	jql := fmt.Sprintf("assignee=%s&status=\"%s\"", user.AccountID, status)
	issues, _, err := j.client.Issue.Search(jql, &jira.SearchOptions{
		Expand: "renderedFields",
	})
	if err != nil {
		return []jira.Issue{}, err
	}
	return issues, nil
}

type IssueSearchOptions struct {
	Status   string
	Assignee string
	Sprint   string
	Project  string
}

func (j Jira) GetIssues(options IssueSearchOptions) ([]jira.Issue, error) {
	// Get Project
	project, _, err := j.client.Project.Get(options.Project)
	if err != nil {
		return []jira.Issue{}, nil
	}
	// Use Project to Get Active Sprint

	// Get Issues matching options.Status for Active Sprint

	boards, _, err := j.client.Board.GetAllBoards(&jira.BoardListOptions{
		ProjectKeyOrID: project.ID,
	})
	if err != nil {
		return []jira.Issue{}, nil
	}

	if boards == nil || len(boards.Values) == 0 {
		return []jira.Issue{}, errors.New("No boards found")
	}

	board := boards.Values[0]
	sprint, err := j.GetActiveSprintForBoard(board.ID)
	if err != nil {
		return []jira.Issue{}, nil
	}

	issues, err := j.GetIssuesForSprint(sprint.ID, options.Status)

	return issues, err
}

func (j Jira) GetIssuesForSprint(sprintId int, status string) ([]jira.Issue, error) {
	path := fmt.Sprintf("/rest/agile/1.0/sprint/%d/issue", sprintId)
	request, err := j.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	query := request.URL.Query()
	query.Add("jql", fmt.Sprintf("status=\"%s\"", status))
	query.Add("expand", "renderedFields")
	request.URL.RawQuery = query.Encode()
	issues := new(jira.IssuesInSprintResult)
	response, err := j.client.Do(request, issues)
	if err != nil {
		return nil, jira.NewJiraError(response, err)
	}
	if issues == nil {
		return nil, errors.New("Issues nil for sprint: ")
	}
	return issues.Issues, nil
}

type SprintResult struct {
	Values []*jira.Sprint `json:"values"`
}

func (j Jira) GetActiveSprintForBoard(boardId int) (*jira.Sprint, error) {
	path := fmt.Sprintf("/rest/agile/1.0/board/%d/sprint", boardId)
	request, err := j.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	query := request.URL.Query()
	query.Add("state", "active")
	query.Add("maxResults", "1")
	request.URL.RawQuery = query.Encode()
	sprintResult := new(SprintResult)
	_, err = j.client.Do(request, sprintResult)
	if err != nil || len(sprintResult.Values) == 0 {
		return nil, errors.New("No sprint values found.")
	}

	return sprintResult.Values[0], err
}

func (j Jira) GetMyAccount() (*jira.User, error) {
	user, response, err := j.client.User.GetSelf()
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to get user info. Response: \n\tstatus: %s\n\tmax results: %d\n\ttotal: %d\n",
			response.Status,
			response.MaxResults,
			response.Total,
		)
	}

	return user, err
}

func (j Jira) GetComments(issueNumber string) ([]*jira.Comment, error) {
	issue, _, err := j.client.Issue.Get(issueNumber, &jira.GetQueryOptions{
		Expand: "renderedFields",
	})
	if err != nil {
		return nil, err
	}
	comments := issue.RenderedFields.Comments.Comments
	return comments, err
}

func (j Jira) GetComment(issueNumber, commentId string) (*jira.Comment, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/comment/%s", issueNumber, commentId)
	request, err := j.client.NewRequest(
		"GET",
		path,
		nil,
	)
	if err != nil {
		return nil, err
	}
	query := request.URL.Query()
	query.Add("expand", "renderedBody")
	request.URL.RawQuery = query.Encode()

	rawComment := make(map[string]any)
	_, err = j.client.Do(request, &rawComment)
	if err != nil {
		return nil, err
	}
	delete(rawComment, "body")
	comment := new(jira.Comment)
	b, err := json.Marshal(rawComment)
	err = json.Unmarshal(b, comment)
	comment.Body = rawComment["renderedBody"].(string)
	return comment, nil
}

func (j Jira) AddComment(issueNumber, content string) error {
	_, response, err := j.client.Issue.AddComment(issueNumber, &jira.Comment{
		Body: content,
	})
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("Failed to create comment: %v", err)
	} else if response.StatusCode != 201 {
		return fmt.Errorf(
			"Response: \n\tstatus: %s\n\tmax results: %d\n\ttotal: %d\n",
			response.Status,
			response.MaxResults,
			response.Total,
		)
	}
	return nil
}

func (j Jira) AddCommentCustom(issueNumber string, content []byte) error {
	data := make(map[string]interface{})
	err := json.Unmarshal(content, &data)
	if err != nil {
		panic(err)
	}
	body := map[string]interface{}{
		"body": data,
	}

	path := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueNumber)
	request, err := j.client.NewRequest(
		"POST",
		path,
		body,
	)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	if err != nil {
		panic(err)
	}
	response, err := j.client.Do(request, nil)
	if err != nil {
		return err
	} else if response.StatusCode != 201 {
		return fmt.Errorf(
			"Response: \n\tstatus: %s\n\tmax results: %d\n\ttotal: %d\n",
			response.Status,
			response.MaxResults,
			response.Total,
		)
	}
	return nil
}

func (j Jira) UpdateCommentCustom(issueNumber, commentId string, content []byte) error {
	data := make(map[string]interface{})
	err := json.Unmarshal(content, &data)
	if err != nil {
		panic(err)
	}
	body := map[string]interface{}{
		"body": data,
	}

	path := fmt.Sprintf("/rest/api/3/issue/%s/comment/%s", issueNumber, commentId)
	request, err := j.client.NewRequest(
		"PUT",
		path,
		body,
	)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	if err != nil {
		panic(err)
	}
	response, err := j.client.Do(request, nil)
	if err != nil {
		return err
	} else if response.StatusCode != 200 {
		return fmt.Errorf(
			"Response: \n\tstatus: %s\n\tmax results: %d\n\ttotal: %d\n",
			response.Status,
			response.MaxResults,
			response.Total,
		)
	}
	return nil
}

func (j Jira) DeleteComment(issueNumber, commentId string) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s/comment/%s", issueNumber, commentId)
	request, err := j.client.NewRequest(
		"DELETE",
		path,
		nil,
	)
	if err != nil {
		return err
	}

	response, err := j.client.Do(request, nil)
	if err != nil {
		fmt.Println(err)
		return err
	} else if response.StatusCode != 204 {
		return errors.New("Failed to delete comment. Expected status 204.")
	}
	return nil
}
