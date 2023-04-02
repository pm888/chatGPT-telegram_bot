package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	gogpt "github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
)

type Config struct {
	TelegramToken string `mapstructure:"tgToken"`
	GptToken      string `mapstructure:"gptToken"`
}

type GptBotStruct struct {
	MaxTokensGpt int
	NameBot      string
}

func LoadConfig(path string) (c Config, err error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	viper.AutomaticEnv()

	err = viper.ReadInConfig()

	if err != nil {
		return
	}

	err = viper.Unmarshal(&c)
	return
}

func sendChatGPT(c *gogpt.Client, sendText string, gptM GptBotStruct) string {
	ctx := context.Background()

	req := gogpt.CompletionRequest{
		Model:            gptM.NameBot,
		MaxTokens:        gptM.MaxTokensGpt,
		Prompt:           sendText,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
	}

	resp, err := c.CreateCompletion(ctx, req)
	if err != nil {
		return "ChatGPT API error"
	} else {
		return resp.Choices[0].Text
	}
}

func main() {
	// Reading config.yaml
	config, err := LoadConfig(".")

	if err != nil {
		panic(fmt.Errorf("fatal error with config.yaml: %w", err))
	}

	// Chat GPT initialization
	chatGPT := gogpt.NewClient(config.GptToken)

	// Telegram initialization
	bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true // set to false for suppress logs in stdout
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Start Telegram long polling update
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 5
	updates, _ := bot.GetUpdatesChan(u)

	GptMode := GptBotStruct{
		MaxTokensGpt: 3000,
		NameBot:      "text-davinci-003",
	}

	//Check message in updates
	for update := range updates {
		if update.Message == nil {
			continue
		}

		checkOne := strings.HasPrefix(update.Message.Text, "/text")
		checkTwo := strings.HasPrefix(update.Message.Text, "/code")
		checkThree := strings.HasPrefix(update.Message.Text, "/curie")
		if checkOne || checkTwo || checkThree {
			if checkOne {
				GptMode.MaxTokensGpt = 3000
				GptMode.NameBot = "text-davinci-003"
			} else if checkTwo {
				GptMode.MaxTokensGpt = 4096
				GptMode.NameBot = "code-davinci-002"
			} else {
				GptMode.MaxTokensGpt = 2048
				GptMode.NameBot = "text-curie-001"
			}
			update.Message.Text = "Вы выбрали режим " + GptMode.NameBot
		} else {
			update.Message.Text = sendChatGPT(chatGPT, update.Message.Text, GptMode)
		}

		// Send message to Telegram
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID

		_, err = bot.Send(msg)
		if err != nil {
			log.Println("Error:", err)
		}
	}
}
