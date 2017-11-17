package main

import (
	"bytes"
	"fmt"
	"log"
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
	totalRespec int
)

func InitRatings() {
	userRatings := make(map[string]int)

	rand.Seed(time.Now().Unix())

	dbLoadRespec(&userRatings)

	log.Println("loaded", len(userRatings), "ratings")

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

func addRespec(guildID string, user *discordgo.User, rating int) {
	temp := addRespecHelp(user, rating)

	if temp < 0 {
		isALoser(guildID, user)
	} else if temp > 0 {
		isNotALoser(guildID, user)
	}
}

func addRespecHelp(user *discordgo.User, rating int) int {
	// abs(userRating) / abs(totalRespec)
	userRespec := dbGetUserRespec(user)
	newRespec := rating
	if totalRespec != 0 && userRespec != 0 {
		temp := math.Abs(float64(userRespec)) * math.Log(1+math.Abs(float64(userRespec))) / math.Abs(float64(totalRespec))
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
	log.Printf("%v %+d respec\n", user, newRespec)

	dbGainRespec(user, newRespec)

	if userRespec >= 0 && userRespec+newRespec < 0 {
		return -1
	} else if userRespec < 0 && userRespec+newRespec >= 0 {
		return 1
	}

	return 0
}

// evaluate messages
func RespecMessage(message *discordgo.Message) {
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

	log.Printf("%v: %v\n", author, message.ContentWithMentionsReplaced())

	numRespec += respecMentions(guild.ID, author, message)

	addRespec(guild.ID, author, numRespec)

	dbNewMessage(author, message, numRespec, timeStamp)
}

func messageExistsInDB(messageID string) bool {
	return dbMessageExists(messageID)
}

// if someone talkin to you you aight
func respecMentions(guildID string, author *discordgo.User, message *discordgo.Message) (respec int) {
	users := message.Mentions
	timeStamp, _ := message.Timestamp.Parse()

	for _, v := range users {
		if v.ID == author.ID {
			log.Println(author, "mentioned self")
			dbMention(author, v, message, -mentionValue, timeStamp)
			respec -= mentionValue
		} else if !canMention(v, timeStamp) {
			log.Println(v, "mentioned by", author, "too soon since last mention")
			dbMention(author, v, message, 0, timeStamp)
		} else {
			log.Println(v, "mentioned by", author)
			addRespec(guildID, v, mentionValue)
			dbMention(author, v, message, mentionValue, timeStamp)
		}
	}

	return
}

func canMention(user *discordgo.User, timeGiven time.Time) bool {
	if oldTime, ok := dbGetUserLastMentionedTime(user.String()); ok {
		timeDelta := timeGiven.Sub(oldTime)
		if timeDelta.Minutes() < 5 {
			return false
		} else {
			return true
		}
	}
	return true
}

// gif someone respec
func GiveRespec(message *discordgo.Message, positive bool) {
	mentions := message.Mentions
	author := message.Author
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
		log.Println(author, "Used respec wrong")
		numRespec *= 2
		addRespec(guild.ID, author, -numRespec)
		dbGiveRespec(author, author, -numRespec, timeStamp)
		mentions = nil
	} else {
		addRespec(guild.ID, author, correctUsageValue)
		dbGiveRespec(author, author, correctUsageValue, timeStamp)
	}

	for _, v := range mentions {
		log.Println(author, " gave respec to ", v)
		addRespec(guild.ID, v, numRespec)
		dbGiveRespec(author, v, numRespec, timeStamp)
	}
}

func canGiveRespec(user *discordgo.User, timeGiven time.Time) bool {
	if oldTime, ok := dbGetUserLastRespecTime(user.String()); ok {
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

func RespecReaction(reaction *discordgo.MessageReaction, added bool) {
	if !messageExistsInDB(reaction.MessageID) {
		return
	}

	if added {
		RespecReactionAdd(reaction)
	} else {
		RespecReactionRemove(reaction)
	}
}

// give respec by reacting
func RespecReactionAdd(reaction *discordgo.MessageReaction) {
	user, _ := DiscordSession.User(reaction.UserID)
	message, _ := DiscordSession.ChannelMessage(reaction.ChannelID, reaction.MessageID)
	author := message.Author
	timeStamp := time.Now()

	channel, _ := DiscordSession.Channel(message.ChannelID)
	guild, _ := DiscordSession.Guild(channel.GuildID)

	if user.ID == author.ID {
		addRespec(guild.ID, author, -reactionValue)
	} else if validReactionAdd(user.String(), author.String(), timeStamp) {
		addRespec(guild.ID, author, reactionValue)
	}

	log.Printf("%v got a reaction from %v\n", author, user)

	dbReactionAdd(user, reaction, timeStamp)
}

func validReactionAdd(GiverID, ReceiverID string, timeGiven time.Time) bool {
	if oldTime, ok := dbGetUserLastReactionAddTime(GiverID, ReceiverID); ok {
		timeDelta := timeGiven.Sub(oldTime)
		if timeDelta.Minutes() < 5 {
			return false
		} else {
			return true
		}
	}
	return true
}

// no fuckin gaming the system
func RespecReactionRemove(reaction *discordgo.MessageReaction) {
	user, _ := DiscordSession.User(reaction.UserID)
	message, _ := DiscordSession.ChannelMessage(reaction.ChannelID, reaction.MessageID)
	author := message.Author
	timeStamp := time.Now()

	channel, _ := DiscordSession.Channel(message.ChannelID)
	guild, _ := DiscordSession.Guild(channel.GuildID)

	if author.ID == user.ID {
		addRespec(guild.ID, author, -reactionValue)
	} else if validReactionRemove(user.String(), author.String(), timeStamp) {
		addRespec(guild.ID, author, -reactionValue)
	}

	log.Printf("%v lost a reaction\n", author)

	log.Printf("%v removed a reaction\n", user)
	addRespec(guild.ID, user, -reactionValue)
	dbReactionRemove(user, reaction, timeStamp)
}

func validReactionRemove(GiverID, ReceiverID string, timeGiven time.Time) bool {
	if oldTime, ok := dbGetUserLastReactionRemoveTime(GiverID, ReceiverID); ok {
		timeDelta := timeGiven.Sub(oldTime)
		if timeDelta.Minutes() < 5 {
			return false
		} else {
			return true
		}
	}
	return true
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
