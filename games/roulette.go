package games

import (
	"sync"

	"github.com/Jaggernaut555/respecbot/bet"
	"github.com/Jaggernaut555/respecbot/state"
	"github.com/bwmarrin/discordgo"
)

type roulette struct {
	bet.Bet
}

var (
	roulettes     map[string]roulette
	rouletteMuxes map[string]*sync.Mutex
)

func init() {
	roulettes = make(map[string]roulette)
	rouletteMuxes = make(map[string]*sync.Mutex)
}

func rouletteCmd(message *discordgo.Message, args []string) {
	/*
		format
		bet 50 @user1 @user2 ... (must have enough score, cap of 50?)
		one to many users in pot may accept (must have enough score)
		after at least one user has accepted, bet is active
		make sure user doesn't mention themself

		maybe just 'bet 50' and anybody can accept into the pool?
		One active bet per channel
	*/

	if len(args) < 2 || args[1] == "help" {
		reply := "```"
		reply += "under construction\n"
		reply += "```"
		state.SendReply(message.ChannelID, reply)
		return
	}

}
