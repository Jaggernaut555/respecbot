package main

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bwmarrin/discordgo"
)

type pair struct {
	Key   string
	Value int
}

type pairList []pair

const ()

var (
	userLastMessage map[string]time.Time
	userLastRespec  map[string]time.Time
	lastUserPost    map[string]string

	totalRespec int

	letters map[rune]string
)

func InitRatings() {
	userRatings := make(map[string]int)
	userLastMessage = make(map[string]time.Time)
	userLastRespec = make(map[string]time.Time)
	lastUserPost = make(map[string]string)

	rand.Seed(time.Now().Unix())

	dbLoadRespec(&userRatings)
	fmt.Println("loaded", len(userRatings), "ratings")

	totalRespec = dbGetTotalRespec()

	letters = make(map[rune]string)

	var vowels = []rune{'a', 'e', 'i', 'o', 'u'}
	var capVowels = []rune{'A', 'E', 'I', 'O', 'U'}
	var consonants = []rune{'b', 'c', 'd', 'f', 'g', 'h', 'j', 'k', 'l', 'm', 'n', 'p', 'q', 'r', 's', 't', 'v', 'w', 'x', 'y', 'z'}
	var capConsonants = []rune{'B', 'C', 'D', 'F', 'G', 'H', 'J', 'K', 'L', 'M', 'N', 'P', 'Q', 'R', 'S', 'T', 'V', 'W', 'X', 'Y', 'Z'}

	for _, v := range vowels {
		letters[v] = "vowel"
	}

	for _, v := range consonants {
		letters[v] = "consonant"
	}

	for _, v := range capVowels {
		letters[v] = "capVowel"
	}

	for _, v := range capConsonants {
		letters[v] = "capConsonant"
	}
}

func respec(user *discordgo.User, rating int) {
	// abs(userRating) / abs(totalRespec)
	userRespec := dbGetUserRespec(user)
	newRespec := rating
	if totalRespec != 0 && userRespec != 0 {
		var temp = math.Abs(float64(userRespec)) / math.Abs(float64(totalRespec))
		if temp > 0.1 {
			temp = 0.1
		} else if temp < 0.01 {
			temp = 0.01
		}
		if rand.Float64() < temp {
			newRespec = -rating
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
	if user == author {
		respec(author, -5)
	} else {
		respec(author, 2)
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
	respec(author, -2)
	fmt.Printf("%v removed a reaction\n", user)
	respec(user, -1)
	dbReactionRemove(author, reaction, timeStamp)
}

// evaluate messages
func RespecMessage(incomingMessage *discordgo.MessageCreate) {
	// if a user is mentioned, respec them
	// if you use more than twice as many consonants as vowels, you lose respec
	// if you use one word only you lose respec
	// if you spam or barely
	message := incomingMessage.Message
	author := message.Author
	totalRespec := 0

	totalRespec += respecLetters(message.Content)

	totalRespec += respecLength(message.Content)

	var mentions = message.Mentions
	respecMentions(mentions)

	newTime, _ := message.Timestamp.Parse()
	respecTime(author, newTime)
	userLastMessage[author.String()] = newTime

	totalRespec += lastPost(author, message.ChannelID)

	fmt.Printf("%v: %v\n", author, message.Content)
	respec(author, totalRespec)

	dbNewMessage(author, incomingMessage, totalRespec, newTime)
}

// fuck you double posters
func lastPost(author *discordgo.User, channel string) (respec int) {
	if user, _ := lastUserPost[channel]; user == author.String() {
		respec -= 1
	} else {
		respec += 1
	}
	lastUserPost[channel] = author.String()
	return
}

// fuck arbitrary amounts of letters
func respecLetters(content string) (respec int) {
	var capsCount int64
	var vowelCount int64
	var consonantCount int64
	var otherCount int64

	for _, c := range content {
		switch letters[c] {
		case "capVowel":
			capsCount++
			vowelCount++
		case "vowel":
			vowelCount++
		case "capConsonant":
			capsCount++
			consonantCount++
		case "consonant":
			consonantCount++
		default:
			otherCount++
		}
	}

	var totalLetters = big.NewInt(consonantCount + vowelCount)

	if totalLetters.ProbablyPrime(2) && totalLetters.Int64() > 10 {
		respec += 5
	}
	if totalLetters.Int64() == capsCount {
		respec -= 5
	}
	if vowelCount > consonantCount {
		respec += 1
	} else if float64(vowelCount) < float64(consonantCount)/1.25 {
		respec -= 1
	}
	if otherCount > totalLetters.Int64() {
		respec -= 5
	}
	if capsCount < 1 && vowelCount > 0 && capsCount > 0 {
		respec -= 1
	}
	return
}

// fuck spammers and afk's
func respecTime(user *discordgo.User, newTime time.Time) (respec int) {
	if oldTime, ok := userLastMessage[user.String()]; ok {
		timeDelta := newTime.Sub(oldTime)
		if timeDelta.Seconds() < 2 {
			respec -= 2
		} else if timeDelta.Hours() > 6 {
			respec -= int(timeDelta.Hours())
		} else {
			respec += 1
		}
	}

	return
}

// fucc 1 word replies or walls of text
func respecLength(content string) (respec int) {
	var length int
	var words = strings.Split(content, " ")
	length = len(words)

	if length < 2 {
		respec = -1
	} else if length > 25 {
		respec -= 5
	}
	return
}

// if someone talkin to you you aight
func respecMentions(users []*discordgo.User) {
	for _, v := range users {
		respec(v, 2)
	}
}

func respecGiven(user *discordgo.User, timeGiven time.Time) {
	userLastRespec[user.String()] = timeGiven
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
		if author == v {
			respec(v, -15)
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
	numRespec := 4
	if !positive {
		numRespec = -numRespec
	}

	// lose respec if you use it wrong
	if len(mentions) < 1 || checkLastRespecGiven(author, timeStamp) || respecingSelf(author, mentions) {
		fmt.Println(author, "Used respec wrong")
		respec(author, -5)
		dbGiveRespec(author, author, -5, timeStamp)
		respecGiven(author, timeStamp)
		return
	}

	for _, v := range mentions {
		fmt.Println(author, " gave respec to ", v)
		respec(v, numRespec)
		dbGiveRespec(author, v, numRespec, timeStamp)
	}

	respecGiven(author, timeStamp)
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
func GetMostRespec() (reply string) {
	var buf bytes.Buffer
	users := getRatingsLists()

	sort.Sort(sort.Reverse(users))

	var padding = 3
	w := new(tabwriter.Writer)
	w.Init(&buf, 0, 0, padding, ' ', 0)
	for k, v := range users {
		if k > 15 {
			break
		}
		fmt.Fprintf(w, "%v\t%v\t\n", v.Key, v.Value)
	}
	w.Flush()

	reply = fmt.Sprintf("%v", buf.String())
	return
}

func (p pairList) Len() int           { return len(p) }
func (p pairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p pairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
