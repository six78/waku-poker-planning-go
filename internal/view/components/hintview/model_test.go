package hintview

import (
	"fmt"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"

	"github.com/six78/2-story-points-cli/internal/view/messages"
	"github.com/six78/2-story-points-cli/pkg/protocol"
)

func TestInit(t *testing.T) {
	model := New()

	cmd := model.Init()
	require.Nil(t, cmd)
	require.Nil(t, model.hint)
}

func TestUpdateNilState(t *testing.T) {
	model := New()
	cmd := model.Init()
	model, cmd = model.Update(messages.GameStateMessage{State: nil})
	require.Nil(t, cmd)
	require.Nil(t, model.hint)
	require.Empty(t, model.View())
}

func TestUpdateVotesNotRevealed(t *testing.T) {
	model := New()
	cmd := model.Init()
	model, cmd = model.Update(messages.GameStateMessage{
		State: &protocol.State{
			VotesRevealed: false,
		},
	})
	require.Nil(t, cmd)
	require.Nil(t, model.hint)
	require.Empty(t, model.View())
}

func TestUpdateAcceptableVote(t *testing.T) {
	model := New()

	cmd := model.Init()
	require.Nil(t, cmd)
	require.Nil(t, model.hint)

	// Test acceptable vote

	testCases := []struct {
		name       string
		acceptable bool
	}{
		{
			name:       "acceptable vote",
			acceptable: true,
		},
		{
			name:       "non-acceptable vote",
			acceptable: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			issue := protocol.Issue{
				ID: protocol.IssueID(gofakeit.UUID()),
				Hint: &protocol.Hint{
					Acceptable:   tc.acceptable,
					Hint:         protocol.VoteValue(gofakeit.LetterN(5)),
					RejectReason: gofakeit.LetterN(10),
				},
			}
			model, cmd = model.Update(messages.GameStateMessage{
				State: &protocol.State{
					Issues:        protocol.IssuesList{&issue},
					ActiveIssue:   issue.ID,
					VotesRevealed: true,
				},
			})
			require.Nil(t, cmd)
			require.NotNil(t, model.hint)
			require.Equal(t, *issue.Hint, *model.hint)

			expectedVerdict := "✓"
			if !tc.acceptable {
				expectedVerdict = "x" + fmt.Sprintf(" (%s)", issue.Hint.RejectReason)
			}

			expectedLines := []string{
				"",
				"Recommended: " + string(issue.Hint.Hint),
				"Acceptable:  " + expectedVerdict,
				"What to do:",
				"",
			}

			lines := strings.Split(model.View(), "\n")
			require.Len(t, lines, len(expectedLines))

			for i, line := range lines {
				trimmedLine := strings.Trim(line, " ")
				require.Equal(t, expectedLines[i], trimmedLine)
			}
		})
	}
}
