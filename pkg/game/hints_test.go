package game

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/pkg/errors"
	"github.com/six78/2-story-points-cli/pkg/protocol"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"math"
	"sort"
	"strings"
	"testing"
)

type Hint struct {
	value     protocol.VoteValue
	deviation float64
}

// TODO: generic
func MedianInt(values []int) (int, int) {
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})

	center := len(values) / 2
	return values[center], center
}

func Median(votes []protocol.VoteValue) (protocol.VoteValue, int) {
	sort.Slice(votes, func(i, j int) bool {
		return votes[i] < votes[j]
	})

	center := len(votes) / 2
	return votes[center], center
}

type Recommendation struct {
	MedianValue              int     `json:"median"`
	MedianAbsoluteDeviation  float64 `json:"median_absolute_deviation"`
	MaximumAbsoluteDeviation float64 `json:"maximum_absolute_deviation"`
}

// recommend returns:
// - median value
// - median absolute deviation
// - maximum absolute deviation
// - error if any occurred
func recommend(values []int) (*Recommendation, error) {
	r := &Recommendation{}

	// Median value
	r.MedianValue, _ = MedianInt(values)

	// TODO: Maximum deviation
	r.MaximumAbsoluteDeviation = 0
	for _, v := range values {
		deviation := math.Abs(float64(r.MedianValue) - float64(v))
		r.MaximumAbsoluteDeviation = math.Max(r.MaximumAbsoluteDeviation, deviation)
	}

	// Average deviation
	sum := 0
	for _, v := range values {
		sum += int(math.Abs(float64(r.MedianValue) - float64(v)))
	}
	r.MedianAbsoluteDeviation = float64(sum) / float64(len(values))

	return r, nil

}

func Recommend(deck protocol.Deck, votes []protocol.VoteValue) (*Recommendation, error) {
	indexes := make([]int, len(votes))
	for i, vote := range votes {
		index := slices.Index(deck, vote)
		if index < 0 {
			return nil, errors.New("vote not found in deck")
		}
		indexes[i] = index
	}

	return recommend(indexes)
}

func TestMedian(t *testing.T) {
	// Odd number of votes
	votes := []protocol.VoteValue{"1", "1", "1", "1", "2"}
	hint, index := Median(votes)
	require.Equal(t, protocol.VoteValue("1"), hint)
	require.Equal(t, 2, index)

	// Even number of votes
	votes = []protocol.VoteValue{"1", "1", "1", "2"}
	hint, index = Median(votes)
	require.Equal(t, protocol.VoteValue("1"), hint)
	require.Equal(t, 2, index)

	// Test round up
	votes = []protocol.VoteValue{"1", "1", "2", "2"}
	hint, index = Median(votes)
	require.Equal(t, protocol.VoteValue("2"), hint)
	require.Equal(t, 2, index)
}

func TestHint(t *testing.T) {
	deck := protocol.Deck{"1", "2", "3", "5", "8", "13", "21"}
	t.Log("deck:", deck)

	records := [][]protocol.VoteValue{
		{"3", "3", "3", "3", "5"},
		{"3", "3", "3", "3", "8"},
		{"3", "3", "3", "3", "13"},
		{"3", "3", "3", "3", "21"},
		{"3", "3", "3", "5", "5"},
		{"3", "3", "3", "5", "8"},
		{"2", "3", "3", "3", "3", "3", "5"},
		{"2", "3", "3", "3", "3", "5"},
		{"2", "3", "3", "3", "5"},
		{"2", "3", "3", "5"},
		{"2", "3", "5"},
		{"2", "3", "5", "8"},
	}

	var rows [][]string

	for _, votes := range records {
		recommendation, err := Recommend(deck, votes)
		require.NoError(t, err)
		require.NotNil(t, recommendation)

		strs := make([]string, 0, len(votes))
		for _, v := range votes {
			strs = append(strs, string(v))
		}

		row := []string{
			strings.Join(strs, " "),
			string(deck[recommendation.MedianValue]),
			fmt.Sprintf("%.2f", recommendation.MedianAbsoluteDeviation),
			fmt.Sprintf("%.2f", recommendation.MaximumAbsoluteDeviation),
		}

		rows = append(rows, row)

		t.Log("votes:", votes,
			" median:", deck[recommendation.MedianValue],
			" median abs deviation:", recommendation.MedianAbsoluteDeviation,
			" max abs deviation:", recommendation.MaximumAbsoluteDeviation)
	}

	results := table.New().
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case col == 0:
				return lipgloss.NewStyle().Align(lipgloss.Left).PaddingLeft(1).PaddingRight(1)
			default:
				return lipgloss.NewStyle().Align(lipgloss.Center).PaddingLeft(1).PaddingRight(1)
			}
		}).
		Headers("Votes", "Median", "Median abs\ndeviation", "Max abs\ndeviation").
		Rows(rows...)

	fmt.Println(results.String())

	//require.Equal(t, 2, medianIndex)
	//require.Equal(t, 1.2, medianDeviation)
}
