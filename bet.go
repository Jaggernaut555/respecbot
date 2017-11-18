package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type betMessage struct {
	user *discordgo.User
	arg  string
}

type Bet struct {
	respec      int
	totalRespec int
	started     bool
	open        bool
	author      *discordgo.User
	userStatus  map[string]bool
	users       map[string]*discordgo.User
	state       chan betMessage
	time        time.Time
	channelID   string
	guildID     string
}

var (
	allBets  map[string]*Bet
	betMuxes map[string]*sync.Mutex
)

func InitBets() {
	allBets = make(map[string]*Bet)
	betMuxes = make(map[string]*sync.Mutex)
}

func bet(message *discordgo.Message, args []string) {
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
		reply += "-Make a wager with\n'[value] @[users/roles/everyone/EMPTY]...'\n"
		reply += "No target is the same as @everyone\n"
		reply += "-Respond with 'call'\n"
		reply += "-To drop out of a bet use 'drop'\n"
		reply += "-To start a bet early use 'start', otherwise it will start 2 minutes after it's made\n"
		reply += "-To cancel the whole bet use 'cancel'\n"
		reply += "-Only the bet creator can start/cancel the bet"
		reply += "```"
		SendReply(message.ChannelID, reply)
		return
	}

	mux, ok := betMuxes[message.ChannelID]

	if !ok {
		mux = new(sync.Mutex)
		betMuxes[message.ChannelID] = mux
	}

	mux.Lock()

	if b, ok := allBets[message.ChannelID]; ok {
		activeBetCommand(mux, b, message.Author, message, args[1])
	} else {
		createBet(mux, message.Author, message, args)
	}

	mux.Unlock()
}

func activeBetCommand(mux *sync.Mutex, b *Bet, author *discordgo.User, message *discordgo.Message, cmd string) {
	// bet exists, check if user is active or able to join

	userStatus, ok := b.userStatus[author.ID]

	if !ok {
		ok = b.open
	}

	switch strings.ToLower(cmd) {

	// begin bet with current active users
	case "start":
		if author.ID == b.author.ID && !b.started {
			b.state <- betMessage{user: author, arg: "start"}
		}

	// cannot lose if not active
	case "lose":
		if userStatus && ok && b.started {
			b.state <- betMessage{user: author, arg: "lose"}
		}

	// drop a bet before it starts
	case "drop":
		if userStatus && ok && !b.started {
			b.state <- betMessage{user: author, arg: "drop"}
		}

	// validate user can call
	case "call":
		if !userStatus && ok && !b.started {
			available := dbGetUserRespec(author)
			if available >= b.respec {
				b.state <- betMessage{user: author, arg: "call"}
			} else {
				SendReply(message.ChannelID, "Not enough respec to call")
			}
		}

	// cannot cancel started bet
	case "cancel":
		if author.ID == b.author.ID {
			b.state <- betMessage{user: author, arg: "cancel"}
		}

	default:
		reply := fmt.Sprintf("Not a valid for active bet, use call/lose/start/cancel")
		SendReply(message.ChannelID, reply)
	}
}

func createBet(mux *sync.Mutex, author *discordgo.User, message *discordgo.Message, args []string) {
	// bet does not exist, check if valid bet then create it
	// validate user has enough respec to create bet
	available := dbGetUserRespec(author)
	num, err := strconv.Atoi(args[1])
	if err != nil || num < 1 || available < num {
		reply := fmt.Sprintf("Invalid wager")
		SendReply(message.ChannelID, reply)
		return
	}

	channel, err := DiscordSession.Channel(message.ChannelID)
	if err != nil {
		return
	}

	var b Bet
	b.author = author
	b.channelID = message.ChannelID
	b.guildID = channel.GuildID
	b.open = message.MentionEveryone
	b.respec = num
	b.state = make(chan betMessage, 5)
	b.time = time.Now()
	b.users = make(map[string]*discordgo.User)
	b.userStatus = make(map[string]bool)

	if b.open || len(args) == 2 ||
		(len(message.Mentions) == 0 && len(message.MentionRoles) == 0) {
		b.open = true
	} else {
		// check if role mentioned
		appendRoles(message, &b)

		for _, v := range message.Mentions {
			b.userStatus[v.ID] = false
			b.users[v.ID] = v
		}
	}

	b.userStatus[author.ID] = true
	b.users[author.ID] = author
	addRespec(b.guildID, b.author, -b.respec)

	if mux != betMuxes[message.ChannelID] {
		return
	}

	allBets[message.ChannelID] = &b

	go betEngage(b.state, &b, mux)
	go startBetTimer(b.state)

	reply := fmt.Sprintf("%v started a bet of %v", author.Mention(), b.respec)
	SendReply(message.ChannelID, reply)
	reply = fmt.Sprintf("%v started a bet of %v", author.String(), b.respec)
	log.Println(reply)
}

func startBetTimer(c chan betMessage) {
	timer := time.NewTicker(time.Minute * 2)
	<-timer.C
	c <- betMessage{user: nil, arg: "start"}
}

func appendRoles(message *discordgo.Message, b *Bet) {
	channel, err := DiscordSession.Channel(message.ChannelID)
	if err != nil {
		panic(err)
	}
	mentionedRoles := message.MentionRoles
	var roleUsers []*discordgo.User

	guild, _ := DiscordSession.Guild(channel.GuildID)

	for _, v := range mentionedRoles {
		roleUsers = append(roleUsers, roleHelper(guild, v)...)
	}

	for _, v := range roleUsers {
		b.users[v.ID] = v
		b.userStatus[v.ID] = false
	}
}

func roleHelper(guild *discordgo.Guild, roleID string) (users []*discordgo.User) {
	members := guild.Members
	for _, v := range members {
		for _, role := range v.Roles {
			if roleID == role {
				users = append(users, v.User)
				break
			}
		}
	}
	return
}

// goroutine to run an active bet
func betEngage(c chan betMessage, b *Bet, mux *sync.Mutex) {
	var winnerID string

Loop:
	for i := range c {
		mux.Lock()
		switch i.arg {
		case "call":
			callBet(b, i.user)
		case "lose":
			loseBet(b, i.user)
		case "drop":
			dropOut(b, i.user)
		case "start":
			startBet(b)
		case "cancel":
			cancelBet(b)
		}

		if !b.started && !b.open {
			if checkBetReady(b.userStatus) {
				startBet(b)
			}
		}
		if b.started {
			var ok bool
			if winnerID, ok = checkWinner(b.userStatus); ok {
				break Loop
			} else {
				reply := "Active Betters: "
				for k, v := range b.userStatus {
					if v {
						reply += b.users[k].Mention()
					}
				}
				SendReply(b.channelID, reply)
			}
		}
		mux.Unlock()
	}

	var reply string

	if winner, ok := b.users[winnerID]; ok {
		betWon(b, winner)
		reply = fmt.Sprintf("Bet ended. %v won %v respec", winner.Mention(), b.totalRespec-b.respec)
		log.Printf("Bet ended. %v won %v respec\n", winner.String(), b.totalRespec-b.respec)
	} else {
		reply = fmt.Sprintf("Bet ended. No winner, repsec refunded")
		log.Println(reply)
	}

	SendReply(b.channelID, reply)

	delete(allBets, b.channelID)
	mux.Unlock()
}

func callBet(b *Bet, user *discordgo.User) {
	if b.userStatus[user.ID] {
		return
	}
	b.userStatus[user.ID] = true
	b.users[user.ID] = user

	log.Printf("%+v called\n", user.String())

	addRespec(b.guildID, user, -b.respec)
}

func loseBet(b *Bet, user *discordgo.User) {
	if !b.userStatus[user.ID] {
		return
	}
	b.userStatus[user.ID] = false

	log.Printf("%+v lost\n", user.String())
}

func dropOut(b *Bet, user *discordgo.User) {
	if !b.userStatus[user.ID] {
		return
	}
	b.userStatus[user.ID] = false

	log.Printf("%+v dropped out\n", user.String())

	addRespec(b.guildID, user, b.respec)
}

func betWon(b *Bet, winner *discordgo.User) {
	addRespec(b.guildID, winner, b.totalRespec)

	for _, v := range b.users {
		if v.ID != winner.ID {
			addRespec(b.guildID, v, -b.respec)
		}
	}
}

func cancelBet(b *Bet) {
	for k, v := range b.userStatus {
		if v {
			addRespec(b.guildID, b.users[k], b.respec)
			b.userStatus[k] = false
		}
	}

	reply := fmt.Sprintf("Bet Cancelled")

	b.started = true
	SendReply(b.channelID, reply)
	log.Println(reply)
}

func startBet(b *Bet) {
	if b.started {
		return
	}
	count := 0
	for k, v := range b.userStatus {
		if !v {
			delete(b.userStatus, k)
			delete(b.users, k)
		} else {
			b.totalRespec += b.respec
			count++
		}
	}
	if count == 0 {
		b.state <- betMessage{user: nil, arg: "cancel"}
		return
	}
	b.started = true
	go betEndTimer(b.state)
	reply := fmt.Sprintf("Bet started: must end before %v", time.Now().Add(time.Minute*30).Format("15:04:05"))
	SendReply(b.channelID, reply)
	log.Println(reply)
}

func betEndTimer(c chan betMessage) {
	timer := time.NewTicker(time.Minute * 30)
	<-timer.C
	c <- betMessage{user: nil, arg: "cancel"}
}

// check if only one user has not lost the bet
func checkWinner(userstatus map[string]bool) (winner string, won bool) {
	count := 0
	for k, v := range userstatus {
		if v {
			winner = k
			count++
		}
		if count > 1 {
			return "", false
		}
	}
	if count == 0 {
		return "", true
	}
	return winner, true
}

func checkBetReady(users map[string]bool) bool {
	for _, v := range users {
		if !v {
			return false
		}
	}
	return true
}

// multiple pot winners?
