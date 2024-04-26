# Jirate

Jirate is a command-line tool for working with Jira Issue comments.
It features an interactive markdown editor for creating and updating comments.

Notes:
> I haven't figured the best way to line-wrap comments/issues containg very long string fields. If the output looks weird/clipped just increase the width of your terminal window. 

> The `renderer` package for converting markdown to ADF was ripped from some random repo I found, so I take no credit for that. The author had a print statement in the `Render()` function that was printing the entire ADF tree which was getting in the way.

# Setup

## Configuration File

First off, you need to create the `config.txt` file that Jirate will use to authenticate to your Jira workspace.

Jirate looks at `$HOME/.config/jirate/config.txt` for this file.

### File Format

The file format is as follows:

```txt
url:{Your Atlassian/Jira Domain}
username:{Your Account Email}
password:{Your Jira API token}
```

Example:

```txt
url:example.atlassian.net
username:giga@chad.com
password:ASDF123
```

### API Token

To generate an API Token: 

1. Log in to https://id.atlassian.com/manage-profile/security/api-tokens.
2. Click Create API token.
3. From the dialog that appears, enter a memorable and concise Label for your token and click Create.
4. Click Copy to clipboard and paste somewhere for safe keeping.

[Reference](https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/)

### Building Jirate

1. Execute `go build`, this will create the `jirate` executable file.
2. Execute `sudo mv jirate /usr/local/bin` to make it globally executable.

Alternatively, run `make build-complete` from the Makefile which just bundles these two commands together.

## Usage

The following are the current commands supported.
* Issues: `get`
* Comments: `add`, `update`, `list`, `delete`

### Issues

#### Retrieve content for a Jira Issue

```sh 
jirate issue get {IssueID}
```

### Comments

> **Note:** Since the `delete` and `update` function from a specific CommentID, I recommend running `list` for the particular Issue to view the comment IDs. Then copy the ID over for the comment you wish to delete/update.

> My goal is to eventually make it completely interactive to enable navigating through comments and performing add/update/delete on them.

#### Add Basic Text Comment to Issue

```sh
jirate comment add {IssueID} {short content}
```

#### Add Comment to Issue via Markdown Editor

```sh
jirate comment add {IssueID} md
```

#### List Comments for Issue By ID

```sh
jirate comment list {IssueID}
```

#### Delete Comment

```sh
jirate comment delete {IssueID} {CommentID}
```

#### Update Comment

```sh
jirate comment update {IssueID} {CommentID}
```
