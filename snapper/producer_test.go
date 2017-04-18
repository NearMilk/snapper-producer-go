package snapper

import (
	"encoding/json"
	"log"
	"sync"
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
)

// Message ...
type Message struct {
	E string      `json:"e"`
	D *MsgComment `json:"d"`
}

type MsgComment struct {
	CommentsCount    int       `json:"commentsCount"`
	AttachmentsCount int       `json:"attachmentsCount"`
	LastCommentedAt  time.Time `json:"lastCommentedAt"`
}

var (
	options = &Options{
		SecretKeys: []string{"Securekey"},
		ExpiresIn:  604800,
		Address:    "127.0.0.1:7720",
		ProducerID: "teambition",
	}

	userroom  = "user" + "58b7ab90b7162c96120cba9b"
	projectid = "project" + "58c919cfaa169c056f36bc11"
	eventname = ":change:task" + "58cb3c62bb53d8c3989e3b59"
)

func TestProducer(t *testing.T) {
	producer, connecterr := New(options)
	if connecterr != nil {
		panic(connecterr)
	}
	go func() {
		event := <-producer.Event
		switch event.EventType {
		case ErrorEvent:
			log.Println("Occur error:", event.EventData)
			break
		case CloseEvent:
			log.Println("Closed")
			break
		case ReconnectEvent:
			log.Println("Try reconnect:", event.EventData)
			break
		}
	}()
	defer producer.Close()

	t.Run("Producer with SendMessage func that should be", func(t *testing.T) {

		msg := &Message{
			E: eventname,
			D: &MsgComment{CommentsCount: 1, AttachmentsCount: 1, LastCommentedAt: time.Now()},
		}
		msgstr, _ := json.Marshal(msg)
		producer.SendMessage(projectid, string(msgstr))
	})

	t.Run("Producer with SendMessage callback that should be", func(t *testing.T) {
		msg := &Message{
			E: eventname,
			D: &MsgComment{CommentsCount: 2, AttachmentsCount: 1, LastCommentedAt: time.Now()},
		}
		msgstr, _ := json.Marshal(msg)

		producer.SendMessage(projectid, string(msgstr))
		time.Sleep(time.Millisecond * 10)
	})
	t.Run("Producer with Request func  that should be", func(t *testing.T) {
		assert := assert.New(t)

		sss := "{\"e\":\":change:user/58b7ab90b7162c96120cba9b\",\"d\":{\"ated\":0,\"normal\":11,\"later\":0,\"private\":2,\"badge\":10,\"hasNormal\":true,\"hasAted\":false,\"hasLater\":false,\"hasPrivate\":true}}"
		msg := [][]string{[]string{userroom, sss}}

		result, err := producer.Request("publish", msg)
		assert.Equal(result, float64(1))
		assert.Nil(err)

		msg = [][]string{[]string{userroom, sss}, []string{userroom, sss}}
		result, err = producer.Request("publish", msg)
		assert.Equal(result, float64(2))
		assert.Nil(err)

	})
	t.Run("Producer with receive notifation that should be", func(t *testing.T) {

		sss := "{\"e\":\":change:user/58b7ab90b7162c96120cba9b\",\"d\":{\"ated\":0,\"normal\":14,\"later\":0,\"private\":1,\"badge\":10,\"hasNormal\":true,\"hasAted\":false,\"hasLater\":false,\"hasPrivate\":true}}"
		producer.SendMessage(userroom, sss)
		time.Sleep(time.Millisecond * 50)
	})
	t.Run("Producer with joinroom func that should be", func(t *testing.T) {
		assert := assert.New(t)

		result, err := producer.JoinRoom(userroom, "xxxxxxxxxxxxxxx")
		assert.Equal(result, float64(1))
		assert.Nil(err)

		result, err = producer.JoinRoom(userroom, "")
		assert.Equal(result, nil)
		assert.Contains(err.Error(), "Invalid params")

	})
	t.Run("Producer with LeaveRoom func that should be", func(t *testing.T) {
		assert := assert.New(t)

		result, err := producer.LeaveRoom(userroom, "xxxxxxxxxxxxxxx")
		assert.Equal(result, float64(1))
		assert.Nil(err)

		result, err = producer.LeaveRoom(userroom, "")

		assert.Equal(result, nil)
		assert.Contains(err.Error(), "Invalid params")
	})
	t.Run("Producer with goroutine Request that should be", func(t *testing.T) {
		assert := assert.New(t)
		var wg sync.WaitGroup
		wg.Add(50 * 100)
		for i := 0; i < 50; i++ {
			go func(num int) {
				for index := 0; index < 100; index++ {
					msg := &Message{
						E: eventname,
						D: &MsgComment{CommentsCount: index, AttachmentsCount: 1, LastCommentedAt: time.Now()},
					}
					msgstr, _ := json.Marshal(msg)
					str := string(msgstr)
					result, err := producer.Request("publish", []string{projectid, str})
					if assert.Nil(err) {
						assert.Equal(result, float64(1))
					} else {
						log.Println(err.Error())
					}
					wg.Done()
				}
			}(i)
		}
		wg.Wait()
	})
	t.Run("Producer with JoinRoom and LeaveRoom func that should be", func(t *testing.T) {
		assert := assert.New(t)

		result, err := producer.JoinRoom(userroom, "xxxxxxxxxxxxxxx")
		assert.Equal(result, float64(1))
		assert.Nil(err)

		result, err = producer.LeaveRoom(userroom, "xxxxxxxxxxxxxxx")

	})
	t.Run("Producer with goroutine SendMessage that should be", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			msg := &Message{
				E: eventname,
				D: &MsgComment{CommentsCount: i, AttachmentsCount: 1, LastCommentedAt: time.Now()},
			}
			msgstr, _ := json.Marshal(msg)
			producer.SendMessage(projectid, string(msgstr))
		}
		time.Sleep(time.Second)
	})

}
func TestProducerReconnect(t *testing.T) {
	assert := assert.New(t)
	producer, connecterr := New(options)
	if connecterr != nil {
		panic(connecterr)
	}
	go func() {
		for {
			event := <-producer.Event
			switch event.EventType {
			case ErrorEvent:
				log.Println("Occur error:", event.EventData)
			case CloseEvent:
				log.Println("Closed")
				return
			case ReconnectEvent:
				log.Println("Try reconnect:", event.EventData)
			}
		}
	}()

	producer.conn.socket.Close()
	result, err := producer.JoinRoom(userroom, "xxxxxxxxxxxxxxxc")
	if assert.Nil(err) {
		assert.Equal(result, float64(1))
	}

	result, err = producer.LeaveRoom(userroom, "xxxxxxxxxxxxxxxc")
	if assert.Nil(err) {
		assert.Equal(result, float64(1))
	}
}
func TestProducerError(t *testing.T) {
	assert := assert.New(t)
	erropts := &Options{
		SecretKeys: []string{"Securekey"},
		ExpiresIn:  604800,
		Address:    "127.0.0.1:7720",
	}
	producer, err := New(erropts)
	assert.NotNil(err)

	erropts = &Options{
		Address: "192.168.0.21x7721",
	}

	producer, err = New(erropts)
	assert.Contains(err.Error(), "missing port in address")

	erropts = &Options{}

	producer, err = New(erropts)
	assert.Contains(err.Error(), "connectex: No connection could be made because the target machine actively refused it.")

	erropts = &Options{
		SecretKeys: []string{"toeknxx"},
		Address:    options.Address,
	}
	producer, err = New(erropts)
	assert.Contains(err.Error(), "Unauthorized")
	assert.NotNil(producer)
}
