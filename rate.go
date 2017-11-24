package main

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"reflect"
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
	chatLimiter       = 111
)

const (
	badChange  = iota
	noChange   = iota
	goodChange = iota
)

var (
	totalRespec       int
	supremeRuler      string
	rulingClass       map[string]bool
	loserRoleID       map[string]string
	rulerRoleID       map[string]string
	rulingClassRoleID map[string]string
)

func InitRatings() {
	userRatings := make(map[string]int)
	rulingClass = make(map[string]bool)
	loserRoleID = make(map[string]string)
	rulerRoleID = make(map[string]string)
	rulingClassRoleID = make(map[string]string)

	rand.Seed(time.Now().Unix())

	dbLoadRespec(&userRatings)

	Log(fmt.Sprintf("loaded %v ratings", len(userRatings)))

	totalRespec = dbGetTotalRespec()
}

func initLosers(guildID string) {
	guild, err := DiscordSession.Guild(guildID)
	if err != nil {
		panic(err)
	}

	loserRoleID[guildID] = getRoleID(guildID, "Losers")

	for _, v := range guild.Members {
		if isBot(v.User) {
			continue
		}
		if dbGetUserRespec(v.User) < 0 {
			isALoser(guildID, v.User)
		} else {
			isNotALoser(guildID, v.User)
		}
	}
}

func initTopUsers(guildID string) {
	guild, err := DiscordSession.Guild(guildID)
	if err != nil {
		panic(err)
	}

	supremeID := getRoleID(guildID, "Supreme Ruler")
	if supremeID != "" {
		rulerRoleID[guildID] = supremeID

		userID := dbGetTopUser()

		for _, v := range guild.Members {
			if isBot(v.User) {
				continue
			}
			if v.User.ID == userID {
				continue
			}
			DiscordSession.GuildMemberRoleRemove(guildID, v.User.ID, supremeID)
		}

		if userID != "" {
			err = DiscordSession.GuildMemberRoleAdd(guildID, userID, supremeID)
			if err == nil {
				supremeRuler = userID
			}
		}
	}

	initRulingClass(guildID)
}

func initRulingClass(guildID string) {
	rulingID := getRoleID(guildID, "Ruling Class")

	if rulingID != "" {
		rulingClassRoleID[guildID] = rulingID

		checkRulingClass(guildID)
	}
}

func checkTopUser(guildID string, user *discordgo.User) {
	roleID, ok := rulerRoleID[guildID]
	if !ok {
		return
	}
	ok = (supremeRuler != "")
	if dbUserIsTop(user) && !ok {
		DiscordSession.GuildMemberRoleAdd(guildID, user.ID, roleID)
		supremeRuler = user.ID
	} else if dbUserIsTop(user) && ok && supremeRuler != user.ID {
		DiscordSession.GuildMemberRoleRemove(guildID, supremeRuler, roleID)
		DiscordSession.GuildMemberRoleAdd(guildID, user.ID, roleID)
		supremeRuler = user.ID
	} else if !dbUserIsTop(user) && ok && supremeRuler == user.ID {
		DiscordSession.GuildMemberRoleRemove(guildID, user.ID, roleID)
		newRuler := dbGetTopUser()
		err := DiscordSession.GuildMemberRoleAdd(guildID, newRuler, roleID)
		if err == nil {
			supremeRuler = newRuler
		}
	}
}

func checkRulingClass(guildID string) {
	guild, err := DiscordSession.Guild(guildID)
	roleID, ok := rulerRoleID[guildID]
	if err != nil || !ok {
		return
	}

	newRulingClass := make(map[string]bool)
	dbGetRulingClass(&newRulingClass)

	if reflect.DeepEqual(newRulingClass, rulingClass) {
		return
	}

	for k, v := range newRulingClass {
		rulingClass[k] = v
	}

	for _, v := range guild.Members {
		if isBot(v.User) {
			continue
		}
		if rulingClass[v.User.ID] {
			DiscordSession.GuildMemberRoleAdd(guildID, v.User.ID, roleID)
		} else {
			DiscordSession.GuildMemberRoleRemove(guildID, v.User.ID, roleID)
		}
	}
}

func isALoser(guildID string, user *discordgo.User) {
	if role, ok := loserRoleID[guildID]; ok {
		DiscordSession.GuildMemberRoleAdd(guildID, user.ID, role)
	}
}

func isNotALoser(guildID string, user *discordgo.User) {
	if role, ok := loserRoleID[guildID]; ok {
		DiscordSession.GuildMemberRoleRemove(guildID, user.ID, role)
	}
}

func getRoleID(guildID, roleName string) (roleID string) {
	roles, _ := DiscordSession.GuildRoles(guildID)
	var role *discordgo.Role
	for _, v := range roles {
		if v.Name == roleName {
			role = v
			break
		}
	}
	if role == nil {
		return ""
	}
	return role.ID
}

func addRespec(guildID string, user *discordgo.User, rating int) {
	change := addRespecHelp(user, rating)

	if change == badChange {
		isALoser(guildID, user)
	} else if change == goodChange {
		isNotALoser(guildID, user)
	}

	checkTopUser(guildID, user)
	checkRulingClass(guildID)
}

func addRespecHelp(user *discordgo.User, rating int) int {
	// abs(userRating) / abs(totalRespec)
	userRespec := dbGetUserRespec(user)
	newRespec := rating

	if totalRespec == 0 {
		totalRespec = 1
	}
	if userRespec == 0 {
		userRespec = 1
	}

	temp := math.Abs(float64(userRespec)) * math.Log(1+math.Abs(float64(userRespec))) / math.Abs(float64(totalRespec)) * 0.65

	if math.Abs(float64(userRespec)) > chatLimiter {
		if userRespec > 0 && newRespec < 0 {
			temp = 0.01
		} else if userRespec < 0 && newRespec > 0 {
			temp = 0.01
		}
	} else if temp > 0.15 {
		temp = 0.15
	} else if temp < 0.01 {
		temp = 0.01
	}
	if rand.Float64() < temp {
		newRespec = -newRespec
	}

	totalRespec += newRespec
	Log(fmt.Sprintf("%v %+d respec", user, newRespec))

	dbGainRespec(user, newRespec)

	if userRespec >= 0 && userRespec+newRespec < 0 {
		return badChange
	} else if userRespec < 0 && userRespec+newRespec >= 0 {
		return goodChange
	}

	return noChange
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

	Log(fmt.Sprintf("%v: %v", author, message.ContentWithMentionsReplaced()))

	numRespec += respecMentions(guild.ID, author, message)

	addRespec(guild.ID, author, numRespec)

	dbNewMessage(author, message, numRespec, timeStamp)
}

func messageExistsInDB(messageID string) bool {
	return dbMessageExists(messageID)
}

// if someone talkin to you you aight
func respecMentions(guildID string, author *discordgo.User, message *discordgo.Message) (respec int) {
	usersList := message.Mentions
	timeStamp, _ := message.Timestamp.Parse()

	roles := message.MentionRoles
	guild, err := DiscordSession.Guild(guildID)

	if err != nil {
		panic(err)
	}

	for _, v := range roles {
		usersList = append(usersList, mentionRoleHelper(guild, v)...)
	}

	users := make(map[string]*discordgo.User)

	for _, v := range usersList {
		users[v.ID] = v
	}

	for _, v := range users {
		if isBot(v) {
			continue
		}
		if v.ID == author.ID {
			Log(fmt.Sprintf("%v mentioned self", author))
			dbMention(author, v, message, -mentionValue, timeStamp)
			respec -= mentionValue
		} else if !canMention(v, timeStamp) {
			Log(fmt.Sprintf("%v mentioned by %v too soon since last mention", v, author))
			dbMention(author, v, message, 0, timeStamp)
		} else {
			Log(fmt.Sprintf("%v mentioned by %v", v, author))
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
		}
		return true
	}
	return true
}

func mentionRoleHelper(guild *discordgo.Guild, roleID string) (users []*discordgo.User) {
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

	Log(fmt.Sprintf("%v got a reaction from %v", author, user))

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

	Log(fmt.Sprintf("%v lost a reaction", author))

	Log(fmt.Sprintf("%v removed a reaction", user))
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
