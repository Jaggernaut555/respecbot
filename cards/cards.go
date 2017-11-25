package cards

import "math/rand"

// CardNames - Array of the names of each card
var CardNames = []string{"A", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}

// SuitNames - Array of the names of each suit
var SuitNames = []string{"♡", "♢", "♤", "♧"}

// Card - Index in CardNames that this Card refers to
type Card struct {
	name      string
	cardIndex int
	suitIndex int
}

// Deck - Just an array of Cards
type Deck struct {
	cards []Card
}

// Package constants
const numSuits = 4
const numCardsPerSuit = 13

// New - Build a new Deck
func New(shuffleTheDeck bool) Deck {
	var newDeck Deck
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

// DrawCard - Pick a card from the deck
func (d Deck) DrawCard() Card {
	var numCards = len(d.cards)
	var pickedCardIdx = rand.Intn(numCards)
	var pickedCard = d.cards[pickedCardIdx]
	d.cards = append(d.cards[:pickedCardIdx], d.cards[pickedCardIdx+1:]...)
	return pickedCard
}

// AddCard - Add a card to the deck, optionally shuffle afterwards
func (d Deck) AddCard(c Card, shuffleTheDeck bool) {
	d.cards = append(d.cards, c)

	if shuffleTheDeck {
		d.Shuffle()
	}
}

// Shuffle - Shuffle the deck up
func (d Deck) Shuffle() {
	var numCards = len(d.cards)
	var newCardsArray = make([]Card, numCards)
	var perm = rand.Perm(numCards)
	for i, j := range perm {
		newCardsArray[j] = d.cards[i]
	}
	d.cards = newCardsArray
}

// GenerateCard - Generate a random card from nothing
func GenerateCard() Card {
	return buildCard(rand.Intn(numSuits), rand.Intn(numCardsPerSuit))
}

func buildCard(suitIndex int, cardIndex int) Card {
	var newCard Card
	newCard.suitIndex = suitIndex
	newCard.cardIndex = cardIndex
	newCard.name = CardNames[cardIndex] + SuitNames[suitIndex]
	return newCard
}
