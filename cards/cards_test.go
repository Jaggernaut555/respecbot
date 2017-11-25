package cards

import "testing"

func TestCards(t *testing.T) {
	var deck Deck

	deck = New(false)
	if deck.Size() != 52 {
		t.Errorf("Could not create deck")
	}

	aceHearts := buildCard(0, 0)

	if !aceHearts.SameNumber(deck.DrawCard()) {
		t.Errorf("Numbers do not match")
	}
	if !aceHearts.SameSuit(deck.DrawCard()) {
		t.Errorf("Suits do not match")
	}
	if deck.Size() != 50 {
		t.Errorf("Drawing cards did not work")
	}

	deck.Shuffle()
	anotherDeck := New(true)

	card1 := deck.DrawCard()
	card2 := anotherDeck.DrawCard()
	if card1.SameNumber(card2) && card1.SameSuit(card2) {
		t.Logf("Statisticly improbable")
	}

	num := deck.Size()
	deck.AddCard(card1, false)
	if num == deck.Size() {
		t.Errorf("AddCard failed")
	}
}
