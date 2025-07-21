package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	"github.com/oliveagle/jsonpath"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"
	"io"
	"io/ioutil"
	"net/http"
	test "net/http/httptest"
	"net/url"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	"nkonev.name/video/db"
	"nkonev.name/video/dto"
	"nkonev.name/video/handlers"
	"nkonev.name/video/listener"
	"nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/rabbitmq"
	"nkonev.name/video/services"
	"nkonev.name/video/tasks"
	"nkonev.name/video/type_registry"
	"nkonev.name/video/utils"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	shutdown()
	os.Exit(retCode)
}

func shutdown() {

}

const aaaEmuPort = "8061"
const chatEmuPort = "8062"

var userTester = base64.StdEncoding.EncodeToString([]byte("tester"))

var theConfig *config.ExtendedConfig
var lgr *logger.Logger

func setup() {
	config.InitViper()
	lgr = logger.NewLogger()

	viper.Set("aaa.url.base", "http://localhost:"+aaaEmuPort)
	viper.Set("chat.url.base", "http://localhost:"+chatEmuPort)

	theConfig, _ = createTypedConfig()

	d, err := db.ConfigureDb(lgr, nil)
	defer d.Close()
	if err != nil {
		lgr.Panicf("Error during getting db connection for test: %v", err)
	}
	d.RecreateDb()
}

type AaaEmu struct{}

func (receiver AaaEmu) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(200)

	ava := "http://image.jpg"
	u1 := &dto.User{
		Id:     1,
		Login:  "testor_protobuf",
		Avatar: &ava,
	}
	u2 := &dto.User{
		Id:    2,
		Login: "testor_protobuf2",
	}
	var users = []*dto.User{u1, u2}
	out, err := json.Marshal(users)
	if err != nil {
		lgr.Errorln("Failed to encode get users request:", err)
		return
	}

	resp.Write(out)
}

func waitForAaaEmu() {
	restClient := client.NewRestClient(theConfig, lgr)
	i := 0
	const maxAttempts = 60
	success := false
	for ; i <= maxAttempts; i++ {
		_, err := restClient.GetUsers(context.Background(), []int64{0})
		if err != nil {
			lgr.Infof("Awaiting while emulator have been started")
			time.Sleep(time.Second * 1)
			continue
		} else {
			success = true
			break
		}
	}
	if !success {
		lgr.Panicf("Cannot await for aaa emu will be started")
	}
	lgr.Infof("Aaa emu have started")
	restClient.CloseIdleConnections()
}

// it's requires to call this method every time when we create real app with startAppFull()
func waitForVideoServer() {
	restClient := client.NewRestClient(theConfig, lgr)
	i := 0
	const maxAttempts = 60
	success := false
	for ; i <= maxAttempts; i++ {
		expirationTime := utils.SecondsToStringMilliseconds(time.Now().Add(4 * time.Hour).Unix())
		contentType := "application/json;charset=UTF-8"
		requestHeaders1 := map[string][]string{
			"Accept":           {contentType},
			"Content-Type":     {contentType},
			"X-Auth-Expiresin": {expirationTime},
			"X-Auth-Username":  {userTester},
			"X-Auth-Userid":    {"1"},
		}
		getChatRequest := &http.Request{
			Method: "GET",
			Header: requestHeaders1,
			URL:    stringToUrl("http://localhost" + viper.GetString("server.apiAddress") + "/api/video/config"),
		}
		getChatResponse, err := restClient.GetClient().Do(getChatRequest)
		if err != nil {
			lgr.Infof("Awaiting while chat have been started - transport error")
			time.Sleep(time.Second * 1)
			continue
		} else if !(getChatResponse.StatusCode >= 200 && getChatResponse.StatusCode < 300) {
			lgr.Infof("Awaiting while chat have been started - non-2xx code")
			time.Sleep(time.Second * 1)
			continue
		} else {
			success = true
			break
		}
	}
	if !success {
		lgr.Panicf("Cannot await for chat will be started")
	}
	lgr.Infof("chat have started")
	restClient.CloseIdleConnections()
}

func startAaaEmu() *http.Server {
	s := &http.Server{
		Addr:    ":" + aaaEmuPort,
		Handler: AaaEmu{},
	}

	go func() {
		lgr.Info(s.ListenAndServe())
	}()

	waitForAaaEmu()

	return s
}

type ChatEmu struct{}

func (receiver ChatEmu) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(200)

	bci := dto.BasicChatDto{
		TetATet:        false,
		ParticipantIds: []int64{1, 2, 3},
	}

	out, err := json.Marshal(bci)
	if err != nil {
		lgr.Errorln("Failed to encode get users request:", err)
		return
	}

	resp.Write(out)
}

func waitForChatEmu() {
	restClient := client.NewRestClient(theConfig, lgr)
	i := 0
	const maxAttempts = 60
	success := false
	for ; i <= maxAttempts; i++ {
		_, err := restClient.GetBasicChatInfo(context.Background(), -1, -1)
		if err != nil {
			lgr.Infof("Awaiting while emulator have been started")
			time.Sleep(time.Second * 1)
			continue
		} else {
			success = true
			break
		}
	}
	if !success {
		lgr.Panicf("Cannot await for aaa emu will be started")
	}
	lgr.Infof("Aaa emu have started")
	restClient.CloseIdleConnections()
}

func startChatEmu() *http.Server {
	s := &http.Server{
		Addr:    ":" + chatEmuPort,
		Handler: ChatEmu{},
	}

	go func() {
		lgr.Info(s.ListenAndServe())
	}()

	waitForChatEmu()

	return s
}

func runTest(t *testing.T, testFunc interface{}) *fxtest.App {
	var s fx.Shutdowner
	app := fxtest.New(
		t,
		fx.Supply(lgr),
		fx.WithLogger(func(log *logger.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log.ZapLogger}
		}),
		fx.Populate(&s),
		fx.Provide(
			createTypedConfig,
			configureTracer,
			configureApiEcho,
			client.NewRestClient,
			newMockLivekitRoomClient(t),
			client.NewEgressClient,
			handlers.NewUserHandler,
			handlers.NewConfigHandler,
			handlers.ConfigureApiStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			handlers.NewLivekitWebhookHandler,
			handlers.NewInviteHandler,
			handlers.NewRecordHandler,
			rabbitmq.CreateRabbitMqConnection,
			producer.NewRabbitUserCountPublisher,
			producer.NewRabbitInvitePublisher,
			producer.NewRabbitUserIdsPublisher,
			producer.NewRabbitDialStatusPublisher,
			producer.NewRabbitRecordingPublisher,
			producer.NewRabbitNotificationsPublisher,
			producer.NewRabbitScreenSharePublisher,
			services.NewNotificationService,
			services.NewUserService,
			services.NewStateChangedEventService,
			services.NewEgressService,
			tasks.RedisV9,
			tasks.RedisLocker,
			tasks.Scheduler,
			tasks.NewSynchronizeWithLivekitService,
			listener.CreateAaaUserSessionsKilledListener,
			type_registry.NewTypeRegistryInstance,
			configureMigrations,
			db.ConfigureDb,
		),
		fx.Invoke(
			runMigrations,
			//runEcho,
			testFunc,
		),
	)
	defer app.RequireStart().RequireStop()
	assert.NoError(t, s.Shutdown(), "error in app shutdown")
	return app
}

func newMockLivekitRoomClient(t *testing.T) func() client.LivekitRoomClient {
	return func() client.LivekitRoomClient {
		return client.NewMockLivekitRoomClient(t)
	}
}

func getJsonPathResult(t *testing.T, body string, jsonpath0 string) interface{} {
	res := getJsonPathRaw(t, body, jsonpath0)
	assert.NotEmpty(t, res)
	return res
}

func getJsonPathRaw(t *testing.T, body string, jsonpath0 string) interface{} {
	var jsonData interface{}
	assert.Nil(t, json.Unmarshal([]byte(body), &jsonData))
	res, err := jsonpath.JsonPathLookup(jsonData, jsonpath0)
	assert.Nil(t, err)
	return res
}

func stringToUrl(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

func stringToReadCloser(s string) io.ReadCloser {
	r := strings.NewReader(s)
	rc := ioutil.NopCloser(r)
	return rc
}

func requestWithHeader(method, path string, h http.Header, body io.Reader, e *ApiEcho) (int, string, http.Header) {
	req := test.NewRequest(method, path, body)
	req.Header = h
	rec := test.NewRecorder()
	e.ServeHTTP(rec, req) // most wanted
	return rec.Code, rec.Body.String(), rec.Result().Header
}

func request(method, path string, userId int64, body io.Reader, e *ApiEcho) (int, string, http.Header) {
	Header := map[string][]string{
		echo.HeaderContentType: {"application/json"},
		"X-Auth-Expiresin":     {"1590022342295000"},
		"X-Auth-Username":      {userTester},
		"X-Auth-Userid":        {utils.Int64ToString(userId)},
	}
	return requestWithHeader(method, path, Header, body, e)
}

func TestLivekitSynchronizeTaskIsGoingToCreateTheMissedEntries(t *testing.T) {
	chatEmu := startChatEmu()
	defer chatEmu.Close()

	runTest(t, func(
		database *db.DB,
		task *tasks.SynchronizeWithLivekitService,
		livekitRoomClient client.LivekitRoomClient,
	) {
		mockLivekitRoomClient := livekitRoomClient.(*client.MockLivekitRoomClient)
		var chatId int64 = 1
		roomName := "chat" + utils.Int64ToString(chatId)
		mockLivekitRoomClient.On("ListRooms", mock.Anything, mock.Anything).Return(&livekit.ListRoomsResponse{
			Rooms: []*livekit.Room{
				{
					Name: roomName,
				},
			},
		}, nil)
		tokenId := "93e9d926-3291-43bb-bfb0-81be053705d9"
		var userId int64 = 3
		md := &dto.MetadataDto{
			UserId:  userId,
			Login:   "userlogin" + utils.Int64ToString(userId),
			Avatar:  "",
			TokenId: uuid.MustParse(tokenId),
		}
		mdb, err := json.Marshal(md)
		assert.NoError(t, err)
		mdbs := string(mdb)
		mockLivekitRoomClient.On("ListParticipants",
			mock.Anything,
			&livekit.ListParticipantsRequest{Room: roomName}).
			Return(&livekit.ListParticipantsResponse{Participants: []*livekit.ParticipantInfo{
				{
					Identity: utils.Int64ToString(userId) + "_02e9d926-3291-43bb-bfb0-81be053705d9",
					Metadata: mdbs,
				},
			}}, nil)

		var numOfEntriesBefore int
		rowBefore := database.QueryRow("select count (*) from user_call_state")
		assert.NoError(t, rowBefore.Scan(&numOfEntriesBefore))
		assert.Equal(t, 0, numOfEntriesBefore)

		// run the periodic task
		task.DoJob(context.Background())

		var numOfEntriesAfter int
		rowAfter := database.QueryRow("select count (*) from user_call_state")
		assert.NoError(t, rowAfter.Scan(&numOfEntriesAfter))
		assert.Equal(t, 1, numOfEntriesAfter)

		userState, err := db.TransactWithResult(context.Background(), database, func(tx *db.Tx) (*dto.UserCallState, error) {
			return tx.Get(context.Background(), dto.UserCallStateId{
				TokenId: uuid.MustParse(tokenId),
				UserId:  userId,
			})
		})
		assert.NoError(t, err)

		assert.Equal(t, uuid.MustParse(tokenId), userState.TokenId)
		assert.Equal(t, (*uuid.UUID)(nil), userState.OwnerTokenId)
		assert.Equal(t, chatId, userState.ChatId)
		assert.Equal(t, db.CallStatusInCall, userState.Status)
		assert.Equal(t, (*int64)(nil), userState.OwnerUserId)
	})
}

func TestItsImpossibleToMakeACallToUserWhoAlreadyInCall(t *testing.T) {
	aaaEmu := startAaaEmu()
	defer aaaEmu.Close()

	chatEmu := startChatEmu()
	defer chatEmu.Close()

	runTest(t, func(
		e *ApiEcho,
		database *db.DB,
	) {
		var calleeUserId int64 = 42
		var chatId int64 = 1

		tokenId := "9449d926-3291-43bb-bfb0-81be053705d9"

		// create an entry that represents calleeUserId is already in call
		assert.NoError(t, db.Transact(context.Background(), database, func(tx *db.Tx) error {
			return tx.Set(context.Background(), dto.UserCallState{
				TokenId:    uuid.MustParse(tokenId),
				UserId:     calleeUserId,
				ChatId:     chatId,
				TokenTaken: true,
				Status:     db.CallStatusInCall,
			})
		}))

		var userId int64 = 4

		// try to invite him
		c, _, _ := request("PUT", "/api/video/"+utils.Int64ToString(chatId)+"/dial/invite?userId="+utils.Int64ToString(calleeUserId)+"&call=true", userId, nil, e)

		assert.Equal(t, http.StatusConflict, c)
	})
}
