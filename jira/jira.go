package jira

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	file, err := os.Open(config.CONFIG_PATH)
	if err != nil {
		return err
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
		url = config.DEFAULT_DOMAIN
		fmt.Println("url not set in config.txt, defaulting to ", url)
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

func (j Jira) GetMyIssues() ([]jira.Issue, error) {
	user, err := j.GetMyAccount()
	if err != nil {
		return []jira.Issue{}, err
	}
	fmt.Println("user: ", user.AccountID)
	jql := fmt.Sprintf("assignee=%s&status=\"In Progress\"", user.AccountID)
	issues, response, err := j.client.Issue.Search(jql, &jira.SearchOptions{})
	if err != nil {
		return []jira.Issue{}, err
	}
	fmt.Printf("Response: \n\tstatus: %s\n\tmax results: %d\n\ttotal: %d\n",
		response.Status,
		response.MaxResults,
		response.Total,
	)

	return issues, nil
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
	comment := new(jira.Comment)
	_, err = j.client.Do(request, comment)
	if err != nil {
		return nil, err
	}
	return comment, nil
}

func (j Jira) AddComment(issueNumber string, content []byte) error {
	data := make(map[string]interface{})
	err := json.Unmarshal(content, &data)
	if err != nil {
		panic(err)
	}
	body := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": data,
	}
	b, err := json.Marshal(body)
	_, response, err := j.client.Issue.AddComment(issueNumber, &jira.Comment{
		Body: string(b),
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

func (j Jira) DeleteComment(issueNumber, commentId string) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s/comment/%s", issueNumber, commentId)
	fmt.Println("path: ", path)
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
