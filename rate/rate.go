package rate

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/Jaggernaut555/respecbot/db"
	"github.com/Jaggernaut555/respecbot/logging"
	"github.com/Jaggernaut555/respecbot/state"
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
	chatLimiter       = 166
)

const (
	badChange  = iota
	noChange   = iota
	goodChange = iota
)

const (
	rulingClassRoleName = "Ruling Class"
	topUserRoleName     = "Supreme Ruler"
	losersRoleNAme      = "Losers"
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

	db.LoadRespec(&userRatings)

	logging.Log(fmt.Sprintf("loaded %v ratings", len(userRatings)))

	totalRespec = db.GetTotalRespec()
}

func InitChannel(channelID string) (err error) {
	channel, err := state.Session.Channel(channelID)
	if err != nil {
		return err
	}

	db.AddChannel(channel, true)
	state.Channels[channel.ID] = true
	state.Servers[channel.GuildID] = true
	if err = initLosers(channel.GuildID); err != nil {
		return err
	}
	err = initTopUsers(channel.GuildID)
	return err
}

func initLosers(guildID string) (err error) {
	guild, err := state.Session.Guild(guildID)
	if err != nil {
		return err
	}

	loserRoleID[guildID] = getRoleID(guildID, losersRoleNAme)

	for _, v := range guild.Members {
		if v.User.Bot {
			continue
		}
		if db.GetUserRespec(v.User) < 0 {
			isALoser(guildID, v.User)
		} else {
			isNotALoser(guildID, v.User)
		}
	}
	return nil
}

func initTopUsers(guildID string) (err error) {
	guild, err := state.Session.Guild(guildID)
	if err != nil {
		return err
	}

	supremeID := getRoleID(guildID, topUserRoleName)
	if supremeID != "" {
		rulerRoleID[guildID] = supremeID

		userID := db.GetTopUser()

		for _, v := range guild.Members {
			if v.User.Bot {
				continue
			}
			if v.User.ID == userID {
				continue
			}
			state.Session.GuildMemberRoleRemove(guildID, v.User.ID, supremeID)
		}

		if userID != "" {
			err = state.Session.GuildMemberRoleAdd(guildID, userID, supremeID)
			if err == nil {
				supremeRuler = userID
			}
		}
	}

	return initRulingClass(guildID)
}

func initRulingClass(guildID string) (err error) {
	rulingID := getRoleID(guildID, rulingClassRoleName)

	if rulingID != "" {
		rulingClassRoleID[guildID] = rulingID

		return checkRulingClass(guildID)
	}
	return nil
}

func checkTopUser(guildID string, user *discordgo.User) {
	roleID, ok := rulerRoleID[guildID]
	if !ok && roleID == "" {
		roleID = getRoleID(guildID, topUserRoleName)
		if roleID == "" {
			return
		}
		rulerRoleID[guildID] = roleID
	}

	if db.UserIsTop(user) && !ok {
		state.Session.GuildMemberRoleAdd(guildID, user.ID, roleID)
		supremeRuler = user.ID
	} else if db.UserIsTop(user) && ok && supremeRuler != user.ID {
		state.Session.GuildMemberRoleRemove(guildID, supremeRuler, roleID)
		state.Session.GuildMemberRoleAdd(guildID, user.ID, roleID)
		supremeRuler = user.ID
	} else if !db.UserIsTop(user) && ok && supremeRuler == user.ID {
		state.Session.GuildMemberRoleRemove(guildID, user.ID, roleID)
		newRuler := db.GetTopUser()
		err := state.Session.GuildMemberRoleAdd(guildID, newRuler, roleID)
		if err == nil {
			supremeRuler = newRuler
		}
	}
}

func checkRulingClass(guildID string) (err error) {
	guild, err := state.Session.Guild(guildID)
	roleID, ok := rulingClassRoleID[guildID]
	if err != nil {
		return err
	} else if !ok && roleID == "" {
		roleID = getRoleID(guildID, rulingClassRoleName)
		if roleID == "" {
			return nil
		}
		rulingClassRoleID[guildID] = roleID
	}

	newRulingClass := make(map[string]bool)
	db.GetRulingClass(&newRulingClass)

	if reflect.DeepEqual(newRulingClass, rulingClass) {
		return
	}

	for k, v := range newRulingClass {
		rulingClass[k] = v
	}

	for _, v := range guild.Members {
		if v.User.Bot {
			continue
		}
		if rulingClass[v.User.ID] {
			state.Session.GuildMemberRoleAdd(guildID, v.User.ID, roleID)
		} else {
			state.Session.GuildMemberRoleRemove(guildID, v.User.ID, roleID)
		}
	}
	return nil
}

func isALoser(guildID string, user *discordgo.User) {
	if roleID, ok := loserRoleID[guildID]; ok {
		state.Session.GuildMemberRoleAdd(guildID, user.ID, roleID)
	} else {
		roleID = getRoleID(guildID, losersRoleNAme)
		if roleID == "" {
			return
		}
		loserRoleID[guildID] = roleID
		state.Session.GuildMemberRoleAdd(guildID, user.ID, roleID)
	}
}

func isNotALoser(guildID string, user *discordgo.User) {
	if roleID, ok := loserRoleID[guildID]; ok {
		state.Session.GuildMemberRoleRemove(guildID, user.ID, roleID)
	} else {
		roleID = getRoleID(guildID, losersRoleNAme)
		if roleID == "" {
			return
		}
		loserRoleID[guildID] = roleID
		state.Session.GuildMemberRoleRemove(guildID, user.ID, roleID)
	}
}

func getRoleID(guildID, roleName string) (roleID string) {
	roles, _ := state.Session.GuildRoles(guildID)
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

func AddRespec(guildID string, user *discordgo.User, rating int) {
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
	userRespec := db.GetUserRespec(user)
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
	logging.Log(fmt.Sprintf("%v %+d respec", user, newRespec))

	db.GainRespec(user, newRespec)

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

	channel, err := state.Session.Channel(message.ChannelID)
	if err != nil {
		return
	}
	guild, err := state.Session.Guild(channel.GuildID)
	if err != nil {
		return
	}

	logging.Log(fmt.Sprintf("%v: %v", author, message.ContentWithMentionsReplaced()))

	numRespec += respecMentions(guild.ID, author, message)

	AddRespec(guild.ID, author, numRespec)

	db.NewMessage(author, message, numRespec, timeStamp)
}

func messageExistsInDB(messageID string) bool {
	return db.MessageExists(messageID)
}

// if someone talkin to you you aight
func respecMentions(guildID string, author *discordgo.User, message *discordgo.Message) (respec int) {
	usersList := message.Mentions
	timeStamp, _ := message.Timestamp.Parse()

	roles := message.MentionRoles
	guild, err := state.Session.Guild(guildID)

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
		if v.Bot {
			continue
		}
		if v.ID == author.ID {
			logging.Log(fmt.Sprintf("%v mentioned self", author))
			db.AddMention(author, v, message, -mentionValue, timeStamp)
			respec -= mentionValue
		} else if !canMention(v, timeStamp) {
			logging.Log(fmt.Sprintf("%v mentioned by %v too soon since last mention", v, author))
			db.AddMention(author, v, message, 0, timeStamp)
		} else {
			logging.Log(fmt.Sprintf("%v mentioned by %v", v, author))
			AddRespec(guildID, v, mentionValue)
			db.AddMention(author, v, message, mentionValue, timeStamp)
		}
	}

	return
}

func canMention(user *discordgo.User, timeGiven time.Time) bool {
	if oldTime, ok := db.GetUserLastMentionedTime(user.String()); ok {
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
	user, _ := state.Session.User(reaction.UserID)
	message, _ := state.Session.ChannelMessage(reaction.ChannelID, reaction.MessageID)
	author := message.Author
	timeStamp := time.Now()

	channel, _ := state.Session.Channel(message.ChannelID)
	guild, _ := state.Session.Guild(channel.GuildID)

	if user.ID == author.ID {
		AddRespec(guild.ID, author, -reactionValue)
	} else if validReactionAdd(user.String(), author.String(), timeStamp) {
		AddRespec(guild.ID, author, reactionValue)
	}

	logging.Log(fmt.Sprintf("%v got a reaction from %v", author, user))

	db.ReactionAdd(user, reaction, timeStamp)
}

func validReactionAdd(GiverID, ReceiverID string, timeGiven time.Time) bool {
	if oldTime, ok := db.GetUserLastReactionAddTime(GiverID, ReceiverID); ok {
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
	user, _ := state.Session.User(reaction.UserID)
	message, _ := state.Session.ChannelMessage(reaction.ChannelID, reaction.MessageID)
	author := message.Author
	timeStamp := time.Now()

	channel, _ := state.Session.Channel(message.ChannelID)
	guild, _ := state.Session.Guild(channel.GuildID)

	if author.ID == user.ID {
		AddRespec(guild.ID, author, -reactionValue)
	} else if validReactionRemove(user.String(), author.String(), timeStamp) {
		AddRespec(guild.ID, author, -reactionValue)
	}

	logging.Log(fmt.Sprintf("%v lost a reaction", author))

	logging.Log(fmt.Sprintf("%v removed a reaction", user))
	AddRespec(guild.ID, user, -reactionValue)
	db.ReactionRemove(user, reaction, timeStamp)
}

func validReactionRemove(GiverID, ReceiverID string, timeGiven time.Time) bool {
	if oldTime, ok := db.GetUserLastReactionRemoveTime(GiverID, ReceiverID); ok {
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
	db.LoadRespec(&temp)

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
