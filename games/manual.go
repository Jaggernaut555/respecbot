package games

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Jaggernaut555/respecbot/bet"
	"github.com/Jaggernaut555/respecbot/db"
	"github.com/Jaggernaut555/respecbot/logging"
	"github.com/Jaggernaut555/respecbot/state"
	"github.com/bwmarrin/discordgo"
)

var (
	allBets  map[string]*bet.Bet
	betMuxes map[string]*sync.Mutex
	location *time.Location
)

func init() {
	allBets = make(map[string]*bet.Bet)
	betMuxes = make(map[string]*sync.Mutex)
	var err error
	location, err = time.LoadLocation("America/Vancouver")
	if err != nil {
		panic(err)
	}
}

func ManualBetCmd(message *discordgo.Message, args []string) {
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
		reply += "'bet help' - display this message\n"
		reply += "'bet status' - display the status of an active bet\n"
		reply += "'bet [value] [@user/role/everyone] - create a bet\n"
		reply += "(No target is the same as @everyone)\n"
		reply += "'bet call' - Call the active bet\n"
		reply += "'bet drop' - Drop out of a bet\n"
		reply += "'bet lose' - Lose the bet\n"
		reply += "'bet start' - Start a bet early, otherwise it will start 2 minutes after it's made or when every target in the bet is ready\n"
		reply += "'bet cancel' - Cancel the active bet\n"
		reply += "(Only the bet creator can start/cancel the bet)"
		reply += "```"
		state.SendReply(message.ChannelID, reply)
		return
	}

	mux, ok := betMuxes[message.ChannelID]

	if !ok {
		mux = new(sync.Mutex)
		betMuxes[message.ChannelID] = mux
	}

	mux.Lock()

	if b, ok := allBets[message.ChannelID]; ok {
		manualActiveBetCommand(mux, b, message, args[1])
	} else {
		manualCreateBet(mux, message, args)
	}

	mux.Unlock()
}

func manualActiveBetCommand(mux *sync.Mutex, b *bet.Bet, message *discordgo.Message, cmd string) {
	// bet exists, check if user is active or able to join

	author := message.Author
	userStatus, ok := b.UserStatus[author.ID]

	if !ok {
		ok = b.Open
	}

	switch strings.ToLower(cmd) {
	// begin bet with current active users
	case "start":
		if author.ID == b.AuthorID && !b.Started {
			b.State <- bet.BetMessage{User: author, Arg: "start"}
		}

	// cannot lose if not active
	case "lose":
		if userStatus == bet.Playing && ok && b.Started {
			b.State <- bet.BetMessage{User: author, Arg: "lose"}
		}

	// drop a bet before it starts
	case "drop":
		if userStatus == bet.Playing && ok {
			if b.Started {
				b.State <- bet.BetMessage{User: author, Arg: "lose"}
			} else {
				b.State <- bet.BetMessage{User: author, Arg: "drop"}
			}
		}

	// validate user can call
	case "call":
		if userStatus == bet.Lost && ok && !b.Started {
			available := db.GetUserRespec(author)
			if available >= b.UserBet[b.AuthorID] {
				b.State <- bet.BetMessage{User: author, Arg: "call", Bet: b.UserBet[b.AuthorID], Odds: 1}
			} else {
				state.SendReply(message.ChannelID, "Not enough respec to call")
			}
		}

	// cannot cancel started bet
	case "cancel":
		if author.ID == b.AuthorID {
			b.State <- bet.BetMessage{User: author, Arg: "cancel"}
		}

	case "status":
		b.State <- bet.BetMessage{User: author, Arg: "status"}

	default:
		reply := fmt.Sprintf("Not a valid for active bet, use call/lose/start/cancel/status")
		state.SendReply(message.ChannelID, reply)
		b.State <- bet.BetMessage{User: author, Arg: "invalid"}
	}
}

func manualCreateBet(mux *sync.Mutex, message *discordgo.Message, args []string) {
	// bet does not exist, check if valid bet then create it
	// validate user has enough respec to create bet
	author := message.Author
	available := db.GetUserRespec(author)
	num, err := strconv.Atoi(args[1])
	if err != nil || num < 1 || available < num {
		reply := fmt.Sprintf("Invalid wager")
		state.SendReply(message.ChannelID, reply)
		return
	}

	channel, err := state.Session.Channel(message.ChannelID)
	if err != nil {
		return
	}

	var b bet.Bet
	b.ManualBet = true
	b.AuthorID = author.ID
	b.ChannelID = message.ChannelID
	b.GuildID = channel.GuildID
	b.Open = message.MentionEveryone
	b.State = make(chan bet.BetMessage, 5)
	b.Time = time.Now().In(location)
	b.Users = make(map[string]*discordgo.User)
	b.UserStatus = make(map[string]int)
	b.UserOdds = make(map[string]float64)
	b.UserBet = make(map[string]int)
	b.UserEarnings = make(map[string]int)

	if b.Open || len(args) == 2 ||
		(len(message.Mentions) == 0 && len(message.MentionRoles) == 0) {
		b.Open = true
	} else {
		// check if role mentioned
		bet.AppendRoles(message, &b)

		for _, v := range message.Mentions {
			if bet.UserCanBet(v, num) {
				b.UserStatus[v.ID] = bet.Lost
				b.Users[v.ID] = v
			}
		}
	}

	if len(b.Users) < 1 && !b.Open {
		reply := "No users can participate in this bet"
		state.SendReply(b.ChannelID, reply)
		return
	}

	if mux != betMuxes[message.ChannelID] {
		return
	}

	allBets[message.ChannelID] = &b

	go bet.BetEngage(b.State, &b, mux)
	go bet.StartBetTimer(b.State)

	b.State <- bet.BetMessage{User: author, Arg: "call", Bet: num, Odds: 1}

	reply := fmt.Sprintf("%v started a bet of %v", author.String(), num)
	logging.Log(reply)
}
