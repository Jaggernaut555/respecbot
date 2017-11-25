package bet

import (
	"fmt"
	"sync"
	"time"

	"github.com/Jaggernaut555/respecbot/db"
	"github.com/Jaggernaut555/respecbot/logging"
	"github.com/Jaggernaut555/respecbot/rate"
	"github.com/Jaggernaut555/respecbot/state"
	"github.com/bwmarrin/discordgo"
)

type BetMessage struct {
	User *discordgo.User
	Arg  string
	Bet  int
	Odds float64
}

type Bet struct {
	TotalRespec  int
	ManualBet    bool
	Started      bool
	AgainstHouse bool
	Open         bool
	Ended        bool
	cancelled    bool
	AuthorID     string
	UserBet      map[string]int
	UserEarnings map[string]int
	UserOdds     map[string]float64
	UserStatus   map[string]int
	Users        map[string]*discordgo.User
	State        chan BetMessage
	Time         time.Time
	EndTime      time.Time
	ChannelID    string
	GuildID      string
	Annoucement  *discordgo.Message
}

const (
	Won     = iota
	Playing = iota
	Lost    = iota
)

var (
	location *time.Location
)

func init() {
	var err error
	location, err = time.LoadLocation("America/Vancouver")
	if err != nil {
		panic(err)
	}
}

func StartBetTimer(c chan BetMessage) {
	timer := time.NewTicker(time.Minute * 2)
	<-timer.C
	c <- BetMessage{User: nil, Arg: "start"}
}

func AppendRoles(message *discordgo.Message, b *Bet) {
	channel, err := state.Session.Channel(message.ChannelID)
	if err != nil {
		panic(err)
	}
	mentionedRoles := message.MentionRoles
	var roleUsers []*discordgo.User

	guild, _ := state.Session.Guild(channel.GuildID)

	for _, v := range mentionedRoles {
		roleUsers = append(roleUsers, roleHelper(guild, v)...)
	}

	for _, v := range roleUsers {
		b.Users[v.ID] = v
		b.UserStatus[v.ID] = Lost
	}
}

func roleHelper(guild *discordgo.Guild, roleID string) (Users []*discordgo.User) {
	members := guild.Members
	for _, v := range members {
		for _, role := range v.Roles {
			if roleID == role {
				Users = append(Users, v.User)
				break
			}
		}
	}
	return
}

func UserCanBet(user *discordgo.User, respecNeeded int) bool {
	if available := db.GetUserRespec(user); available >= respecNeeded || !user.Bot {
		return true
	}
	return false
}

// goroutine to run an active bet
// this handles all the winnin' 'n stuff
func BetEngage(c chan BetMessage, b *Bet, mux *sync.Mutex) {
Loop:
	for i := range c {
		mux.Lock()
		switch i.Arg {
		case "call":
			callBet(b, i)
		case "lose":
			loseBet(b, i.User)
		case "drop":
			dropOut(b, i.User)
		case "start":
			startBet(b)
		case "cancel":
			cancelBet(b)
		case "end":
			b.Ended = true
		default:
		}

		if !b.Started && !b.Open && !b.AgainstHouse && !b.Ended {
			if checkBetReady(b) {
				startBet(b)
			}
		} else if b.Started && !b.AgainstHouse && !b.Ended {
			b.Ended = checkWinner(b)
		} else if b.Started && b.AgainstHouse && !b.Ended {
			//stuff goes here
		}

		if b.Ended || b.cancelled {
			break Loop
		} else {
			activeBetEmbed(b)
		}
		mux.Unlock()
	}

	if b.Started && b.Ended && !b.cancelled {
		betWon(b)
		recordBet(b)
	} else {
		cancelBet(b)
		deleteEmbed(b)
	}

	mux.Unlock()
}

func callBet(b *Bet, msg BetMessage) {
	if b.UserStatus[msg.User.ID] == Playing {
		return
	}
	b.UserStatus[msg.User.ID] = Playing
	b.Users[msg.User.ID] = msg.User
	b.UserBet[msg.User.ID] = msg.Bet
	b.UserOdds[msg.User.ID] = msg.Odds
	b.TotalRespec += msg.Bet

	logging.Log(fmt.Sprintf("%+v called", msg.User.String()))

	rate.AddRespec(b.GuildID, msg.User, -msg.Bet)
}

func loseBet(b *Bet, user *discordgo.User) {
	if b.UserStatus[user.ID] == Lost {
		return
	}
	b.UserStatus[user.ID] = Lost

	logging.Log(fmt.Sprintf("%+v Lost", user.String()))
}

func dropOut(b *Bet, user *discordgo.User) {
	if b.UserStatus[user.ID] != Playing {
		return
	}
	b.UserStatus[user.ID] = Lost
	b.TotalRespec -= b.UserBet[user.ID]

	logging.Log(fmt.Sprintf("%+v dropped out", user.String()))

	rate.AddRespec(b.GuildID, user, b.UserBet[user.ID])
}

func betWon(b *Bet) {
	logging.Log("Bet Ended")
	if !b.AgainstHouse {
		for k, v := range b.UserStatus {
			if v != Lost {
				b.UserEarnings[k] = b.TotalRespec
				rate.AddRespec(b.GuildID, b.Users[k], b.TotalRespec)
				logging.Log(fmt.Sprintf("%v Won %v respec", b.Users[k].Username, b.TotalRespec))
				b.UserStatus[k] = Won
			}
		}
	} else {
		for k, v := range b.UserStatus {
			if v != Lost {
				b.UserEarnings[k] = int(float64(b.UserBet[k]) * b.UserOdds[k])
				rate.AddRespec(b.GuildID, b.Users[k], b.UserEarnings[k])
				logging.Log(fmt.Sprintf("%v Won %v respec", b.Users[k].Username, b.UserEarnings[k]))
				b.UserStatus[k] = Won
			}
		}
	}
	winnerCard(b)
}

func cancelBet(b *Bet) {
	if b.cancelled {
		return
	}
	for k, v := range b.UserStatus {
		if v == Playing {
			rate.AddRespec(b.GuildID, b.Users[k], b.UserBet[k])
		}
		delete(b.UserStatus, k)
	}

	reply := fmt.Sprintf("Bet Cancelled, respec refunded")

	b.Started = true
	b.cancelled = true
	state.SendReply(b.ChannelID, reply)
	logging.Log(reply)
}

func startBet(b *Bet) {
	if b.Started {
		return
	}
	b.Started = true
	count := 0
	for k, v := range b.UserStatus {
		if v != Playing {
			delete(b.UserStatus, k)
			delete(b.Users, k)
		} else {
			count++
		}
	}
	if count < 2 {
		b.cancelled = true
		b.Ended = true
		reply := "Not enough Users entered the bet"
		state.SendReply(b.ChannelID, reply)
		logging.Log(reply)
		return
	}
	go betEndTimer(b.State)
	b.EndTime = b.Time.Add(time.Minute * 30)
	timeStamp := fmt.Sprintf(b.EndTime.Format("15:04:05"))
	reply := fmt.Sprintf("Bet Started: Total pot:%v Must end before %v.", b.TotalRespec, timeStamp)

	logging.Log(reply)
}

func checkBetReady(b *Bet) bool {
	for _, v := range b.UserStatus {
		if v != Playing {
			return false
		}
	}
	return true
}

func betEndTimer(c chan BetMessage) {
	timer := time.NewTicker(time.Minute * 30)
	<-timer.C
	c <- BetMessage{User: nil, Arg: "end"}
}

// check if only one user has not Lost the bet
func checkWinner(b *Bet) (Won bool) {
	count := 0
	for _, v := range b.UserStatus {
		if v == Playing {
			count++
		}
		if count > 1 {
			return false
		}
	}
	if count == 0 {
		return true
	}
	return true
}

func activeBetEmbed(b *Bet) {
	embed := new(discordgo.MessageEmbed)
	embed.Footer = new(discordgo.MessageEmbedFooter)
	embed.Thumbnail = new(discordgo.MessageEmbedThumbnail)
	var title string

	if b.Started {
		title = fmt.Sprintf("Bet Started")
		embed.Footer.Text = fmt.Sprintf("Bet ends at %v", b.EndTime.Format("15:04:05"))

	} else {
		title = fmt.Sprintf("Bet Not Started")
		if b.Open {
			title += " (ANYONE CAN JOIN)"
		}
		embed.Footer.Text = fmt.Sprintf("Bet starts at %v", b.Time.Add(time.Minute*2).Format("15:04:05"))
	}

	embed.Title = title
	embed.Description = fmt.Sprintf("Total Pot: %v", b.TotalRespec)
	embed.URL = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	embed.Thumbnail.URL = "https://i.imgur.com/aUeMzFC.png"
	embed.Type = "rich"

	for k, v := range b.Users {
		field := new(discordgo.MessageEmbedField)
		field.Inline = true
		field.Name = v.Username
		if b.UserStatus[k] == Playing {
			field.Value = fmt.Sprintf("In (%v)", b.UserBet[k])
		} else {
			field.Value = "out"
		}
		embed.Fields = append(embed.Fields, field)
	}

	msg := state.SendEmbed(b.ChannelID, embed)

	if b.Annoucement != nil {
		deleteEmbed(b)
	}

	b.Annoucement = msg
}

func winnerCard(b *Bet) {
	embed := new(discordgo.MessageEmbed)
	embed.Footer = new(discordgo.MessageEmbedFooter)
	embed.Thumbnail = new(discordgo.MessageEmbedThumbnail)

	title := fmt.Sprintf("Bet Ended")

	embed.Title = title
	embed.Description = fmt.Sprintf("Total Pot: %v", b.TotalRespec)
	embed.URL = "https://www.youtube.com/watch?v=1EKTw50Uf8M"
	embed.Thumbnail.URL = "https://i.imgur.com/5Gwne2N.png"
	embed.Type = "rich"
	embed.Footer.Text = fmt.Sprintf("Bet Ended at %v", time.Now().In(location).Format("15:04:05"))

	for k, v := range b.Users {
		field := new(discordgo.MessageEmbedField)
		field.Inline = true
		field.Name = v.Username
		if b.UserStatus[k] == Won {
			field.Value = fmt.Sprintf("Won %v", b.UserEarnings[k])
		} else {
			field.Value = "LOSER"
		}
		embed.Fields = append(embed.Fields, field)
	}

	msg := state.SendEmbed(b.ChannelID, embed)

	if b.Annoucement != nil {
		deleteEmbed(b)
	}

	b.Annoucement = msg
}

func deleteEmbed(b *Bet) {
	state.Session.ChannelMessageDelete(b.Annoucement.ChannelID, b.Annoucement.ID)
}

func recordBet(b *Bet) {
	var bet db.DBBet

	bet.ChannelID = b.ChannelID
	bet.Pot = b.TotalRespec
	bet.StarterID = b.AuthorID
	bet.Time = b.Time

	var Users []db.BetUsers

	for k := range b.UserStatus {
		Users = append(Users, db.BetUsers{UserID: k, Bet: b.UserBet[k], Won: b.UserEarnings[k]})
	}

	db.RecordBet(bet, Users)
}
