package issueview

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-github/v61/github"
	"github.com/muesli/termenv"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/six78/2-story-points-cli/internal/config"
	"github.com/six78/2-story-points-cli/internal/view/messages"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

var (
	errorStyle = lipgloss.NewStyle().Foreground(config.ForegroundShadeColor)
	//defaultStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
)

type issueFetchedMessage struct {
	url  string
	info *issueInfo
}

type Model struct {
	issue  *protocol.Issue
	issues map[string]*issueInfo

	client  *github.Client
	spinner spinner.Model
}

type issueInfo struct {
	err    error
	title  *string
	labels []labelInfo
}

type labelInfo struct {
	name  *string
	style lipgloss.Style
}

func New() Model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot

	return Model{
		issue:   nil,
		issues:  make(map[string]*issueInfo),
		client:  github.NewClient(nil),
		spinner: s,
	}
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case messages.GameStateMessage:
		if msg.State == nil {
			m.issue = nil
			break
		}
		if msg.State.Issues == nil {
			m.issue = nil
			break
		}
		m.issue = msg.State.Issues.Get(msg.State.ActiveIssue)
		if m.issue == nil {
			break
		}
		_, ok := m.issues[m.issue.TitleOrURL]
		if ok {
			break
		}
		cmd = fetchIssue(m.client, m.issue)
		cmds = append(cmds, cmd)
		m.issues[m.issue.TitleOrURL] = nil

	case issueFetchedMessage:
		config.Logger.Debug("<<< issue fetched",
			zap.Any("msg", msg),
		)
		m.issues[msg.url] = msg.info
	}

	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	rightColumn := lipgloss.JoinVertical(lipgloss.Top,
		m.renderRow1(),
		m.renderInfo())
	return lipgloss.JoinHorizontal(lipgloss.Left,
		"Issue:  \n\n", // Fill at least 3 lines
		rightColumn)
}

func (m *Model) renderRow1() string {
	if m.issue != nil {
		return m.issue.TitleOrURL
	}
	return "-"
}

func (m *Model) renderInfo() string {
	if m.issue == nil {
		return ""
	}

	info, ok := m.issues[m.issue.TitleOrURL]
	if !ok {
		return ""
	}

	if info == nil {
		return errorStyle.Render(m.spinner.View() + " fetching title")
	}

	if info.err != nil {
		return errorStyle.Render(fmt.Sprintf("[%s]", info.err.Error()))
	}

	row1 := ""
	if info.title == nil {
		row1 = errorStyle.Render("[empty issue title]")
	} else {
		row1 = *info.title
	}

	var labels []string
	for _, l := range info.labels {
		if l.name == nil {
			continue
		}
		labelName := fmt.Sprintf("[%s]", *l.name)
		labels = append(labels, l.style.Render(labelName))
	}
	row2 := strings.Join(labels, " ")

	return lipgloss.JoinVertical(lipgloss.Top, row1, row2)
}

type githubIssueRequest struct {
	owner  string
	repo   string
	number int
}

func parseUrl(input string) (*githubIssueRequest, error) {
	u, err := url.Parse(input)
	if err != nil {
		return nil, nil
	}
	if u.Host != "github.com" {
		return nil, errors.New("only github links are unfurled")
	}
	path := strings.Split(u.Path, "/")
	if len(path) != 5 {
		return nil, errors.New("invalid github issue link")
	}

	issueNumber, err := strconv.Atoi(path[4])
	if err != nil {
		return nil, errors.New("invalid github issue number")
	}

	return &githubIssueRequest{
		owner:  path[1],
		repo:   path[2],
		number: issueNumber,
	}, nil
}

func fetchIssue(client *github.Client, input *protocol.Issue) tea.Cmd {
	return func() tea.Msg {
		if input == nil {
			return nil
		}
		request, err := parseUrl(input.TitleOrURL)
		if err != nil {
			return issueFetchedMessage{
				url: input.TitleOrURL,
				info: &issueInfo{
					err: err,
				},
			}
		}
		if request == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		issue, _, err := client.Issues.Get(ctx, request.owner, request.repo, request.number)
		if err != nil {
			return issueFetchedMessage{
				url: input.TitleOrURL,
				info: &issueInfo{
					err: errors.New("failed to fetch github issue"),
				},
			}
		}

		labels := make([]labelInfo, len(issue.Labels))
		for i, label := range issue.Labels {
			labels[i].name = label.Name
			labels[i].style = labelStyle(label.Color)
		}

		return issueFetchedMessage{
			url: input.TitleOrURL,
			info: &issueInfo{
				err:    nil,
				title:  issue.Title,
				labels: labels,
			},
		}
	}
}

func labelStyle(input *string) lipgloss.Style {
	if input == nil {
		return lipgloss.NewStyle().Foreground(config.ForegroundShadeColor)
	}

	color := lipgloss.Color("#" + *input)
	dark := colorIsDark(color)

	if lipgloss.DefaultRenderer().HasDarkBackground() == dark {
		return lipgloss.NewStyle().Background(color)
	}

	return lipgloss.NewStyle().Foreground(color)
}

func colorIsDark(color lipgloss.Color) bool {
	renderer := lipgloss.DefaultRenderer()
	c := renderer.ColorProfile().Color(string(color))
	rgb := termenv.ConvertToRGB(c)
	//_, _, lightness := rgb.Hsl()
	perceivedLightness := 0.2126*rgb.R + 0.7152*rgb.G + 0.0722*rgb.B
	return perceivedLightness < 0.453
}
