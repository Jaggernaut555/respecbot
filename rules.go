package main

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Rule func(*discordgo.User, *discordgo.Message) int

const (
	bigValue   = 5
	midValue   = 3
	smallValue = 2
	minValue   = 1
)

var (
	rules              []Rule
	letters            map[rune]string
	channelLastMessage map[string]*discordgo.Message
	userLastMessage    map[string]time.Time
)

func InitRules() {
	rules = []Rule{lastPost,
		respecLetters,
		respecLength,
		respecTime,
	}

	letters = make(map[rune]string)
	channelLastMessage = make(map[string]*discordgo.Message)
	userLastMessage = make(map[string]time.Time)

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

func applyRules(author *discordgo.User, message *discordgo.Message) (respec int) {
	for _, v := range rules {
		respec += v(author, message)
	}
	return
}

// if a user is mentioned, respec them
// if you use more than twice as many consonants as vowels, you lose respec
// if you use one word only you lose respec
// if you spam or barely talk fucc u

// fuck you double posters
func lastPost(author *discordgo.User, newMessage *discordgo.Message) (respec int) {
	if message, ok := channelLastMessage[newMessage.ChannelID]; ok {
		if message.Author.ID == author.ID {
			respec -= minValue
		} else {
			respec += smallValue
		}

		if message.Content == newMessage.Content {
			respec -= bigValue
		}
	} else {
		respec += minValue
	}

	channelLastMessage[newMessage.ChannelID] = newMessage

	fmt.Println("lastPost", respec)
	return
}

// fuck arbitrary amounts of letters
func respecLetters(author *discordgo.User, message *discordgo.Message) (respec int) {
	content := message.Content
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

	totalLetters := big.NewInt(consonantCount + vowelCount)

	if totalLetters.ProbablyPrime(2) && totalLetters.Int64() > 10 {
		respec += bigValue
	}
	if totalLetters.Int64() == capsCount {
		respec -= bigValue
	}
	if vowelCount > consonantCount {
		respec += minValue
	} else if float64(vowelCount) < float64(consonantCount)*0.65 {
		respec -= smallValue
	}
	if otherCount > totalLetters.Int64() {
		respec -= bigValue
	}
	if capsCount < 1 && (vowelCount > 0 || consonantCount > 0) {
		respec -= smallValue
	} else {
		respec += minValue
	}

	fmt.Println("letters", respec)
	return
}

// fuck spammers and afk's
func respecTime(author *discordgo.User, message *discordgo.Message) (respec int) {
	//func respecTime(user *discordgo.User, newTime time.Time) (respec int) {

	timeStamp, _ := message.Timestamp.Parse()
	if oldTime, ok := userLastMessage[author.String()]; ok {
		timeDelta := timeStamp.Sub(oldTime)
		if timeDelta.Seconds() < 2 {
			respec -= smallValue
		} else if timeDelta.Hours() > 6 {
			respec -= int(timeDelta.Hours()) * minValue
		} else {
			respec += minValue
		}
	} else {
		respec += minValue
	}

	userLastMessage[author.String()] = timeStamp

	fmt.Println("time", respec)
	return
}

// fucc 1 word replies or walls of text
//func respecLength(content string) (respec int) {
func respecLength(author *discordgo.User, message *discordgo.Message) (respec int) {
	content := message.Content

	words := strings.Split(content, " ")
	length := len(words)

	if length < 2 {
		respec -= smallValue
	} else if length > 25 {
		respec -= bigValue
	} else {
		respec += minValue
	}

	fmt.Println("length", respec)
	return
}
