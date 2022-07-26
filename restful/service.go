package restful

import (
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/gorilla/mux"
	"net/http"
	"time"
	"twitter_oracle/db"
	"twitter_oracle/log"
	"twitter_oracle/stream"
)

var (
	WriteResponseErr = errors.New("write response error")
)

const (
	DefaultRespStatus     = 100
	Success               = 200
	ConversationIdInvalid = 24
)

type Service struct {
	port       string
	db         *db.DBService
	Subscriber *stream.Subscriber
}

func InitRestService(port string, db *db.DBService) *Service {
	return &Service{
		port: port,
		db:   db,
	}
}

type Resp struct {
	Status int    `json:"status"`
	Value  string `json:"value"`
}

func NewResp() Resp {
	return Resp{
		Status: DefaultRespStatus,
		Value:  "",
	}
}

func AutoResponse(writer http.ResponseWriter, resp Resp) {
	b, _ := json.Marshal(resp)
	_, err := writer.Write(b)
	if err != nil {
		log.Error(WriteResponseErr, err)
		return
	}
}

func (c *Service) Start() error {
	log.Info("start queryer rpc port:" + c.port)
	address := "0.0.0.0:" + c.port
	r := mux.NewRouter()

	r.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {

	})

	r.HandleFunc("/add_conversation/{conversation}", func(writer http.ResponseWriter, request *http.Request) {
		resp := NewResp()
		defer AutoResponse(writer, resp)
		vars := mux.Vars(request)
		conv := vars["conversation"]
		//verify
		if conv == "" {
			resp.Status = ConversationIdInvalid
			return
		}
		c.Subscriber.AddConversation(conv, stream.DefaultHandler)
		resp.Status = Success
		return
	})

	go func() {
		err := http.ListenAndServe(address, r)
		if err != nil {
			utils.Fatalf("http listen error", err)
		}
	}()
	return nil
}

func PrintErrorStr(prefix string, detail string) string {
	if detail != "" {
		return time.Now().String() + " ERROR " + prefix + ":" + detail
	} else {
		return time.Now().String() + " ERROR " + prefix
	}
}
