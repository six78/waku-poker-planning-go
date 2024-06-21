package game

import (
	"golang.org/x/exp/maps"

	"github.com/six78/2-story-points-cli/pkg/protocol"
)

var fibonacciDeck = protocol.Deck{"1", "2", "3", "5", "8", "13", "21", "?"}

const Fibonacci = "fibonacci"

//var TShirtDeck = []protocol.VoteResult{
//	"XS", "S", "M", "L", "XL", "XXL",
//}

var decks = map[string]protocol.Deck{
	Fibonacci: fibonacciDeck,
}

func GetDeck(deckName string) (protocol.Deck, bool) {
	deck, ok := decks[deckName]
	return deck, ok
}

func AvailableDecks() []string {
	return maps.Keys(decks)
}

func CreateDeck(votes []string) protocol.Deck {
	result := protocol.Deck{}
	for _, value := range votes {
		result = append(result, protocol.VoteValue(value))
	}
	return result
}
