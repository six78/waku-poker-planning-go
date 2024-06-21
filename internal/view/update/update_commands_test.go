package update

import (
	"reflect"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/suite"

	"github.com/six78/2-story-points-cli/internal/testcommon"
)

func TestUpdateCommands(t *testing.T) {
	suite.Run(t, new(UpdateCommandsSuite))
}

type UpdateCommandsSuite struct {
	testcommon.Suite
}

func (s *UpdateCommandsSuite) TestEmpty() {
	update := NewUpdateCommands()
	batch := update.Batch()
	s.Require().Nil(batch)
}

func (s *UpdateCommandsSuite) TestAppendCommand() {
	sentMessage := gofakeit.LetterN(5)
	sentCommand := func() tea.Msg {
		return sentMessage
	}

	update := NewUpdateCommands()
	update.AppendCommand(sentCommand)

	batch := update.Batch()
	s.Require().NotNil(batch)
	s.Require().Equal(reflect.Func, reflect.TypeOf(batch).Kind())

	result := batch()
	s.Require().NotNil(result)

	batchMessage := result.(tea.BatchMsg)
	s.Require().NotNil(batchMessage)
	s.Require().Len(batchMessage, 1)

	receivedCommand := batchMessage[0]
	receivedMessage := receivedCommand()
	s.Require().Equal(sentMessage, receivedMessage)
}

func (s *UpdateCommandsSuite) TestAppendMessage() {
	sentMessage := gofakeit.LetterN(5)

	update := NewUpdateCommands()
	update.AppendMessage(sentMessage)

	batch := update.Batch()
	s.Require().NotNil(batch)
	s.Require().Equal(reflect.Func, reflect.TypeOf(batch).Kind())

	result := batch()
	s.Require().NotNil(result)

	batchMessage := result.(tea.BatchMsg)
	s.Require().NotNil(batchMessage)
	s.Require().Len(batchMessage, 1)

	receivedCommand := batchMessage[0]
	receivedMessage := receivedCommand()
	s.Require().Equal(sentMessage, receivedMessage)
}

func (s *UpdateCommandsSuite) TestStandardCommands() {
	var messages = make([]string, 5)
	for i := 0; i < len(messages); i++ {
		messages[i] = gofakeit.LetterN(5)
	}

	var commands = make([]tea.Cmd, len(messages))
	for i := 0; i < len(commands); i++ {
		message := messages[i]
		commands[i] = func() tea.Msg {
			return message
		}
	}

	update := NewUpdateCommands()
	update.InputCommand = commands[0]
	update.SpinnerCommand = commands[1]
	update.PlayersCommand = commands[2]
	update.IssueViewCommand = commands[3]
	update.IssuesListViewCommand = commands[4]

	batch := update.Batch()
	s.Require().NotNil(batch)

	batchMessage := s.SplitBatch(batch)
	s.Require().Len(batchMessage, len(commands))

	for i := 0; i < len(commands); i++ {
		receivedCommand := batchMessage[i]
		receivedMessage := receivedCommand()
		s.Require().Equal(messages[i], receivedMessage)
	}
}
