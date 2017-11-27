package cards

import "testing"

func TestCards(t *testing.T) {
	var deck Deck

	deck = New(false)
	if deck.Size() != 52 {
		t.Error("Could not create deck")
	}

	aceHearts := buildCard(0, 0)
	if aceHearts.String() != "A♡" {
		t.Error("Card string not formed correctly")
	}
	if deck.String() != "[A♡,2♡,3♡,4♡,5♡,6♡,7♡,8♡,9♡,10♡,J♡,Q♡,K♡,A♢,2♢,3♢,4♢,5♢,6♢,7♢,8♢,9♢,10♢,J♢,Q♢,K♢,A♤,2♤,3♤,4♤,5♤,6♤,7♤,8♤,9♤,10♤,J♤,Q♤,K♤,A♧,2♧,3♧,4♧,5♧,6♧,7♧,8♧,9♧,10♧,J♧,Q♧,K♧]" {
		t.Error("Deck string not formed correctly")
	}

	if !aceHearts.SameNumber(deck.DrawCard()) {
		t.Error("Numbers do not match")
	}
	if !aceHearts.SameSuit(deck.DrawCard()) {
		t.Error("Suits do not match")
	}
	if deck.Size() != 50 {
		t.Error("Drawing cards did not work")
	}

	deck.Shuffle()
	anotherDeck := New(true)

	card1 := deck.DrawCard()
	card2 := anotherDeck.DrawCard()
	if card1.SameNumber(card2) && card1.SameSuit(card2) {
		t.Log("Statisticly improbable")
	}

	num := deck.Size()
	deck.AddCard(card1, false)
	deck.AddCard(GenerateCard(), true)
	if num == deck.Size() {
		t.Error("AddCard failed")
	}
}
