package cards

import (
	"math/rand"
	"strings"
	"time"

	"github.com/Jaggernaut555/respecbot/queue"
)

// CardNames - Array of the names of each card
var CardNames = []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}

// SuitNames - Array of the names of each suit
var SuitNames = []string{"♡", "♢", "♤", "♧"}

// Card - Index in CardNames that this Card refers to
type Card struct {
	cardIndex int
	suitIndex int
}

func (c Card) String() string {
	return CardNames[c.cardIndex] + SuitNames[c.suitIndex]
}

// Deck - Just an array of Cards
type Deck struct {
	cards *queue.ListQueue
}

func (d Deck) String() string {
	var cards []string
	tail := d.cards.End()
	for v := d.cards.Pop(); v != tail; v = d.cards.Pop() {
		cards = append(cards, v.Data.(Card).String())
		d.cards.Push(v.Data)
	}
	d.cards.Push(tail.Data)
	cards = append(cards, tail.Data.(Card).String())
	return strings.Join(cards, ", ")
}

// Package constants
const numSuits = 4
const numCardsPerSuit = 13

func init() {
	rand.Seed(time.Now().Unix())
}

// New - Build a new Deck
func New(shuffleTheDeck bool) Deck {
	var newDeck Deck
	newDeck.cards = queue.NewListQueue(Card{})
	for i := 0; i < numSuits; i++ {
		for j := 0; j < numCardsPerSuit; j++ {
			newDeck.AddCard(buildCard(i, j), false)
		}
	}

	if shuffleTheDeck {
		newDeck.Shuffle()
	}

	return newDeck
}

// DrawCard - Pick a card from the top of the deck
func (d Deck) DrawCard() Card {
	c := d.cards.Pop()
	return c.Data.(Card)
}

// AddCard - Add a card to the deck, optionally shuffle afterwards
func (d Deck) AddCard(c Card, shuffleTheDeck bool) {
	d.cards.Push(c)

	if shuffleTheDeck {
		d.Shuffle()
	}
}

// Shuffle - Shuffle the deck up
func (d *Deck) Shuffle() {
	var numCards = d.cards.Length()
	var n int
	for i := 0; i < numCards; i++ {
		n = rand.Intn(numCards - i)
		d.AddCard(d.cards.Remove(n).Data.(Card), false)
	}
}

// Size - Get the amount of cards left
func (d Deck) Size() int {
	return d.cards.Length()
}

// GenerateCard - Generate a random card from nothing
func GenerateCard() Card {
	return buildCard(rand.Intn(numSuits), rand.Intn(numCardsPerSuit))
}

func buildCard(suitIndex int, cardIndex int) Card {
	var newCard Card
	newCard.suitIndex = suitIndex
	newCard.cardIndex = cardIndex
	return newCard
}

// SameSuit - Compare one card to another to check if they match suits
func (c Card) SameSuit(card Card) bool {
	return c.suitIndex == card.suitIndex
}

// SameNumber - Compare one card to another to check if they match numbers
func (c Card) SameNumber(card Card) bool {
	return c.cardIndex == card.cardIndex
}
