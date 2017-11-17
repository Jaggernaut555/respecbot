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
	respec      int
	totalRespec int
	started     bool
	open        bool
	author      *discordgo.User
	userStatus  map[*discordgo.User]bool
	state       chan betMessage
	time        time.Time
	channelID   string
}

var (
	allBets  map[string]*Bet
	betMuxes map[string]*sync.Mutex
)

func InitBets() {
	allBets = make(map[string]*Bet)
	betMuxes = make(map[string]*sync.Mutex)
}

func bet(message *discordgo.MessageCreate, args []string) {
	/*
		format
		bet 50 @user1 @user2 ... (must have enough score, cap of 50?)
		one to many users in pot may accept (must have enough score)
		after at least one user has accepted, bet is active
		make sure user doesn't mention themself

		maybe just 'bet 50' and anybody can accept into the pool?
		One active bet per channel
	*/
	channelID := message.ChannelID
	author := message.Author
	mentions := message.Mentions

	if len(args) < 2 {
		fmt.Println("Bet Used Wrong")
		return
	}

	mux, ok := betMuxes[channelID]

	if !ok {
		mux = new(sync.Mutex)
		betMuxes[channelID] = mux
	}

	mux.Lock()

	if b, ok := allBets[channelID]; ok {
		// bet exists, check if user is active or able to join

		userStatus, ok := b.userStatus[author]

		if !ok {
			ok = b.open
		}

		switch strings.ToLower(args[1]) {

		// begin bet with current active users
		case "start":
			if author == b.author && !b.started {
				b.state <- betMessage{user: author, arg: "start"}
			}

		// cannot lose if not active
		case "lose":
			if userStatus && ok && b.started {
				b.state <- betMessage{user: author, arg: "lose"}
			}

		// validate user can accept
		case "accept":
			if !userStatus && ok && !b.started {
				b.state <- betMessage{user: author, arg: "accept"}
			}

		// cannot cancel started bet
		case "cancel":
			if author == b.author && !b.started {
				close(b.state)
			}

		default:
			reply := fmt.Sprintf("Not a valid command for active bet, %v", author.Mention())
			SendReply(channelID, reply)
		}
	} else {
		// bet does not exist, check if valid bet then create it
		// validate user has enough respec to create bet
		num, err := strconv.Atoi(args[1])
		if err != nil {
			reply := fmt.Sprintf("%v did not specify bet amount", author.Mention())
			SendReply(channelID, reply)
			return
		}

		var b Bet
		b.userStatus = make(map[*discordgo.User]bool)
		b.respec = num
		b.totalRespec = num
		b.state = make(chan betMessage)
		b.open = message.MentionEveryone
		b.time = time.Now()

		if len(args) == 2 {
			b.open = true
		} else {
			// check if role mentioned
			for _, v := range mentions {
				b.userStatus[v] = true
			}
			b.userStatus[author] = true
		}

		if mux != betMuxes[channelID] {
			mux.Unlock()
			return
		}

		allBets[channelID] = &b

		go betEngage(b.state, &b, mux)

		reply := fmt.Sprintf("%v started a bet for %v", author.Mention(), b.respec)
		SendReply(channelID, reply)
		reply = fmt.Sprintf("%v started a bet for %v", author.String(), b.respec)
		fmt.Println(reply)
	}

	mux.Unlock()
}

// goroutine to run an active bet
func betEngage(c chan betMessage, b *Bet, mux *sync.Mutex) {
	var winner *discordgo.User
	for i := range c {
		mux.Lock()
		switch i.arg {
		case "accept":
			b.userStatus[i.user] = true
			fmt.Printf("%+v\n", i)
		case "lose":
			b.userStatus[i.user] = false
			fmt.Printf("%+v\n", i)
		case "start":
			fmt.Println("Bet Started")
			startBet(b)
		case "cancel":
			fmt.Println("Bet Cancelled")
			break
		}

		if !b.started {
			if checkBetReady(b.userStatus) {
				startBet(b)
			}
		} else {
			var ok bool
			if winner, ok = checkWinner(b.userStatus); ok {
				break
			}
		}
		mux.Unlock()
	}
	mux.Lock()
	delete(allBets, b.channelID)
	mux.Unlock()

	var reply string
	if winner != nil {
		reply = fmt.Sprintf("%v won %v respec", winner.Mention(), b.totalRespec)
		fmt.Printf("%v won %v respec\n", winner.String(), b.totalRespec)
	} else {
		reply = fmt.Sprintf("No winner, repsec refunded")
		fmt.Println(reply)
	}

	SendReply(b.channelID, reply)
	fmt.Println("Bet ended")
}

func startBet(b *Bet) {
	if b.started {
		return
	}
	for k, v := range b.userStatus {
		if !v {
			delete(b.userStatus, k)
		}
	}
	b.started = true
}

// check if only one user has not lost the bet
func checkWinner(users map[*discordgo.User]bool) (winner *discordgo.User, won bool) {
	count := 0
	for k, v := range users {
		if v {
			winner = k
			count++
		}
		if count > 1 {
			return nil, false
		}
	}
	return winner, true
}

func checkBetReady(users map[*discordgo.User]bool) bool {
	for _, v := range users {
		if !v {
			return false
		}
	}
	return true
}

// make @role to return a list of all users of that role
// make timer to auto start bet after x time
// make timer to end bet after x time
// multiple pot winners?
