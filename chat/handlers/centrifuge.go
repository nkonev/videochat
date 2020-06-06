package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/centrifugal/centrifuge"
	"github.com/centrifugal/protocol"
	"go.uber.org/fx"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"time"
)

func handleLog(e centrifuge.LogEntry) {
	Logger.Printf("%s: %v", e.Message, e.Fields)
}

func getChanPresenceStats(engine centrifuge.Engine, client *centrifuge.Client, e interface{}) *centrifuge.PresenceStats {
	var channel string
	switch v := e.(type) {
	case centrifuge.SubscribeEvent:
		channel = v.Channel
		break
	case centrifuge.UnsubscribeEvent:
		channel = v.Channel
		break
	default:
		Logger.Errorf("Unknown type of event")
		return nil
	}
	stats, err := engine.PresenceStats(channel)
	if err != nil {
		Logger.Errorf("Error during get stats %v", err)
	}
	Logger.Printf("client id=%v, userId=%v acting with channel %s, channelStats.NumUsers %v", client.ID(), client.UserID(), channel, stats.NumUsers)
	return &stats
}

func createPresence(credso *centrifuge.Credentials, client *centrifuge.Client) (*protocol.ClientInfo, time.Duration, error) {
	expiresInString := fmt.Sprintf("%v000", credso.ExpireAt) // to milliseconds for put into dateparse.ParseLocal
	t, err0 := dateparse.ParseLocal(expiresInString)
	if err0 != nil {
		return nil, 0, err0
	}

	presenceDuration := t.Sub(time.Now())
	Logger.Debugf("Calculated session duration %v for credentials %v", presenceDuration, credso)

	clientInfo := &protocol.ClientInfo{
		User:   client.ID(),
		Client: client.UserID(),
	}
	Logger.Infof("Created ClientInfo(Client: %v, UserId: %v)", client.ID(), client.UserID())
	return clientInfo, presenceDuration, nil
}

type PassData struct {
	Payload  utils.H `json:"payload"`
	Metadata utils.H `json:"metadata"`
}

func modifyMessage(msg []byte, originatorUserId string) ([]byte, error) {
	var v = &PassData{}
	if err := json.Unmarshal(msg, v); err != nil {
		return nil, err
	}
	v.Metadata = utils.H{"originatorUserId": originatorUserId}
	return json.Marshal(v)
}

func ConfigureCentrifuge(lc fx.Lifecycle) *centrifuge.Node {
	// We use default config here as starting point. Default config contains
	// reasonable values for available options.
	cfg := centrifuge.DefaultConfig
	// In this example we want client to do all possible actions with server
	// without any authentication and authorization. Insecure flag DISABLES
	// many security related checks in library. This is only to make example
	// short. In real app you most probably want authenticate and authorize
	// access to server. See godoc and examples in repo for more details.
	cfg.ClientInsecure = false
	// By default clients can not publish messages into channels. Setting this
	// option to true we allow them to publish.
	cfg.Publish = true

	// Centrifuge library exposes logs with different log level. In your app
	// you can set special function to handle these log entries in a way you want.
	cfg.LogLevel = centrifuge.LogLevelDebug
	cfg.LogHandler = handleLog

	// Node is the core object in Centrifuge library responsible for many useful
	// things. Here we initialize new Node instance and pass config to it.
	node, _ := centrifuge.New(cfg)

	engine, _ := centrifuge.NewMemoryEngine(node, centrifuge.MemoryEngineConfig{})
	node.SetEngine(engine)

	// ClientConnected node event handler is a point where you generally create a
	// binding between Centrifuge and your app business logic. Callback function you
	// pass here will be called every time new connection established with server.
	// Inside this callback function you can set various event handlers for connection.
	node.On().ClientConnected(func(ctx context.Context, client *centrifuge.Client) {
		// Set Subscribe Handler to react on every channel subscription attempt
		// initiated by client. Here you can theoretically return an error or
		// disconnect client from server if needed. But now we just accept
		// all subscriptions.
		var credso, ok = centrifuge.GetCredentials(ctx)
		if !ok {
			Logger.Infof("Cannot extract credentials")
			return
		}
		Logger.Infof("Connected websocket centrifuge client hasCredentials %v, credentials %v", ok, credso)

		client.On().Subscribe(func(e centrifuge.SubscribeEvent) centrifuge.SubscribeReply {
			clientInfo, presenceDuration, err := createPresence(credso, client)
			if err != nil {
				Logger.Errorf("Error during creating presence %v", err)
				return centrifuge.SubscribeReply{Error: centrifuge.ErrorInternal}
			}
			err = engine.AddPresence(e.Channel, client.UserID(), clientInfo, presenceDuration)
			if err != nil {
				Logger.Errorf("Error during AddPresence %v", err)
			}
			Logger.Infof("Added presence for userId %v", client.UserID())
			getChanPresenceStats(engine, client, e)

			return centrifuge.SubscribeReply{}
		})

		client.On().Unsubscribe(func(e centrifuge.UnsubscribeEvent) centrifuge.UnsubscribeReply {
			err := engine.RemovePresence(e.Channel, client.UserID())
			if err != nil {
				Logger.Errorf("Error during RemovePresence %v", err)
			}
			Logger.Infof("Removed presence for userId %v", client.UserID())
			getChanPresenceStats(engine, client, e)

			return centrifuge.UnsubscribeReply{}
		})

		// Set Publish Handler to react on every channel Publication sent by client.
		// Inside this method you can validate client permissions to publish into
		// channel. But in our simple chat app we allow everyone to publish into
		// any channel.
		client.On().Publish(func(e centrifuge.PublishEvent) centrifuge.PublishReply {
			Logger.Printf("client %v publishes into channel %s: %s", credso.UserID, e.Channel, string(e.Data))
			message, err := modifyMessage(e.Data, e.Info.GetUser())
			if err != nil {
				Logger.Errorf("Error during modifyMessage %v", err)
				return centrifuge.PublishReply{Error: centrifuge.ErrorInternal}
			}
			return centrifuge.PublishReply{Data: message}
		})

		// Set Disconnect Handler to react on client disconnect events.
		client.On().Disconnect(func(e centrifuge.DisconnectEvent) centrifuge.DisconnectReply {
			Logger.Printf("client %v disconnected", credso.UserID)
			return centrifuge.DisconnectReply{}
		})

		// In our example transport will always be Websocket but it can also be SockJS.
		transportName := client.Transport().Name()
		// In our example clients connect with JSON protocol but it can also be Protobuf.
		transportEncoding := client.Transport().Encoding()
		Logger.Printf("client %v connected via %s (%s)", credso.UserID, transportName, transportEncoding)
	})

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// do some work on application stop (like closing connections and files)
			Logger.Infof("Stopping centrifuge")
			return node.Shutdown(ctx)
		},
	})

	return node
}
