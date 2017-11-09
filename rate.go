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
	userLastRespec map[string]time.Time

	totalRespec int
)

func InitRatings() {
	userRatings := make(map[string]int)
	userLastRespec = make(map[string]time.Time)

	rand.Seed(time.Now().Unix())

	dbLoadRespec(&userRatings)
	fmt.Println("loaded", len(userRatings), "ratings")

	totalRespec = dbGetTotalRespec()
}

func addRespec(user *discordgo.User, rating int) {
	// abs(userRating) / abs(totalRespec)
	userRespec := dbGetUserRespec(user)
	newRespec := rating
	if totalRespec != 0 && userRespec != 0 {
		temp := math.Abs(float64(userRespec)) * math.Log(1+math.Abs(float64(userRespec))) / math.Abs(float64(totalRespec))
		//var temp = math.Abs(float64(userRespec)) / math.Abs(float64(totalRespec))
		if temp > 0.33 {
			temp = 0.33
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
}

// give respec by reacting
func RespecReactionAdd(session *discordgo.Session, reaction *discordgo.MessageReactionAdd) {
	user, _ := session.User(reaction.UserID)
	message, _ := session.ChannelMessage(reaction.ChannelID, reaction.MessageID)
	author := message.Author
	timeStamp, _ := message.Timestamp.Parse()
	fmt.Printf("%v got a reaction from %v\n", author, user)
	if user.ID == author.ID {
		addRespec(author, -reactionValue)
	} else {
		addRespec(author, reactionValue)
	}

	dbReactionAdd(author, reaction, timeStamp)
}

// no fuckin gaming the system
func RespecReactionRemove(session *discordgo.Session, reaction *discordgo.MessageReactionRemove) {
	user, _ := session.User(reaction.UserID)
	message, _ := session.ChannelMessage(reaction.ChannelID, reaction.MessageID)
	author := message.Author
	timeStamp, _ := message.Timestamp.Parse()
	fmt.Printf("%v lost a reaction\n", author)
	addRespec(author, -reactionValue)
	fmt.Printf("%v removed a reaction\n", user)
	addRespec(user, -reactionValue)
	dbReactionRemove(author, reaction, timeStamp)
}

// evaluate messages
func RespecMessage(incomingMessage *discordgo.MessageCreate) {

	message := incomingMessage.Message
	author := message.Author
	timeStamp, _ := message.Timestamp.Parse()
	numRespec := applyRules(author, message)

	fmt.Printf("%v: %v\n", author, message.Content)
	addRespec(author, numRespec)

	respecMentions(author, message)

	dbNewMessage(author, incomingMessage, numRespec, timeStamp)
}

// if someone talkin to you you aight
//func respecMentions(user *discordgo.User, users []*discordgo.User, message *discordgo.Message, timeStamp time.Time) {
func respecMentions(author *discordgo.User, message *discordgo.Message) (respec int) {
	users := message.Mentions
	timeStamp, _ := message.Timestamp.Parse()

	for _, v := range users {
		if v.ID == author.ID {
			fmt.Println(v, "mentioned by", author)
			addRespec(v, -mentionValue)
			dbMention(author, v, message, -mentionValue, timeStamp)
		} else {
			fmt.Println(v, "mentioned by", author)
			addRespec(v, mentionValue)
			dbMention(author, v, message, mentionValue, timeStamp)
		}
	}

	return 0
}

func checkLastRespecGiven(user *discordgo.User, timeGiven time.Time) bool {
	if oldTime, ok := userLastRespec[user.String()]; ok {
		timeDelta := timeGiven.Sub(oldTime)
		if timeDelta.Minutes() < 30 {
			return true
		}
	}
	return false
}

// if you try to respec yourself fuck you
func respecingSelf(author *discordgo.User, users []*discordgo.User) bool {
	for _, v := range users {
		if author.ID == v.ID {
			return true
		}
	}
	return false
}

// gif someone respec
func GiveRespec(incomingMessage *discordgo.MessageCreate, positive bool) {
	mentions := incomingMessage.Message.Mentions
	author := incomingMessage.Message.Author
	timeStamp, _ := incomingMessage.Timestamp.Parse()
	respec := dbGetUserRespec(author)
	numRespec := 0

	if respec <= 0 {
		numRespec = 2
	} else {
		numRespec = respec / 10
		if numRespec < 2 {
			numRespec = 2
		} else if numRespec > 25 {
			numRespec = 25
		}
	}

	if !positive {
		numRespec = -numRespec
	}

	// lose respec if you use it wrong
	if len(mentions) < 1 || checkLastRespecGiven(author, timeStamp) || respecingSelf(author, mentions) {
		fmt.Println(author, "Used respec wrong")
		addRespec(author, -numRespec)
		dbGiveRespec(author, author, -numRespec, timeStamp)
		mentions = nil
	} else {
		addRespec(author, correctUsageValue)
		dbGiveRespec(author, author, correctUsageValue, timeStamp)
	}

	for _, v := range mentions {
		fmt.Println(author, " gave respec to ", v)
		addRespec(v, numRespec)
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
		if v.Value > 0 {
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
