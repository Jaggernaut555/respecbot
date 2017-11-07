package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"math/big"
	"math/rand"
	"os"
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

const (
	ratingPath     = "ratings"
	userRatingPath = "ratings/users"

	numRespecsToSave = 5
)

var (
	userRatings     map[string]int
	userLastMessage map[string]time.Time
	userLastRespec  map[string]time.Time
	lastUserPost    map[string]string

	respecsToSave int
	totalRespec   int

	letters map[rune]string
)

func InitRatings() {
	userRatings = make(map[string]int)
	userLastMessage = make(map[string]time.Time)
	userLastRespec = make(map[string]time.Time)
	lastUserPost = make(map[string]string)

	rand.Seed(time.Now().Unix())

	// If rating path does not exist create it
	if _, err := os.Stat(ratingPath); os.IsNotExist(err) {
		err = os.Mkdir(ratingPath, 0755)
		if err != nil {
			log.Printf("Error creating directory: %s\n", err)
		}
		fmt.Printf("Creating ratings directory %s\n", ratingPath)
	}

	loadRatings(&userRatings, userRatingPath)

	totalRespec = getTotalRespec()

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

// Load the given rating map from the path
func loadRatings(list *map[string]int, path string) {
	ratingsFile, err := os.Open(path)
	defer ratingsFile.Close()
	if os.IsNotExist(err) {
		fmt.Printf("No file %s\n", path)
	} else {
		decoder := gob.NewDecoder(ratingsFile)
		err = decoder.Decode(list)
		if err != nil {
			log.Printf("Error loading file %s: %s\n", path, err)
		}
		fmt.Printf("loaded %d ratings from %s\n", len(*list), path)
	}
}

// save all ratings maps
func SaveRatings() {
	saveMap(userRatings, userRatingPath)
}

func getTotalRespec() (total int) {
	var users = getRatingsLists()
	for _, v := range users {
		total += v.Value
	}
	return
}

func respec(user *discordgo.User, rating int) {
	userRatings[user.String()] += rating
	// abs(userRating) / abs(totalRespec)
	if totalRespec != 0 && userRatings[user.String()] != 0 {
		var temp = math.Abs(float64(userRatings[user.String()])) / math.Abs(float64(totalRespec))
		if temp > 0.1 {
			temp = 0.1
		} else if temp < 0.01 {
			temp = 0.01
		}
		if rand.Float64() < temp {
			rating = -rating * 2
			userRatings[user.String()] += rating
		}
	}

	totalRespec += rating
	fmt.Printf("%v %+d respec\n", user, rating)

	if respecsToSave >= numRespecsToSave {
		//SaveRatings()
		respecsToSave = 0
	}
	respecsToSave++
}

// give respec by reacting
func RespecReactionAdd(session *discordgo.Session, reaction *discordgo.MessageReactionAdd) {
	user, _ := session.User(reaction.UserID)
	message, _ := session.ChannelMessage(reaction.ChannelID, reaction.MessageID)
	author := message.Author
	fmt.Printf("%v got a reaction from %v\n", author, user)
	if user == author {
		respec(author, -5)
	} else {
		respec(author, 2)
	}
}

// no fuckin gaming the system
func RespecReactionRemove(session *discordgo.Session, reaction *discordgo.MessageReactionRemove) {
	user, _ := session.User(reaction.UserID)
	message, _ := session.ChannelMessage(reaction.ChannelID, reaction.MessageID)
	author := message.Author
	fmt.Printf("%v lost a reaction\n", author)
	respec(author, -2)
	fmt.Printf("%v removed a reaction\n", user)
	respec(user, -1)
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
func GiveRespec(incomingMessage *discordgo.MessageCreate) {
	mentions := incomingMessage.Message.Mentions
	author := incomingMessage.Message.Author
	timeStamp, _ := incomingMessage.Timestamp.Parse()

	// lose respec if you use it wrong
	if len(mentions) < 1 || checkLastRespecGiven(author, timeStamp) || respecingSelf(author, mentions) {
		fmt.Println(author, "Used respec wrong")
		respec(author, -5)
		respecGiven(author, timeStamp)
		return
	}

	for _, v := range mentions {
		respec(v, 4)
		fmt.Println("Repec given to ", v)
	}

	respecGiven(author, timeStamp)
}

//fucc someones respec
func NoRespec(incomingMessage *discordgo.MessageCreate) {
	mentions := incomingMessage.Message.Mentions
	author := incomingMessage.Message.Author
	timeStamp, _ := incomingMessage.Timestamp.Parse()

	// lose respec if you use it wrong
	if len(mentions) < 1 || checkLastRespecGiven(author, timeStamp) || respecingSelf(author, mentions) {
		fmt.Println(author, "Used respec wrong")
		respec(author, -10)
		respecGiven(author, timeStamp)
		return
	}

	for _, v := range mentions {
		respec(v, -4)
		fmt.Println("Repec taken from ", v.String())
	}

	respecGiven(author, timeStamp)
}

// save the ratings map to given path
func saveMap(data map[string]int, path string) {
	ratingFile, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0666)
	defer ratingFile.Close()
	if err != nil {
		log.Printf("failed to open or create file %s: %s\n", path, err)
	}

	encoder := gob.NewEncoder(ratingFile)

	if err := encoder.Encode(data); err != nil {
		log.Printf("Failed to save ratings to %s: %s", path, err)
	}
	fmt.Println("Saved to " + path)
}

// get all da users in list
func getRatingsLists() (users pairList) {
	for k, v := range userRatings {
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
