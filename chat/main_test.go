package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/labstack/echo/v4"
	"github.com/oliveagle/jsonpath"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"
	"io"
	"io/ioutil"
	"net/http"
	test "net/http/httptest"
	"net/url"
	"nkonev.name/chat/client"
	"nkonev.name/chat/config"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/handlers"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/producer"
	myRabbitmq "nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/services"
	"nkonev.name/chat/utils"
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

var userTester = base64.StdEncoding.EncodeToString([]byte("tester"))
var userTester2 = base64.StdEncoding.EncodeToString([]byte("tester2"))
var lgr *logger.Logger

func setup() {
	config.InitViper()
	lgr = logger.NewLogger()

	viper.Set("aaa.url.base", "http://localhost:"+aaaEmuPort)

	d, err := db.ConfigureDb(lgr, nil)
	defer d.Close()
	if err != nil {
		lgr.Panicf("Error during getting db connection for test: %v", err)
	}
	d.RecreateDb()
}

func TestExtractAuth(t *testing.T) {
	req := test.NewRequest("GET", "/api/should-be-secured", nil)
	headers := map[string][]string{
		"X-Auth-Expiresin": {"1590022342295000"},
		"X-Auth-Username":  {userTester},
		"X-Auth-Userid":    {"1"},
	}
	req.Header = headers

	auth, err := handlers.ExtractAuth(req, lgr)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), auth.UserId)
	assert.Equal(t, "tester", auth.UserLogin)
	assert.Equal(t, int64(1590022342), auth.ExpiresAt)
}

func requestWithHeader(method, path string, h http.Header, body io.Reader, e *echo.Echo) (int, string, http.Header) {
	req := test.NewRequest(method, path, body)
	req.Header = h
	rec := test.NewRecorder()
	e.ServeHTTP(rec, req) // most wanted
	return rec.Code, rec.Body.String(), rec.Result().Header
}

type AaaEmu struct{}

func (receiver AaaEmu) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(200)

	a := "http://image.jpg"
	u1 := &dto.User{
		Id:     1,
		Login:  "testor_protobuf",
		Avatar: &a,
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
	restClient := client.NewRestClient(lgr)
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
func waitForChatServer() {
	restClient := client.NewRestClient(lgr)
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
			Method: "POST",
			Header: requestHeaders1,
			URL:    stringToUrl("http://localhost" + viper.GetString("server.address") + "/api/chat/search"),
		}
		getChatResponse, err := restClient.Do(getChatRequest)
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

func request(method, path string, body io.Reader, e *echo.Echo) (int, string, http.Header) {
	Header := map[string][]string{
		echo.HeaderContentType: {"application/json"},
		"X-Auth-Expiresin":     {"1590022342295000"},
		"X-Auth-Username":      {userTester},
		"X-Auth-Userid":        {"1"},
	}
	return requestWithHeader(method, path, Header, body, e)
}

func configureTestMigrations() *db.MigrationsConfig {
	return &db.MigrationsConfig{AppendTestData: true}
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
			configureTracer,
			client.NewRestClient,
			services.CreateSanitizer,
			services.CreateStripTags,
			services.StripStripSourcePolicy,
			handlers.NewChatHandler,
			handlers.NewMessageHandler,
			handlers.NewBlogHandler,
			configureEcho,
			handlers.ConfigureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			configureTestMigrations,
			db.ConfigureDb,
			services.NewEvents,
			producer.NewRabbitEventsPublisher,
			producer.NewRabbitNotificationsPublisher,
			myRabbitmq.CreateRabbitMqConnection,
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

func startAppFull(t *testing.T) (*fxtest.App, fx.Shutdowner) {
	var s fx.Shutdowner
	app := fxtest.New(
		t,
		fx.Supply(lgr),
		fx.WithLogger(func(log *logger.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log.ZapLogger}
		}),
		fx.Populate(&s),
		fx.Provide(
			configureTracer,
			client.NewRestClient,
			services.CreateSanitizer,
			services.CreateStripTags,
			services.StripStripSourcePolicy,
			handlers.NewChatHandler,
			handlers.NewMessageHandler,
			handlers.NewBlogHandler,
			configureEcho,
			handlers.ConfigureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			configureTestMigrations,
			db.ConfigureDb,
			services.NewEvents,
			producer.NewRabbitEventsPublisher,
			producer.NewRabbitNotificationsPublisher,
			myRabbitmq.CreateRabbitMqConnection,
		),
		fx.Invoke(
			createFanoutNotificationsChannel,
			runMigrations,
			runEcho,
		),
	)
	waitForChatServer()
	return app, s
}

func createFanoutNotificationsChannel(connection *rabbitmq.Connection, lc fx.Lifecycle) error {
	consumeCh, err := connection.Channel(nil)
	if err != nil {
		return err
	}

	err = consumeCh.ExchangeDeclare(producer.EventsFanoutExchange, "fanout", true, false, false, false, nil)
	if err != nil {
		return err
	}
	return nil
}

func TestGetChats(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		c, b, _ := request("POST", "/api/chat/search", nil, e)
		assert.Equal(t, http.StatusOK, c)
		assert.NotEmpty(t, b)
	})
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

func TestGetChatsPaginated(t *testing.T) {
	emu := startAaaEmu()
	defer emu.Close()
	runTest(t, func(e *echo.Echo) {
		// get initial page
		firstReqBytes, _ := json.Marshal(handlers.GetChatsRequestDto{
			Size: 40,
		})
		httpFirstPage, bodyFirstPage, _ := request("POST", "/api/chat/search", bytes.NewReader(firstReqBytes), e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)
		typedResFirst := handlers.GetChatsResponseDto{}
		err := json.Unmarshal([]byte(bodyFirstPage), &typedResFirst)
		assert.NoError(t, err)

		assert.Equal(t, 40, len(typedResFirst.Items))

		assert.Equal(t, "generated_chat1000", typedResFirst.Items[0].Name)
		assert.Equal(t, "generated_chat999", typedResFirst.Items[1].Name)
		assert.Equal(t, "generated_chat998", typedResFirst.Items[2].Name)
		assert.Equal(t, "generated_chat961", typedResFirst.Items[39].Name)

		// also check get additional info from aaa emu
		assert.Equal(t, "testor_protobuf", typedResFirst.Items[0].Participants[0].Login)
		im := "http://image.jpg"
		assert.Equal(t, &im, typedResFirst.Items[0].Participants[0].Avatar)

		lastPinned := typedResFirst.Items[len(typedResFirst.Items)-1].Pinned
		lastId := typedResFirst.Items[len(typedResFirst.Items)-1].Id
		lastLastUpdateDateTime := typedResFirst.Items[len(typedResFirst.Items)-1].LastUpdateDateTime

		secondReqBytes, _ := json.Marshal(handlers.GetChatsRequestDto{
			StartingFromItemId: &dto.ChatId{
				Pinned:             lastPinned,
				LastUpdateDateTime: lastLastUpdateDateTime,
				Id:                 lastId,
			},
			Size: 40,
		})
		// get second page
		httpSecondPage, bodySecondPage, _ := request("POST", "/api/chat/search", bytes.NewReader(secondReqBytes), e)
		assert.Equal(t, http.StatusOK, httpSecondPage)
		assert.NotEmpty(t, bodySecondPage)
		typedResSecond := handlers.GetChatsResponseDto{}
		err = json.Unmarshal([]byte(bodySecondPage), &typedResSecond)
		assert.NoError(t, err)

		assert.Equal(t, 40, len(typedResSecond.Items))

		assert.Equal(t, "generated_chat960", typedResSecond.Items[0].Name)
		assert.Equal(t, "generated_chat959", typedResSecond.Items[1].Name)
		assert.Equal(t, "generated_chat958", typedResSecond.Items[2].Name)
		assert.Equal(t, "generated_chat921", typedResSecond.Items[39].Name)
	})
}

func TestGetChatsPaginatedSearch(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		// get initial page
		firstReqBytes, _ := json.Marshal(handlers.GetChatsRequestDto{
			Size:         40,
			SearchString: "gen",
		})
		httpFirstPage, bodyFirstPage, _ := request("POST", "/api/chat/search", bytes.NewReader(firstReqBytes), e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)
		typedResFirst := handlers.GetChatsResponseDto{}
		err := json.Unmarshal([]byte(bodyFirstPage), &typedResFirst)
		assert.NoError(t, err)

		assert.Equal(t, 40, len(typedResFirst.Items))

		assert.Equal(t, "generated_chat1000", typedResFirst.Items[0].Name)
		assert.Equal(t, "generated_chat999", typedResFirst.Items[1].Name)
		assert.Equal(t, "generated_chat998", typedResFirst.Items[2].Name)
		assert.Equal(t, "generated_chat961", typedResFirst.Items[39].Name)

		lastPinned := typedResFirst.Items[len(typedResFirst.Items)-1].Pinned
		lastId := typedResFirst.Items[len(typedResFirst.Items)-1].Id
		lastLastUpdateDateTime := typedResFirst.Items[len(typedResFirst.Items)-1].LastUpdateDateTime

		secondReqBytes, _ := json.Marshal(handlers.GetChatsRequestDto{
			StartingFromItemId: &dto.ChatId{
				Pinned:             lastPinned,
				LastUpdateDateTime: lastLastUpdateDateTime,
				Id:                 lastId,
			},
			Size:         40,
			SearchString: "gen",
		})

		// get second page
		httpSecondPage, bodySecondPage, _ := request("POST", "/api/chat/search", bytes.NewReader(secondReqBytes), e)
		assert.Equal(t, http.StatusOK, httpSecondPage)
		assert.NotEmpty(t, bodySecondPage)

		typedResSecond := handlers.GetChatsResponseDto{}
		err = json.Unmarshal([]byte(bodySecondPage), &typedResSecond)
		assert.NoError(t, err)

		assert.Equal(t, 40, len(typedResSecond.Items))

		assert.Equal(t, "generated_chat960", typedResSecond.Items[0].Name)
		assert.Equal(t, "generated_chat959", typedResSecond.Items[1].Name)
		assert.Equal(t, "generated_chat958", typedResSecond.Items[2].Name)
		assert.Equal(t, "generated_chat921", typedResSecond.Items[39].Name)
	})
}

func TestChatValidation(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		c, b, _ := request("POST", "/api/chat", strings.NewReader(`{"name": ""}`), e)
		assert.Equal(t, http.StatusBadRequest, c)
		textString := utils.InterfaceToString(getJsonPathResult(t, b, "$.name").(interface{}))
		assert.Equal(t, "cannot be blank", textString)

		c2, b2, _ := request("POST", "/api/chat", strings.NewReader(``), e)
		assert.Equal(t, http.StatusBadRequest, c2)
		textString2 := utils.InterfaceToString(getJsonPathResult(t, b2, "$.name").(interface{}))
		assert.Equal(t, "cannot be blank", textString2)

		c3, b3, _ := request("PUT", "/api/chat", strings.NewReader(``), e)
		assert.Equal(t, http.StatusBadRequest, c3)
		textString30 := utils.InterfaceToString(getJsonPathResult(t, b3, "$.name").(interface{}))
		assert.Equal(t, "cannot be blank", textString30)
		textString31 := utils.InterfaceToString(getJsonPathResult(t, b3, "$.id").(interface{}))
		assert.Equal(t, "cannot be blank", textString31)
	})
}

func TestChatCrud(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		// test not found
		c30, _, _ := request("GET", "/api/chat/50666", nil, e)
		assert.Equal(t, http.StatusNoContent, c30)

		chatsBefore, _ := db.CountChats(context.Background())
		c, b, _ := request("POST", "/api/chat", strings.NewReader(`{"name": "Ultra new chat"}`), e)
		assert.Equal(t, http.StatusCreated, c)

		chatsAfterCreate, _ := db.CountChats(context.Background())
		assert.Equal(t, chatsBefore+1, chatsAfterCreate)

		idInterface := getJsonPathResult(t, b, "$.id").(interface{})
		idString := utils.InterfaceToString(idInterface)
		id, _ := utils.ParseInt64(idString)
		assert.True(t, id > 0)

		c3, b3, _ := request("GET", "/api/chat/"+idString, nil, e)
		assert.Equal(t, http.StatusOK, c3)
		nameString := utils.InterfaceToString(getJsonPathResult(t, b3, "$.name").(interface{}))
		assert.Equal(t, "Ultra new chat", nameString)

		c2, _, _ := request("PUT", "/api/chat", strings.NewReader(`{ "id": `+idString+`, "name": "Mega ultra new chat"}`), e)
		assert.Equal(t, http.StatusAccepted, c2)
		row := db.QueryRow("SELECT title FROM chat WHERE id = $1", id)
		var newTitle string
		assert.Nil(t, row.Scan(&newTitle))
		assert.Equal(t, "Mega ultra new chat", newTitle)

		c1, _, _ := request("DELETE", "/api/chat/"+idString, nil, e)
		assert.Equal(t, http.StatusAccepted, c1)
		chatsAfterDelete, _ := db.CountChats(context.Background())
		assert.Equal(t, chatsBefore, chatsAfterDelete)
	})
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

func TestCreateNewMessageMakesNotificationToOtherParticipant(t *testing.T) {
	app, s := startAppFull(t)
	defer app.RequireStart().RequireStop()

	emu := startAaaEmu()
	defer emu.Close()

	expirationTime := utils.SecondsToStringMilliseconds(time.Now().Add(2 * time.Hour).Unix())
	contentType := "application/json;charset=UTF-8"
	requestHeaders1 := map[string][]string{
		"Accept":           {contentType},
		"Content-Type":     {contentType},
		"X-Auth-Expiresin": {expirationTime},
		"X-Auth-Username":  {userTester},
		"X-Auth-Userid":    {"1"},
	}

	createChatRequest := &http.Request{
		Method: "POST",
		Header: requestHeaders1,
		Body:   stringToReadCloser(`{"name": "Chat for test the Centrifuge notifications about unread messages", "participantIds": [1, 2]}`),
		URL:    stringToUrl("http://localhost" + viper.GetString("server.address") + "/api/chat"),
	}

	cl := client.NewRestClient(lgr)
	createChatResponse, err := cl.Do(createChatRequest)
	assert.Nil(t, err)
	assert.Equal(t, 201, createChatResponse.StatusCode)

	var responseDto *dto.ChatDto = new(dto.ChatDto)
	body, err := ioutil.ReadAll(createChatResponse.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(body, responseDto)
	assert.Nil(t, err)

	var chatIdString = fmt.Sprintf("%v", responseDto.Id)

	// send message to chat
	messageRequest := &http.Request{
		Method: "POST",
		Header: requestHeaders1,
		Body:   stringToReadCloser(`{"text": "Hello dude"}`),
		URL:    stringToUrl("http://localhost" + viper.GetString("server.address") + "/api/chat/" + chatIdString + "/message"),
	}
	messageResponse, err := cl.Do(messageRequest)
	assert.Nil(t, err)
	assert.Equal(t, 201, messageResponse.StatusCode)

	// send message to chat again
	messageRequest2 := &http.Request{
		Method: "POST",
		Header: requestHeaders1,
		Body:   stringToReadCloser(`{"text": "Hello dude"}`),
		URL:    stringToUrl("http://localhost" + viper.GetString("server.address") + "/api/chat/" + chatIdString + "/message"),
	}
	messageResponse2, err := cl.Do(messageRequest2)
	assert.Nil(t, err)
	assert.Equal(t, 201, messageResponse2.StatusCode)

	assert.NoError(t, s.Shutdown(), "error in app shutdown")
}

func TestBadRequestShouldReturn400(t *testing.T) {
	app, s := startAppFull(t)
	defer app.RequireStart().RequireStop()

	emu := startAaaEmu()
	defer emu.Close()

	expirationTime := utils.SecondsToStringMilliseconds(time.Now().Add(2 * time.Hour).Unix())
	contentType := "application/json;charset=UTF-8"
	requestHeaders1 := map[string][]string{
		"Accept":           {contentType},
		"Content-Type":     {contentType},
		"X-Auth-Expiresin": {expirationTime},
		"X-Auth-Username":  {userTester},
		"X-Auth-Userid":    {"1"},
	}

	createChatRequest := &http.Request{
		Method: "POST",
		Header: requestHeaders1,
		Body:   stringToReadCloser(`{"name": "Chat for test the Centrifuge notifications about unread messages", "participantIds": [1, 2]`),
		URL:    stringToUrl("http://localhost" + viper.GetString("server.address") + "/api/chat"),
	}

	cl := client.NewRestClient(lgr)
	createChatResponse, err := cl.Do(createChatRequest)
	assert.Nil(t, err)
	assert.Equal(t, 400, createChatResponse.StatusCode)

	assert.NoError(t, s.Shutdown(), "error in app shutdown")
}

func TestGetMessagesPaginated(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		// get first page
		httpFirstPage, bodyFirstPage, _ := request("GET", "/api/chat/1/message/search?startingFromItemId=6&size=3", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)

		firstPageResultWrapper := new(handlers.MessagesResponseDto)
		err := json.Unmarshal([]byte(bodyFirstPage), firstPageResultWrapper)
		assert.NoError(t, err)
		firstPageResult := firstPageResultWrapper.Items

		assert.Equal(t, 3, len(firstPageResult))
		assert.True(t, strings.HasPrefix(firstPageResult[0].Text, "generated_message5"))
		assert.True(t, strings.HasPrefix(firstPageResult[1].Text, "generated_message6"))
		assert.True(t, strings.HasPrefix(firstPageResult[2].Text, "generated_message7"))
		assert.Equal(t, int64(7), firstPageResult[0].Id)
		assert.Equal(t, int64(8), firstPageResult[1].Id)
		assert.Equal(t, int64(9), firstPageResult[2].Id)

		// get second page
		httpSecondPage, bodySecondPage, _ := request("GET", "/api/chat/1/message/search?startingFromItemId=9&size=3", nil, e)
		assert.Equal(t, http.StatusOK, httpSecondPage)
		assert.NotEmpty(t, bodySecondPage)

		secondPageResultWrapper := new(handlers.MessagesResponseDto)
		err = json.Unmarshal([]byte(bodySecondPage), secondPageResultWrapper)
		assert.NoError(t, err)
		secondPageResult := secondPageResultWrapper.Items

		assert.Equal(t, 3, len(secondPageResult))
		assert.True(t, strings.HasPrefix(secondPageResult[0].Text, "generated_message8"))
		assert.True(t, strings.HasPrefix(secondPageResult[1].Text, "generated_message9"))
		assert.True(t, strings.HasPrefix(secondPageResult[2].Text, "generated_message10"))
		assert.Equal(t, int64(10), secondPageResult[0].Id)
		assert.Equal(t, int64(11), secondPageResult[1].Id)
		assert.Equal(t, int64(12), secondPageResult[2].Id)
	})
}

func TestGetMessagesPaginatedSearch(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		// get first page
		httpFirstPage, bodyFirstPage, _ := request("GET", "/api/chat/1/message/search?startingFromItemId=6&size=3&searchString=gen", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)

		firstPageResultWrapper := new(handlers.MessagesResponseDto)
		err := json.Unmarshal([]byte(bodyFirstPage), firstPageResultWrapper)
		assert.NoError(t, err)
		firstPageResult := firstPageResultWrapper.Items

		assert.Equal(t, 3, len(firstPageResult))
		assert.True(t, strings.HasPrefix(firstPageResult[0].Text, "generated_message5"))
		assert.True(t, strings.HasPrefix(firstPageResult[1].Text, "generated_message6"))
		assert.True(t, strings.HasPrefix(firstPageResult[2].Text, "generated_message7"))
		assert.Equal(t, int64(7), firstPageResult[0].Id)
		assert.Equal(t, int64(8), firstPageResult[1].Id)
		assert.Equal(t, int64(9), firstPageResult[2].Id)

		// get second page
		httpSecondPage, bodySecondPage, _ := request("GET", "/api/chat/1/message/search?startingFromItemId=9&size=3", nil, e)
		assert.Equal(t, http.StatusOK, httpSecondPage)
		assert.NotEmpty(t, bodySecondPage)

		secondPageResultWrapper := new(handlers.MessagesResponseDto)
		err = json.Unmarshal([]byte(bodySecondPage), secondPageResultWrapper)
		assert.NoError(t, err)
		secondPageResult := secondPageResultWrapper.Items

		assert.Equal(t, 3, len(secondPageResult))
		assert.True(t, strings.HasPrefix(secondPageResult[0].Text, "generated_message8"))
		assert.True(t, strings.HasPrefix(secondPageResult[1].Text, "generated_message9"))
		assert.True(t, strings.HasPrefix(secondPageResult[2].Text, "generated_message10"))
		assert.Equal(t, int64(10), secondPageResult[0].Id)
		assert.Equal(t, int64(11), secondPageResult[1].Id)
		assert.Equal(t, int64(12), secondPageResult[2].Id)
	})
}

func TestMessageValidation(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		c, b, _ := request("POST", "/api/chat/1/message", strings.NewReader(`{"text": ""}`), e)
		assert.Equal(t, http.StatusBadRequest, c)
		textString := utils.InterfaceToString(getJsonPathResult(t, b, "$.text").(interface{}))
		assert.Equal(t, "cannot be blank", textString)

		c2, b2, _ := request("POST", "/api/chat/1/message", strings.NewReader(``), e)
		assert.Equal(t, http.StatusBadRequest, c2)
		textString2 := utils.InterfaceToString(getJsonPathResult(t, b2, "$.text").(interface{}))
		assert.Equal(t, "cannot be blank", textString2)
	})
}

func TestMessageCrud(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		messagesBefore, _ := db.CountMessages(context.Background(), 1)
		c, b, _ := request("POST", "/api/chat/1/message", strings.NewReader(`{"text": "Ultra new message"}`), e)
		assert.Equal(t, http.StatusCreated, c)

		messagesAfterCreate, _ := db.CountMessages(context.Background(), 1)
		assert.Equal(t, messagesBefore+1, messagesAfterCreate)

		idInterface := getJsonPathResult(t, b, "$.id").(interface{})
		idString := utils.InterfaceToString(idInterface)
		id, _ := utils.ParseInt64(idString)
		assert.True(t, id > 0)

		c3, b3, _ := request("GET", "/api/chat/1/message/"+idString, nil, e)
		assert.Equal(t, http.StatusOK, c3)
		textString := utils.InterfaceToString(getJsonPathResult(t, b3, "$.text").(interface{}))
		assert.Equal(t, "Ultra new message", textString)

		c4, _, _ := request("PUT", "/api/chat/1/message", strings.NewReader(`{"text": "Edited ultra new message", "id": `+idString+`}`), e)
		assert.Equal(t, http.StatusCreated, c4)

		c5, b5, _ := request("GET", "/api/chat/1/message/"+idString, nil, e)
		assert.Equal(t, http.StatusOK, c5)
		textString5 := utils.InterfaceToString(getJsonPathResult(t, b5, "$.text").(interface{}))
		assert.Equal(t, "Edited ultra new message", textString5)

		dateTimeInterface5 := utils.InterfaceToString(getJsonPathResult(t, b5, "$.editDateTime").(interface{}))
		assert.NotEmpty(t, dateTimeInterface5)

		c1, _, _ := request("DELETE", "/api/chat/1/message/"+idString, nil, e)
		assert.Equal(t, http.StatusAccepted, c1)
		messagesAfterDelete, _ := db.CountMessages(context.Background(), 1)
		assert.Equal(t, messagesBefore, messagesAfterDelete)
	})
}

func TestMessageIsSanitized(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		c, b, _ := request("POST", "/api/chat/1/message", strings.NewReader(`{"text": "<a onblur=\"alert(secret)\" href=\"http://www.google.com\">Google</a>"}`), e)
		assert.Equal(t, http.StatusCreated, c)

		idInterface := getJsonPathResult(t, b, "$.id").(interface{})
		idString := utils.InterfaceToString(idInterface)

		c3, b3, _ := request("GET", "/api/chat/1/message/"+idString, nil, e)
		assert.Equal(t, http.StatusOK, c3)
		textInterface := getJsonPathResult(t, b3, "$.text").(interface{})
		textString := utils.InterfaceToString(textInterface)
		assert.Equal(t, `<a href="http://www.google.com" rel="nofollow">Google</a>`, textString)
	})
}

func TestNotPossibleToWriteAMessageWithNotAllowedMediaUrl(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		c, b, _ := request("POST", "/api/chat/1/message", strings.NewReader(`{"text": "<img src=\"http://malicious.example.com/virus.jpg\"> Lorem ipsum"}`), e)
		assert.Equal(t, http.StatusBadRequest, c)

		messageInterface := getJsonPathResult(t, b, "$.message").(interface{})
		messageString := utils.InterfaceToString(messageInterface)
		assert.Equal(t, "Media url is not allowed in image src: http://malicious.example.com/virus.jpg", messageString)
	})
}

func TestNotPossibleToEditAMessageAndSetNotAllowedMediaUrl(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		c1, b1, _ := request("POST", "/api/chat/1/message", strings.NewReader(`{"text": "Lorem ipsum"}`), e)
		assert.Equal(t, http.StatusCreated, c1)
		idInterface := getJsonPathResult(t, b1, "$.id").(interface{})
		idString := utils.InterfaceToString(idInterface)

		c2, b2, _ := request("PUT", "/api/chat/1/message", strings.NewReader(fmt.Sprintf(`{ "id": %v, "text": "<img src=\"http://malicious.example.com/virus.jpg\"> Lorem ipsum"}`, idString)), e)
		assert.Equal(t, http.StatusBadRequest, c2)

		messageInterface := getJsonPathResult(t, b2, "$.message").(interface{})
		messageString := utils.InterfaceToString(messageInterface)
		assert.Equal(t, "Media url is not allowed in image src: http://malicious.example.com/virus.jpg", messageString)
	})
}

func TestItIsNotPossibleToWriteToForeignChat(t *testing.T) {
	h1 := map[string][]string{
		echo.HeaderContentType: {"application/json"},
		"X-Auth-Expiresin":     {"1590022342295000"},
		"X-Auth-Username":      {userTester}, // tester
		"X-Auth-Userid":        {"1"},
	}
	h2 := map[string][]string{
		echo.HeaderContentType: {"application/json"},
		"X-Auth-Expiresin":     {"1590022342295000"},
		"X-Auth-Username":      {userTester2}, // tester2
		"X-Auth-Userid":        {"2"},
	}

	runTest(t, func(e *echo.Echo, db *db.DB) {
		c, b, _ := requestWithHeader("POST", "/api/chat", h2, strings.NewReader(`{"name": "Chat of second user"}`), e)
		assert.Equal(t, http.StatusCreated, c)
		idInterface := getJsonPathResult(t, b, "$.id").(interface{})
		idString := utils.InterfaceToString(idInterface)

		// test not found
		c3, _, _ := requestWithHeader("GET", "/api/chat/"+idString+"/message/666", h2, nil, e)
		assert.Equal(t, http.StatusNotFound, c3)

		// first user tries to write to second user's chat
		c2, b2, _ := requestWithHeader("POST", "/api/chat/"+idString+"/message", h1, strings.NewReader(`{"text": "Ultra new message to the foreign chat"}`), e)
		assert.Equal(t, http.StatusBadRequest, c2)
		messageString := utils.InterfaceToString(getJsonPathResult(t, b2, "$.message").(interface{}))
		assert.Equal(t, "You are not allowed to write to this chat", messageString)
	})
}

func TestGetBlogsPaginated(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		httpFirstPage, bodyFirstPage, _ := request("GET", "/api/blog?page=2&size=3", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)

		firstResult := handlers.BlogPostsDTO{}
		err := json.Unmarshal([]byte(bodyFirstPage), &firstResult)
		assert.NoError(t, err)

		firstPageResult := firstResult.Items

		assert.Equal(t, 3, len(firstPageResult))
		assert.Equal(t, int64(994), firstPageResult[0].Id)
		assert.Equal(t, int64(993), firstPageResult[1].Id)
		assert.Equal(t, int64(992), firstPageResult[2].Id)
		assert.Equal(t, "generated_chat994", firstPageResult[0].Title)
		assert.Equal(t, "generated_chat993", firstPageResult[1].Title)
		assert.Equal(t, "generated_chat992", firstPageResult[2].Title)
	})
}

func TestGetBlogsPaginatedSearch(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		httpFirstPage, bodyFirstPage, _ := request("GET", "/api/blog?size=3&searchString=generated_chat994", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)

		firstResult := handlers.BlogPostsDTO{}
		err := json.Unmarshal([]byte(bodyFirstPage), &firstResult)
		assert.NoError(t, err)

		firstPageResult := firstResult.Items

		assert.Equal(t, 1, len(firstPageResult))
		assert.Equal(t, int64(994), firstPageResult[0].Id)
		assert.Equal(t, "generated_chat994", firstPageResult[0].Title)
	})
}
