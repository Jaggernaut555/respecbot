package main

import (
	"fmt"
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
	respec       int
	totalRespec  int
	started      bool
	open         bool
	cancelled    bool
	author       *discordgo.User
	winner       *discordgo.User
	userStatus   map[string]bool
	users        map[string]*discordgo.User
	state        chan betMessage
	time         time.Time
	endTime      time.Time
	channelID    string
	guildID      string
	announcement *discordgo.Message
}

var (
	allBets  map[string]*Bet
	betMuxes map[string]*sync.Mutex
	location *time.Location
)

func InitBets() {
	allBets = make(map[string]*Bet)
	betMuxes = make(map[string]*sync.Mutex)
	var err error
	location, err = time.LoadLocation("America/Vancouver")
	if err != nil {
		panic(err)
	}
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
		if userStatus && ok {
			if b.started {
				b.state <- betMessage{user: author, arg: "lose"}
			} else {
				b.state <- betMessage{user: author, arg: "drop"}
			}
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

	case "status":
		b.state <- betMessage{user: author, arg: "status"}

	default:
		reply := fmt.Sprintf("Not a valid for active bet, use call/lose/start/cancel/status")
		SendReply(message.ChannelID, reply)
		b.state <- betMessage{user: author, arg: "invalid"}
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
	b.totalRespec = num
	b.state = make(chan betMessage, 5)
	b.time = time.Now().In(location)
	b.users = make(map[string]*discordgo.User)
	b.userStatus = make(map[string]bool)

	if b.open || len(args) == 2 ||
		(len(message.Mentions) == 0 && len(message.MentionRoles) == 0) {
		b.open = true
	} else {
		// check if role mentioned
		appendRoles(message, &b)

		for _, v := range message.Mentions {
			if userCanBet(v, b.respec) {
				b.userStatus[v.ID] = false
				b.users[v.ID] = v
			}
		}
	}

	if len(b.users) < 1 && !b.open {
		reply := "No users can participate in this bet"
		SendReply(b.channelID, reply)
		return
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

	reply := fmt.Sprintf("%v started a bet of %v", author.String(), b.respec)
	Log(reply)
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
		roleUsers = append(roleUsers, roleHelper(guild, v, b.respec)...)
	}

	for _, v := range roleUsers {
		b.users[v.ID] = v
		b.userStatus[v.ID] = false
	}
}

func roleHelper(guild *discordgo.Guild, roleID string, respecNeeded int) (users []*discordgo.User) {
	members := guild.Members
	for _, v := range members {
		for _, role := range v.Roles {
			if roleID == role {
				if userCanBet(v.User, respecNeeded) {
					users = append(users, v.User)
					break
				}
			}
		}
	}
	return
}

func userCanBet(user *discordgo.User, respecNeeded int) bool {
	if available := dbGetUserRespec(user); available >= respecNeeded || !isBot(user) {
		return true
	}
	return false
}

// goroutine to run an active bet
// this handles all the winnin' 'n stuff
func betEngage(c chan betMessage, b *Bet, mux *sync.Mutex) {
	activeBetEmbed(b)

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
		default:
		}

		if !b.started && !b.open {
			if checkBetReady(b) {
				startBet(b)
			}
			activeBetEmbed(b)
		} else if b.started {
			var ok bool
			if ok = checkWinner(b); ok {
				break Loop
			} else {
				activeBetEmbed(b)
			}
		} else {
			activeBetEmbed(b)
		}
		mux.Unlock()
	}

	if b.winner != nil && len(b.users) > 1 {
		betWon(b)
		Log(fmt.Sprintf("Bet ended. %v won %v respec", b.winner, b.totalRespec-b.respec))
		winnerCard(b)
		dbRecordBet(b)
	} else {
		cancelBet(b)
		deleteEmbed(b)
	}

	delete(allBets, b.channelID)
	mux.Unlock()
}

func callBet(b *Bet, user *discordgo.User) {
	if b.userStatus[user.ID] {
		return
	}
	b.userStatus[user.ID] = true
	b.users[user.ID] = user
	b.totalRespec += b.respec

	Log(fmt.Sprintf("%+v called", user.String()))

	addRespec(b.guildID, user, -b.respec)
}

func loseBet(b *Bet, user *discordgo.User) {
	if !b.userStatus[user.ID] {
		return
	}
	b.userStatus[user.ID] = false

	Log(fmt.Sprintf("%+v lost", user.String()))
}

func dropOut(b *Bet, user *discordgo.User) {
	if !b.userStatus[user.ID] {
		return
	}
	b.userStatus[user.ID] = false
	b.totalRespec -= b.respec

	Log(fmt.Sprintf("%+v dropped out", user.String()))

	addRespec(b.guildID, user, b.respec)
}

func betWon(b *Bet) {
	addRespec(b.guildID, b.winner, b.totalRespec)

	for _, v := range b.users {
		if v.ID != b.winner.ID {
			addRespec(b.guildID, v, -b.respec)
		}
	}
}

func cancelBet(b *Bet) {
	if b.cancelled {
		return
	}
	for k, v := range b.userStatus {
		if v {
			addRespec(b.guildID, b.users[k], b.respec)
			b.userStatus[k] = false
		}
		delete(b.userStatus, k)
	}

	reply := fmt.Sprintf("Bet Cancelled, respec refunded")

	b.started = true
	b.cancelled = true
	SendReply(b.channelID, reply)
	Log(reply)
}

func startBet(b *Bet) {
	if b.started {
		return
	}
	b.started = true
	count := 0
	for k, v := range b.userStatus {
		if !v {
			delete(b.userStatus, k)
			delete(b.users, k)
		} else {
			count++
		}
	}
	if count < 2 {
		b.state <- betMessage{user: nil, arg: "cancel"}
		reply := "Not enough users entered the bet"
		SendReply(b.channelID, reply)
		Log(reply)
		return
	}
	go betEndTimer(b.state)
	b.endTime = b.time.Add(time.Minute * 30)
	timeStamp := fmt.Sprintf(b.endTime.Format("15:04:05"))
	reply := fmt.Sprintf("Bet started: Total pot:%v Must end before %v.", b.totalRespec, timeStamp)

	Log(reply)
}

func checkBetReady(b *Bet) bool {
	for _, v := range b.userStatus {
		if !v {
			return false
		}
	}
	return true
}

func betEndTimer(c chan betMessage) {
	timer := time.NewTicker(time.Minute * 30)
	<-timer.C
	c <- betMessage{user: nil, arg: "cancel"}
}

// check if only one user has not lost the bet
func checkWinner(b *Bet) (won bool) {
	count := 0
	var winnerID string
	for k, v := range b.userStatus {
		if v {
			winnerID = k
			count++
		}
		if count > 1 {
			return false
		}
	}
	if count == 0 {
		return true
	}
	b.winner = b.users[winnerID]
	return true
}

func activeBetEmbed(b *Bet) {
	embed := new(discordgo.MessageEmbed)
	embed.Footer = new(discordgo.MessageEmbedFooter)
	embed.Thumbnail = new(discordgo.MessageEmbedThumbnail)
	var title string

	if b.started {
		title = fmt.Sprintf("Bet (%v) Started", b.respec)
		embed.Footer.Text = fmt.Sprintf("Bet ends at %v", b.endTime.Format("15:04:05"))

	} else {
		title = fmt.Sprintf("Bet (%v) Not Started", b.respec)
		if b.open {
			title += " (ANYONE CAN JOIN)"
		}
		embed.Footer.Text = fmt.Sprintf("Bet starts at %v", b.time.Add(time.Minute*2).Format("15:04:05"))
	}

	embed.Title = title
	embed.Description = fmt.Sprintf("Total Pot: %v", b.totalRespec)
	embed.URL = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	embed.Thumbnail.URL = "https://i.imgur.com/aUeMzFC.png"
	embed.Type = "rich"

	for k, v := range b.users {
		field := new(discordgo.MessageEmbedField)
		field.Inline = true
		field.Name = v.Username
		if b.userStatus[k] {
			field.Value = "in"
		} else {
			field.Value = "out"
		}
		embed.Fields = append(embed.Fields, field)
	}

	msg := SendEmbed(b.channelID, embed)

	if b.announcement != nil {
		deleteEmbed(b)
	}

	b.announcement = msg
}

func winnerCard(b *Bet) {
	embed := new(discordgo.MessageEmbed)
	embed.Footer = new(discordgo.MessageEmbedFooter)
	embed.Thumbnail = new(discordgo.MessageEmbedThumbnail)

	title := fmt.Sprintf("%v won %v respec", b.winner.Username, b.totalRespec-b.respec)

	embed.Title = title
	embed.Description = fmt.Sprintf("Total Pot: %v", b.totalRespec)
	embed.URL = "https://www.youtube.com/watch?v=1EKTw50Uf8M"
	embed.Thumbnail.URL = "https://i.imgur.com/5Gwne2N.png"
	embed.Type = "rich"
	embed.Footer.Text = fmt.Sprintf("Bet ended at %v", time.Now().In(location).Format("15:04:05"))

	for k, v := range b.users {
		field := new(discordgo.MessageEmbedField)
		field.Inline = true
		field.Name = v.Username
		if b.userStatus[k] {
			field.Value = "WINNER"
		} else {
			field.Value = "LOSER"
		}
		embed.Fields = append(embed.Fields, field)
	}

	msg := SendEmbed(b.channelID, embed)

	if b.announcement != nil {
		deleteEmbed(b)
	}

	b.announcement = msg
}

func deleteEmbed(b *Bet) {
	DiscordSession.ChannelMessageDelete(b.announcement.ChannelID, b.announcement.ID)
}

// multiple pot winners?
