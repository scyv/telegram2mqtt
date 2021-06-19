package main

import (
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	tb "gopkg.in/tucnak/telebot.v2"
)

func main() {

	teleToken, err := ioutil.ReadFile("token.txt")

	if err != nil {
		log.Fatal("Could not read token.txt. Create a token.txt file that contains the token of your telegram bot.")
		return
	}

	b, err := tb.NewBot(tb.Settings{
		URL: "",

		Token:  string(teleToken),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle("/connect", func(m *tb.Message) {
		mqttserver := m.Payload

		b.Send(m.Sender, "Connecting to: "+mqttserver)
		b.Send(m.Sender, "Enter topic to publish commands with /topic")
		ioutil.WriteFile("connects/"+strconv.FormatInt(m.Chat.ID, 10), []byte(mqttserver), 0600)
	})
	b.Handle("/topic", func(m *tb.Message) {
		topic := m.Payload

		b.Send(m.Sender, "Sending commands to: "+topic)
		ioutil.WriteFile("connects/"+strconv.FormatInt(m.Chat.ID, 10)+"_topic", []byte(topic), 0600)
	})
	b.Handle("/sethelp", func(m *tb.Message) {
		help := m.Payload

		b.Send(m.Sender, "Storing help: "+help)
		ioutil.WriteFile("connects/"+strconv.FormatInt(m.Chat.ID, 10)+"_help", []byte(help), 0600)
	})
	b.Handle("/?", func(m *tb.Message) {
		help, err := ioutil.ReadFile("connects/" + strconv.FormatInt(m.Chat.ID, 10) + "_help")
		if err != nil {
			b.Send(m.Sender, "No help stored :-/")
			return
		}
		b.Send(m.Sender, string(help))
	})

	b.Handle(tb.OnText, func(m *tb.Message) {
		command := m.Text

		if strings.HasPrefix(command, "/") {
			return
		}

		mqttserver, err := ioutil.ReadFile("connects/" + strconv.FormatInt(m.Chat.ID, 10))
		if err != nil {
			b.Send(m.Sender, "Not connected. Use /connect <mqtt-broker-url> to connect.")
			return
		}
		topic, err := ioutil.ReadFile("connects/" + strconv.FormatInt(m.Chat.ID, 10) + "_topic")
		if err != nil {
			b.Send(m.Sender, "Don't know where to send command to. Use /topic <topicname> to specify a topic to send your commands to.")
			return
		}

		opts := mqtt.NewClientOptions()
		opts.AddBroker(string(mqttserver))
		opts.SetClientID("telemqtt")
		client := mqtt.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}

		token := client.Publish(string(topic), 0, false, command)
		token.Wait()
		client.Disconnect(250)

		b.Send(m.Sender, "Command sent: "+command)

	})

	log.Print("Starting Bot")
	b.Start()
}
