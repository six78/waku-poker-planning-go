package view

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/six78/2-story-points-cli/internal/config"
	"github.com/six78/2-story-points-cli/internal/transport"
	"github.com/six78/2-story-points-cli/internal/view/commands"
	"github.com/six78/2-story-points-cli/internal/view/components/deckview"
	"github.com/six78/2-story-points-cli/internal/view/components/errorview"
	"github.com/six78/2-story-points-cli/internal/view/components/eventhandler"
	"github.com/six78/2-story-points-cli/internal/view/components/issuesview"
	"github.com/six78/2-story-points-cli/internal/view/components/issueview"
	"github.com/six78/2-story-points-cli/internal/view/components/playersview"
	"github.com/six78/2-story-points-cli/internal/view/components/shortcutsview"
	"github.com/six78/2-story-points-cli/internal/view/components/userinput"
	"github.com/six78/2-story-points-cli/internal/view/components/wakustatusview"
	"github.com/six78/2-story-points-cli/internal/view/messages"
	"github.com/six78/2-story-points-cli/internal/view/states"
	"github.com/six78/2-story-points-cli/internal/view/update"
	"github.com/six78/2-story-points-cli/pkg/game"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

type model struct {
	game      *game.Game
	transport transport.Service

	// Actual nextState that will be rendered in components.
	// This is filled from app during Update stage.
	state            states.AppState
	fatalError       error
	gameState        *protocol.State
	roomID           protocol.RoomID
	connectionStatus transport.ConnectionStatus

	// UI components state
	commandMode           bool
	roomViewState         states.RoomView
	errorView             errorview.Model
	playersView           playersview.Model
	shortcutsView         shortcutsview.Model
	wakuStatusView        wakustatusview.Model
	deckView              deckview.Model
	issueView             issueview.Model
	issuesListView        issuesview.Model
	gameEventHandler      eventhandler.Model[*protocol.State, messages.GameStateMessage]
	transportEventHandler eventhandler.Model[transport.ConnectionStatus, messages.ConnectionStatus]

	// Workaround: Used to allow pasting multiline text (list of issues)
	disableEnterKey     bool
	disableEnterRestart chan struct{}

	// Components to be rendered
	// This is filled from actual nextState during View stage.
	input   userinput.Model
	spinner spinner.Model
}

func initialModel(game *game.Game, transport transport.Service) model {
	const initialRoomViewState = states.ActiveIssueView
	deckView := deckview.New()
	deckView.Focus()

	return model{
		game:      game,
		transport: transport,
		// Initial model values
		state:     states.Initializing,
		gameState: nil,
		roomID:    protocol.RoomID{},
		// UI components state
		commandMode:   false,
		roomViewState: initialRoomViewState,
		// View components
		input:          userinput.New(false),
		spinner:        createSpinner(),
		errorView:      errorview.New(),
		playersView:    playersview.New(),
		shortcutsView:  shortcutsview.New(),
		wakuStatusView: wakustatusview.New(),
		deckView:       deckView,
		issueView:      issueview.New(),
		issuesListView: issuesview.New(),
		// Other
		disableEnterKey:     false,
		disableEnterRestart: nil,
	}
}

func createSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	return s
}

func (m model) Init() tea.Cmd {
	m.deckView.Focus()
	return tea.Batch(
		m.input.Init(),
		m.spinner.Tick,
		m.errorView.Init(),
		m.playersView.Init(),
		m.shortcutsView.Init(),
		m.wakuStatusView.Init(),
		m.deckView.Init(),
		m.issueView.Init(),
		m.issuesListView.Init(),
		commands.InitializeApp(m.game, m.transport),
	)
}

// TODO: Rendering could be cached inside components in most cases.
//		 This will require to only call Update when needed (because the object is copied).
//		 On Update() we would set `cache.valid = false`
//		 and then in View() the string could be cached and set `cache.valid = true`.

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := update.NewUpdateCommands()

	switchToState := func(state states.AppState) {
		m.state = state
		cmds.AppendMessage(messages.AppStateMessage{State: state})
	}

	switch msg := msg.(type) {
	case messages.FatalErrorMessage:
		m.fatalError = msg.Err

	case messages.AppStateFinishedMessage:
		switch msg.State {
		case states.Initializing:
			// Notify playerID generated
			cmds.AppendMessage(messages.PlayerIDMessage{
				PlayerID: m.game.Player().ID,
			})
			// Determine next state
			if m.game.Player().Name == "" {
				switchToState(states.InputPlayerName)
			} else {
				switchToState(states.WaitingForPeers)
			}

			// Subscribe to states when app initialized
			convert := func(status transport.ConnectionStatus) messages.ConnectionStatus {
				return messages.ConnectionStatus{Status: status}
			}
			m.transportEventHandler = eventhandler.New[transport.ConnectionStatus, messages.ConnectionStatus](convert)
			cmds.AppendCommand(m.transportEventHandler.Init(
				m.transport.SubscribeToConnectionStatus(),
				m.transport.ConnectionStatus(),
			))

			convert2 := func(status *protocol.State) messages.GameStateMessage {
				return messages.GameStateMessage{State: status}
			}
			m.gameEventHandler = eventhandler.New[*protocol.State, messages.GameStateMessage](convert2)
			cmds.AppendCommand(m.gameEventHandler.Init(
				m.game.SubscribeToStateChanges(),
				m.game.CurrentState(),
			))

		case states.InputPlayerName:
			switchToState(states.WaitingForPeers)

		case states.WaitingForPeers:
			switchToState(states.Playing)
			if config.InitialAction() != "" {
				m.input.SetValue(config.InitialAction())
				cmd := ProcessInput(&m)
				cmds.AppendCommand(cmd)
			}
		case states.Playing:
			break
		}
	case messages.AppStateMessage:
		// Immediately skip to next state if peers already connected
		if m.state == states.WaitingForPeers && m.connectionStatus.PeersCount > 0 {
			cmds.AppendMessage(messages.AppStateFinishedMessage{
				State: states.WaitingForPeers,
			})
		}

	case messages.ConnectionStatus:
		m.connectionStatus = msg.Status
		if m.state == states.WaitingForPeers && m.connectionStatus.PeersCount > 0 {
			cmds.AppendMessage(messages.AppStateFinishedMessage{
				State: states.WaitingForPeers,
			})
		}

	case messages.GameStateMessage:
		if m.gameState != nil && msg.State != nil && msg.State.ActiveIssue != m.gameState.ActiveIssue {
			cmds.AppendMessage(messages.MyVote{Result: m.game.MyVote()})
		}
		m.gameState = msg.State

	case messages.CommandModeChange:
		m.commandMode = msg.CommandMode

	case messages.RoomJoin:
		m.roomID = msg.RoomID
		config.Logger.Debug("room joined",
			zap.String("roomID", msg.RoomID.String()),
			zap.Bool("isDealer", msg.IsDealer))
		cmds.AppendMessage(messages.MyVote{Result: m.game.MyVote()})

	case messages.EnableEnterKey:
		m.disableEnterKey = false
		m.disableEnterRestart = nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			cmds.AppendCommand(commands.QuitApp(m.game))
		case tea.KeyEnter:
			var cmd tea.Cmd
			if m.disableEnterKey {
				break
			}
			if m.input.Focused() {
				cmd = ProcessUserInput(&m)
				cmds.AppendCommand(cmd)
				break
			}
			if m.gameState == nil {
				break
			}
			switch m.roomViewState {
			case states.ActiveIssueView:
				// FIXME: https://github.com/six78/2-story-points-cli/issues/8
				//		  Check `m.gameState == nil`
				if m.gameState.VoteState() == protocol.VotingState {
					cmd = VoteOnCursor(&m)
				} else if m.gameState.VoteState() == protocol.RevealedState {
					cmd = FinishOnCursor(&m)
				}
			case states.IssuesListView:
				cmd = commands.SelectIssue(m.game, m.issuesListView.CursorPosition())
				toggleRoomView(&m)
			}
			cmds.AppendCommand(cmd)
		case tea.KeyTab:
			if !m.roomID.Empty() {
				toggleRoomView(&m)
			}
		case tea.KeyShiftTab:
			cmds.AppendMessage(messages.CommandModeChange{
				CommandMode: !m.commandMode,
			})
		default:
		}

		if m.input.Focused() {
			break
		}

		if !m.roomID.Empty() {
			switch {
			case key.Matches(msg, commands.DefaultKeyMap.ExitRoom):
				cmds.AppendCommand(runExitAction(&m, nil))
			case key.Matches(msg, commands.DefaultKeyMap.RevealVotes):
				cmds.AppendCommand(runRevealAction(&m, nil))
			case key.Matches(msg, commands.DefaultKeyMap.FinishVote):
				cmds.AppendCommand(runFinishAction(&m, nil))
			case key.Matches(msg, commands.DefaultKeyMap.RevokeVote):
				cmds.AppendCommand(commands.PublishVote(m.game, ""))
			}
		} else {
			switch {
			case key.Matches(msg, commands.DefaultKeyMap.NewRoom):
				cmds.AppendCommand(runNewAction(&m, nil))
			}
		}

		message, command := m.handlePastedText(msg.String())
		cmds.AppendMessage(message)
		cmds.AppendCommand(command)
	}

	m.input, cmds.InputCommand = m.input.Update(msg)
	m.spinner, cmds.SpinnerCommand = m.spinner.Update(msg)
	m.errorView = m.errorView.Update(msg)
	m.playersView, cmds.PlayersCommand = m.playersView.Update(msg)
	m.shortcutsView = m.shortcutsView.Update(msg, m.roomViewState)
	m.wakuStatusView = m.wakuStatusView.Update(msg)
	m.deckView = m.deckView.Update(msg)
	m.issueView, cmds.IssueViewCommand = m.issueView.Update(msg)
	m.issuesListView, cmds.IssuesListViewCommand = m.issuesListView.Update(msg)
	m.gameEventHandler, cmds.GameEventHandlerCommand = m.gameEventHandler.Update(msg)
	m.transportEventHandler, cmds.TransportEventHandlerCommand = m.transportEventHandler.Update(msg)

	return m, cmds.Batch()
}

func (m model) View() string {
	if m.fatalError != nil {
		return fmt.Sprintf(" ☠️ fatal error: %s\n%s", m.fatalError, renderLogPath())
	}

	view := "\n"
	if config.Debug() {
		view += fmt.Sprintf("%s\n\n", renderLogPath())
	}
	view += m.renderAppState()

	return lipgloss.JoinHorizontal(lipgloss.Left, "  ", view)
}

func VoteOnCursor(m *model) tea.Cmd {
	return cursorCommand(m, m.deckView.VoteCursor(), commands.PublishVote)
}

func FinishOnCursor(m *model) tea.Cmd {
	return cursorCommand(m, m.deckView.FinishCursor(), commands.FinishVoting)
}

func cursorCommand(m *model, cursor int, command func(*game.Game, protocol.VoteValue) tea.Cmd) tea.Cmd {
	if m.gameState == nil {
		return nil
	}
	if cursor < 0 || cursor > len(m.gameState.Deck) {
		return nil
	}
	cursorValue := m.gameState.Deck[cursor]
	return command(m.game, cursorValue)
}

func toggleRoomView(m *model) {
	switch m.roomViewState {
	case states.ActiveIssueView:
		m.roomViewState = states.IssuesListView
		m.deckView.Blur()
		m.issuesListView.Focus()
	case states.IssuesListView:
		m.roomViewState = states.ActiveIssueView
		m.deckView.Focus()
		m.issuesListView.Blur()
	}
}

func (m *model) handlePastedText(text string) (tea.Msg, tea.Cmd) {
	if len(text) < 16 {
		return nil, nil
	}

	config.Logger.Debug("handlePastedText", zap.String("text", text))

	// Try to parse as room id
	room, err := protocol.ParseRoomID(text)
	if err == nil {
		if !room.VersionSupported() {
			err = errors.Wrap(err, fmt.Sprintf("this room has unsupported version %d", room.Version))
			return messages.NewErrorMessage(err), nil
		}
		roomID := protocol.NewRoomID(text)
		return nil, commands.JoinRoom(m.game, roomID)
	}

	// Try to parse as issues list
	lines := strings.Split(text, "\n")
	cmds := make([]tea.Cmd, 0, len(lines))

	for _, line := range lines {
		u, err := url.Parse(line)
		if err != nil {
			config.Logger.Warn("failed to parse issue url", zap.Error(err))
			continue
		}
		cmds = append(cmds, commands.AddIssue(m.game, u.String()))
	}

	m.disableEnterKey = true

	if m.disableEnterRestart != nil {
		m.disableEnterRestart <- struct{}{}
	} else {
		m.disableEnterRestart = make(chan struct{}, 100)
		cmd := commands.DelayMessage(
			100*time.Millisecond,
			messages.EnableEnterKey{},
			m.disableEnterRestart,
		)
		cmds = append(cmds, cmd)
	}

	return nil, tea.Batch(cmds...)
}

// Ensure that model fulfils the tea.Model interface at compile time.
// ref: https://www.inngest.com/blog/interactive-clis-with-bubbletea
var _ tea.Model = (*model)(nil)
