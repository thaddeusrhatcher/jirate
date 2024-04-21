package jira

import (
	"bufio"
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

	fmt.Printf("Response: \n\tstatus: %s\n\tmax results: %d\n\ttotal: %d\n",
		response.Status,
		response.MaxResults,
		response.Total,
	)
	return user, err
}

func (j Jira) GetComments(issueNumber string) ([]*jira.Comment, error) {
	issue, response, err := j.client.Issue.Get(issueNumber, nil)
	if err != nil {
	}
	fmt.Printf("Response: \n\tstatus: %s\n\tmax results: %d\n\ttotal: %d\n",
		response.Status,
		response.MaxResults,
		response.Total,
	)
	comments := issue.Fields.Comments.Comments
	for _, v := range comments {
		fmt.Printf("\tID: %s\n\tAuthor Email: %s\n\tBody: %s\n\n",
			v.ID,
			v.Author.EmailAddress,
			v.Body,
		)
	}
	return comments, err
}

func (j Jira) AddComment(issueNumber, content string) (*jira.Comment, error) {
	comment, response, err := j.client.Issue.AddComment(issueNumber, &jira.Comment{
		Body: content,
	})
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("Failed to create comment: %v", err)
	}
	fmt.Printf("Response: \n\tstatus: %s\n\tmax results: %d\n\ttotal: %d\n",
		response.Status,
		response.MaxResults,
		response.Total,
	)
	return comment, nil
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
	}
	fmt.Printf("Response: \n\tstatus: %s\n\tmax results: %d\n\ttotal: %d\n",
		response.Status,
		response.MaxResults,
		response.Total,
	)
	return nil
}
