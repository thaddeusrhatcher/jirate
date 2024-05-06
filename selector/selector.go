package selector

import (
	"github.com/andygrunwald/go-jira"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

var SelectedTransition Transition
var Quit bool

type Model struct {
	form *huh.Form
	err  error
}

type Transition struct {
	Id string
	Name string
}

func getOptions(transitions *[]jira.Transition) []huh.Option[Transition] {
	var options []huh.Option[Transition]
	for _, t := range *transitions {
		transition := Transition{
			Id: t.ID,
			Name: t.Name,
		}
		options = append(options, huh.NewOption[Transition](t.Name, transition))
	}
	return options
}

func InitialModel(transitions *[]jira.Transition) Model {
	options := getOptions(transitions)
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[Transition]().
			Title("Choose status transition").
			Description("Transitions represent the status this story can be moved to.").
			Key("transition").
			Options(options...),
		),
	)
	return Model{
		form: f,
		err:  nil,
	}
}

// Could modify Init() to fetch transitions on IssueId passed in to InitialModel()
func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}
	if m.form.State == huh.StateCompleted {
		transition := m.form.Get("transition")
		SelectedTransition = transition.(Transition)
		return m, tea.Quit
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type.String() {
		case "esc", "ctrl+c":
			Quit = true
			return m, tea.Quit
		}
	}
	return m, cmd 
}

func (m Model) View() string {
	return m.form.View()
}
