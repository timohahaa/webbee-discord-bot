package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	jira "github.com/andygrunwald/go-jira"
	"github.com/bwmarrin/discordgo"
	confluence "github.com/essentialkaos/go-confluence/v5"
	bitbucket "github.com/ktrysmt/go-bitbucket"
	//bbb "github.com/go-bitbucket/bitbucket"
)

var (
	Conf       *Config = &Config{}
	EmbedColor int     = 0x0c84df
)

// checks for issue priroity and marks it with a discord emoji
func checkPriority(priority string) string {
	switch priority {
	case "Highest":
		return fmt.Sprintf(":bangbang: **%v** :bangbang:", priority)
	case "High":
		return fmt.Sprintf(":red_square: **%v**", priority)
	case "Medium":
		return fmt.Sprintf(":orange_square: **%v**", priority)
	case "Low":
		return fmt.Sprintf(":green_square:  **%v**", priority)
	case "Lowest":
		return fmt.Sprintf(":white_large_square: **%v**", priority)
	default:
		return priority
	}
}

// gets all jira issue info and formats it to a discord message
func getJiraResponce(issueID string) (string, error) {
	// create a jira client
	auth := &jira.BasicAuthTransport{
		Username: Conf.Atlassian.Username,
		Password: Conf.Atlassian.Password,
	}
	jiraClient, err := jira.NewClient(auth.Client(), Conf.Atlassian.JiraUrl)
	if err != nil {
		return "", fmt.Errorf("error while connecting to Jira: %w", err)
	}
	// retrieve the issue object
	issue, _, err := jiraClient.Issue.Get(issueID, &jira.GetQueryOptions{})
	if err != nil {
		return "", fmt.Errorf("error while getting jira issue: %w", err)
	}

	// it can be that some issue fields are nil
	// so we need to check for nillness
	// it actually happens, that there could be no Assignee or Description
	assignee := "Нет"
	if issue.Fields.Assignee != nil {
		assignee = issue.Fields.Assignee.DisplayName
	}
	// no need to check for Description, because of a string "nil" value
	// вот тут немного некрасиво, raw string litteral, но поверьте, так сильно легче настраивать сообщение!
	return fmt.Sprintf(
		`:arrow_forward: **[%v](%v)** :arrow_backward:

**Приоритет:** %v
**Assignee:** %v
**Описание:**

%v`,
		issue.Fields.Summary,
		Conf.Atlassian.JiraUrl+"/browse/"+issueID,
		checkPriority(issue.Fields.Priority.Name),
		assignee,
		issue.Fields.Description), nil
}

// gets confluence page description and formats it to a discrod message
func getConfluenceResponce(contentID string) (string, error) {
	// create confluence api
	cfApi, err := confluence.NewAPI(Conf.Atlassian.ConfluenceUrl, Conf.Atlassian.Username, Conf.Atlassian.Password)
	if err != nil {
		return "", fmt.Errorf("error while connecting to confluence: %w", err)
	}
	// get the content by id
	content, err := cfApi.GetContentByID(contentID, confluence.ContentIDParameters{
		Expand: []string{"space"},
	})
	if err != nil {
		return "", fmt.Errorf("error while getting confluence content: %w", err)
	}
	// вот тут немного некрасиво, raw string litteral, но поверьте, так сильно легче настраивать сообщение!
	return fmt.Sprintf(
		`:large_blue_diamond: **[%v](%v)**
**Space:** %v`,
		content.Title, Conf.Atlassian.ConfluenceUrl+"/pages/viewpage.action?pageId="+contentID, content.Space.Name), nil
}

func getBitbucketResponse() (string, error) {
	bb := bitbucket.NewBasicAuth(Conf.Atlassian.Username, Conf.Atlassian.Password)
	url, _ := url.Parse("https://bitbucket.moskit.pro/rest/api/1.0")
	bb.SetApiBaseURL(*url)
	repo, err := bb.Repositories.Repository.Get(&bitbucket.RepositoryOptions{RepoSlug: "leaders2023"})
	fmt.Println(bb.GetApiHostnameURL())
	fmt.Printf("Repo is: %+v\nError is:%v\n", repo, err)

	return "", nil
}

func parseURL(URL string) (string, error) {
	// here I used regexp to find confluence "contentID", to easily dscard non-page links
	var pageID = regexp.MustCompile("pageId=[0-9]*")
	switch {

	// jira
	case strings.HasPrefix(URL, "https://jira.moskit.pro/browse/"):
		// dont check the "found" parameter, because switch-case has already matched it
		issueID, _ := strings.CutPrefix(URL, "https://jira.moskit.pro/browse/")
		return getJiraResponce(issueID)
	case strings.HasPrefix(URL, "https://jira.web-bee.ru/browse/"):
		issueID, _ := strings.CutPrefix(URL, "https://jira.web-bee.ru/browse/")
		return getJiraResponce(issueID)

	//confluence
	case strings.HasPrefix(URL, "https://confluence.moskit.pro"):
		contentID := pageID.FindString(URL)
		// get rid off of "pageId=" part
		if len(contentID) != 7 { // len("pageId=") == 7
			contentID = contentID[7:]
		}
		return getConfluenceResponce(contentID)
	case strings.HasPrefix(URL, "https://confluence.web-bee.ru"):
		contentID := pageID.FindString(URL)
		// get rid off of "pageId=" part
		if len(contentID) != 7 {
			contentID = contentID[7:]
		}
		return getConfluenceResponce(contentID)

	// no match
	default:
		return "", errors.New("no URL found")
	}
}

func getErrorMessage(err error) string {
	// this is what I call "ЧИСТАЯ АРХИТЕКТУРА"
	// can use switch-case too :/
	if strings.HasPrefix(err.Error(), "error while connecting to jira") {
		return "Не могу подключиться к Jira :("
	} else if strings.HasPrefix(err.Error(), "error while getting jira issue") {
		return "Не могу найти такую задачу в Jira :("
	} else if strings.HasPrefix(err.Error(), "error while connecting to confluence") {
		return "Не могу подключиться с Confluence :("
	} else if strings.HasPrefix(err.Error(), "error while getting confluence content") {
		return "Не могу найти страницу с таким айди :("
	} else if strings.HasPrefix(err.Error(), "no URL found") {
		return "Не вижу валидной ссылки на Jira/Confluence в сообщении :("
	} else {
		return "Неизвестная ошибка :("
	}
}

// message handler will be called every time a new message is send to a chat
func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// if a message was sent by a bot itself, ignore it
	if m.Author.ID == s.State.User.ID {
		return
	}
	// check for the "!desc" command, prefix can be changed in the config
	if strings.HasPrefix(m.Content, Conf.Prefix+"desc") {
		// if a message is just "!desc", then it does not contain any links to jira/confluence, so ignore it
		if len(m.Content) == 4+len(Conf.Prefix) {
			return
		}
		// get the url
		URL := m.Content[4+len(Conf.Prefix):]
		// get the description
		description, err := parseURL(URL)
		// сейчас вы увидите то, что я называю "ЧИСТАЯ АРХИТЕКТУРА"
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, getErrorMessage(err))
		}
		s.ChannelMessageSend(m.ChannelID, description)
	}
}

// message parser will be called every time a new message is send to a chat
func messageParser(s *discordgo.Session, m *discordgo.MessageCreate) {
	// if a message was sent by a bot itself, ignore it
	if m.Author.ID == s.State.User.ID {
		return
	}
	// all possible link variants
	// var jiraMoskit = regexp.MustCompile(`https:\/\/jira.moskit.pro\/browse\/[a-zA-Z]*\-[0-9]*`)
	// var jiraWebbee = regexp.MustCompile(`https:\/\/jira\.web\-bee\.ru\/browse\/[a-zA-Z]*\-[0-9]*`)
	var confMoskit = regexp.MustCompile(`https:\/\/confluence\.moskit\.pro\S*`)
	var confWebbee = regexp.MustCompile(`https:\/\/confluence\.web\-bee\.ru\S*`)
	// this scans for jira issueID by itself
	var jiraIssue = regexp.MustCompile(`[a-zA-Z]*\-[0-9]+`)

	// start with jira first
	matches := jiraIssue.FindAllString(m.Content, -1)
	// send an issue description for every match
	fmt.Println(matches)
	for _, match := range matches {
		description, err := getJiraResponce(match)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, match+" - ошибка - "+getErrorMessage(err))
		} else {
			//s.ChannelMessageSend(m.ChannelID, description)
			s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Description: description,
				Color:       EmbedColor, // yellow, because bees are yellow!!!
			})
		}
	}
	// then confluence
	pages := confMoskit.FindAllString(m.Content, -1)
	pages = append(pages, confWebbee.FindAllString(m.Content, -1)...)
	if len(pages) == 0 {
		return
	}
	fmt.Println(pages)
	// send a page description for every match
	for _, match := range pages {
		// get a description
		description, err := parseURL(match)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, match+" - ошибка - "+getErrorMessage(err))
		} else {
			//s.ChannelMessageSend(m.ChannelID, description)
			s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Description: description,
				Color:       EmbedColor, // yellow, because bees are yellow!!!
			})
		}
	}
}

func main() {
	// read config first
	var err error
	Conf, err = NewConfig("config.yaml")
	if err != nil {
		fmt.Println("Could not read config!")
		log.Fatal(err)
	}

	getBitbucketResponse()

	bot, err := discordgo.New("Bot " + Conf.Token)
	if err != nil {
		log.Fatal(err)
	}
	// allow a bot to read and  send messages
	bot.Identify.Intents |= discordgo.IntentsGuildMessages

	// use messageHandler and bot will respond in "command mode"
	// bot.AddHandler(messageHandler)
	// use message parser and bot will silently parse each message looking for links
	bot.AddHandler(messageParser)
	// open websocket connection to discord
	err = bot.Open()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Bot is running fine!")

	// graceful shutdown
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// close the websocket connection
	bot.Close()
	fmt.Println("Bot is done running!")
}
