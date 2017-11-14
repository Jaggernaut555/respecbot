package main

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/bwmarrin/discordgo"
)

type pair struct {
	Key   string
	Value int
}

type pairList []pair

const (
	correctUsageValue = 2
	reactionValue     = 2
	mentionValue      = 3
)

var (
	userLastRespec  map[string]time.Time
	userLastMention map[string]time.Time

	totalRespec int
)

func InitRatings() {
	userRatings := make(map[string]int)
	userLastRespec = make(map[string]time.Time)
	userLastMention = make(map[string]time.Time)

	rand.Seed(time.Now().Unix())

	dbLoadRespec(&userRatings)
	dbGetLastRespec(&userLastRespec)
	dbGetLastMention(&userLastMention)

	fmt.Println("loaded", len(userRatings), "ratings")

	totalRespec = dbGetTotalRespec()
}

func isALoser(guildID string, user *discordgo.User) {
	roles, _ := DiscordSession.GuildRoles(guildID)
	var role *discordgo.Role
	for _, v := range roles {
		if v.Name == "Losers" {
			role = v
			break
		}
	}
	if role == nil {
		return
	}
	DiscordSession.GuildMemberRoleAdd(guildID, user.ID, role.ID)
}

func isNotALoser(guildID string, user *discordgo.User) {
	roles, _ := DiscordSession.GuildRoles(guildID)
	var role *discordgo.Role
	for _, v := range roles {
		if v.Name == "Losers" {
			role = v
			break
		}
	}
	if role == nil {
		return
	}
	DiscordSession.GuildMemberRoleRemove(guildID, user.ID, role.ID)
}

func addRespec(guildId string, user *discordgo.User, rating int) {
	temp := addRespecHelp(user, rating)

	if temp < 0 {
		isALoser(guildId, user)
	} else if temp > 0 {
		isNotALoser(guildId, user)
	}
}

func addRespecHelp(user *discordgo.User, rating int) int {
	// abs(userRating) / abs(totalRespec)
	userRespec := dbGetUserRespec(user)
	newRespec := rating
	if totalRespec != 0 && userRespec != 0 {
		temp := math.Abs(float64(userRespec)) * math.Log(1+math.Abs(float64(userRespec))) / math.Abs(float64(totalRespec))
		//var temp = math.Abs(float64(userRespec)) / math.Abs(float64(totalRespec))
		if temp > 0.15 {
			temp = 0.15
		} else if temp < 0.01 {
			temp = 0.01
		}
		if rand.Float64() < temp {
			newRespec = -newRespec
		}
	}

	totalRespec += newRespec
	fmt.Printf("%v %+d respec\n", user, newRespec)

	dbGainRespec(user, newRespec)

	if userRespec >= 0 && userRespec+newRespec < 0 {
		return -1
	} else if userRespec < 0 && userRespec+newRespec >= 0 {
		return 1
	}

	return 0
}

// give respec by reacting
func RespecReactionAdd(reaction *discordgo.MessageReactionAdd) {
	user, _ := DiscordSession.User(reaction.UserID)
	message, _ := DiscordSession.ChannelMessage(reaction.ChannelID, reaction.MessageID)
	author := message.Author
	timeStamp, _ := message.Timestamp.Parse()

	channel, _ := DiscordSession.Channel(message.ChannelID)
	guild, _ := DiscordSession.Guild(channel.GuildID)

	fmt.Printf("%v got a reaction from %v\n", author, user)
	if user.ID == author.ID {
		addRespec(guild.ID, author, -reactionValue)
	} else {
		addRespec(guild.ID, author, reactionValue)
	}

	dbReactionAdd(author, reaction, timeStamp)
}

// no fuckin gaming the system
func RespecReactionRemove(reaction *discordgo.MessageReactionRemove) {
	user, _ := DiscordSession.User(reaction.UserID)
	message, _ := DiscordSession.ChannelMessage(reaction.ChannelID, reaction.MessageID)
	author := message.Author
	timeStamp, _ := message.Timestamp.Parse()

	channel, _ := DiscordSession.Channel(message.ChannelID)
	guild, _ := DiscordSession.Guild(channel.GuildID)

	fmt.Printf("%v lost a reaction\n", author)
	addRespec(guild.ID, author, -reactionValue)
	fmt.Printf("%v removed a reaction\n", user)
	addRespec(guild.ID, user, -reactionValue)
	dbReactionRemove(author, reaction, timeStamp)
}

// evaluate messages
func RespecMessage(incomingMessage *discordgo.MessageCreate) {
	message := incomingMessage.Message
	author := message.Author
	timeStamp, _ := message.Timestamp.Parse()
	numRespec := applyRules(author, message)

	channel, err := DiscordSession.Channel(message.ChannelID)
	if err != nil {
		return
	}
	guild, err := DiscordSession.Guild(channel.GuildID)
	if err != nil {
		return
	}

	fmt.Printf("%v: %v\n", author, message.ContentWithMentionsReplaced())

	numRespec += respecMentions(guild.ID, author, message)

	addRespec(guild.ID, author, numRespec)

	dbNewMessage(author, incomingMessage, numRespec, timeStamp)
}

// if someone talkin to you you aight
func respecMentions(guildID string, author *discordgo.User, message *discordgo.Message) (respec int) {
	users := message.Mentions
	timeStamp, _ := message.Timestamp.Parse()

	for _, v := range users {
		if v.ID == author.ID {
			fmt.Println(author, "mentioned self")
			dbMention(author, v, message, -mentionValue, timeStamp)
			respec -= mentionValue
		} else if !canMention(v, timeStamp) {
			fmt.Println(v, "mentioned by", author, "too soon since last mention")
			dbMention(author, v, message, 0, timeStamp)
		} else {
			userLastMention[v.String()] = timeStamp
			fmt.Println(v, "mentioned by", author)
			addRespec(guildID, v, mentionValue)
			dbMention(author, v, message, mentionValue, timeStamp)
		}
	}

	return
}

func canMention(user *discordgo.User, timeGiven time.Time) bool {
	if oldTime, ok := userLastMention[user.String()]; ok {
		timeDelta := timeGiven.Sub(oldTime)
		if timeDelta.Minutes() < 5 {
			return false
		} else {
			return true
		}
	}
	return true
}

func canGiveRespec(user *discordgo.User, timeGiven time.Time) bool {
	if oldTime, ok := userLastRespec[user.String()]; ok {
		timeDelta := timeGiven.Sub(oldTime)
		if timeDelta.Minutes() < 30 {
			return false
		}
	}
	return true
}

// if you try to respec yourself fuck you
func validGiveRespec(author *discordgo.User, users []*discordgo.User, timeGiven time.Time) bool {
	if !canGiveRespec(author, timeGiven) {
		return false
	}
	for _, v := range users {
		if author.ID == v.ID {
			return false
		}
	}
	return true
}

// gif someone respec
func GiveRespec(message *discordgo.MessageCreate, positive bool) {
	mentions := message.Message.Mentions
	author := message.Message.Author
	timeStamp, _ := message.Timestamp.Parse()
	respec := dbGetUserRespec(author)
	numRespec := 0

	channel, _ := DiscordSession.Channel(message.ChannelID)
	guild, _ := DiscordSession.Guild(channel.GuildID)

	if respec <= 0 {
		numRespec = 2
	} else {
		numRespec = respec / 10
		if numRespec < 5 {
			numRespec = 5
		} else if numRespec > 25 {
			numRespec = 25
		}
	}

	if !positive {
		numRespec = -numRespec
	}

	// lose respec if you use it wrong
	if len(mentions) < 1 || !validGiveRespec(author, mentions, timeStamp) {
		fmt.Println(author, "Used respec wrong")
		numRespec *= 2
		addRespec(guild.ID, author, -numRespec)
		dbGiveRespec(author, author, -numRespec, timeStamp)
		mentions = nil
	} else {
		addRespec(guild.ID, author, correctUsageValue)
		dbGiveRespec(author, author, correctUsageValue, timeStamp)
	}

	for _, v := range mentions {
		fmt.Println(author, " gave respec to ", v)
		addRespec(guild.ID, v, numRespec)
		dbGiveRespec(author, v, numRespec, timeStamp)
	}

	userLastRespec[author.String()] = timeStamp
}

// get all da users in list
func getRatingsLists() (users pairList) {
	temp := make(map[string]int)
	dbLoadRespec(&temp)

	for k, v := range temp {
		users = append(users, pair{k, v})
	}

	return
}

// show 10 most RESPEC peep
func GetRespec() (Leaderboard string, negativeUsers []string) {
	var buf bytes.Buffer
	negativeUsers = make([]string, 0)
	users := getRatingsLists()

	sort.Sort(sort.Reverse(users))

	var padding = 3
	w := new(tabwriter.Writer)
	w.Init(&buf, 0, 0, padding, ' ', 0)
	for k, v := range users {
		if k > 15 {
			break
		}
		if v.Value >= 0 {
			fmt.Fprintf(w, "%v\t%v\t\n", v.Key, v.Value)
		} else {
			negativeUsers = append(negativeUsers, v.Key)
		}
	}
	w.Flush()
	Leaderboard = fmt.Sprintf("%v", buf.String())
	sort.Strings(negativeUsers)
	return
}

func (p pairList) Len() int           { return len(p) }
func (p pairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p pairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
