package handlers

import (
	"errors"
	"fmt"
	"github.com/centrifugal/centrifuge"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	. "nkonev.name/chat/logger"
	name_nkonev_aaa "nkonev.name/chat/proto"
	"nkonev.name/chat/utils"
)

type Participant struct {
	Id     int64       `json:"id"`
	Login  string      `json:"login"`
	Avatar null.String `json:"avatar"`
}

type ChatDto struct {
	Id             int64         `json:"id"`
	Name           string        `json:"name"`
	ParticipantIds []int64       `json:"participantIds"`
	Participants   []Participant `json:"participants"`
}

type EditChatDto struct {
	Id             int64   `json:"id"`
	Name           string  `json:"name"`
	ParticipantIds []int64 `json:"participantIds"`
}

type CreateChatDto struct {
	Name           string  `json:"name"`
	ParticipantIds []int64 `json:"participantIds"`
}

func (a *CreateChatDto) Validate() error {
	return validation.ValidateStruct(a, validation.Field(&a.Name, validation.Required, validation.Length(1, 256)))
}

func (a *EditChatDto) Validate() error {
	return validation.ValidateStruct(a, validation.Field(&a.Name, validation.Required, validation.Length(1, 256)), validation.Field(&a.Id, validation.Required))
}

func GetChats(db db.DB, restClient client.RestClient) func(c echo.Context) error {
	return func(c echo.Context) error {
		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		page := utils.FixPageString(c.QueryParam("page"))
		size := utils.FixSizeString(c.QueryParam("size"))
		offset := utils.GetOffset(page, size)

		if chats, err := db.GetChats(userPrincipalDto.UserId, size, offset); err != nil {
			GetLogEntry(c.Request()).Errorf("Error get chats from db %v", err)
			return err
		} else {
			chatDtos := make([]*ChatDto, 0)
			for _, cc := range chats {
				if ids, err := db.GetParticipantIds(cc.Id); err != nil {
					return err
				} else {
					if users, err := restClient.GetUsers(ids); err != nil {
						GetLogEntry(c.Request()).Errorf("Error get participants for chat id=%v %v", cc.Id, err)
						chatDtos = append(chatDtos, convertToDto(cc, ids, nil))
					} else {
						chatDtos = append(chatDtos, convertToDto(cc, ids, users))
					}
				}
			}
			GetLogEntry(c.Request()).Infof("Successfully returning %v chats", len(chatDtos))
			return c.JSON(200, chatDtos)
		}
	}
}

func GetChat(dbR db.DB, restClient client.RestClient) func(c echo.Context) error {
	return func(c echo.Context) error {
		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		chatId, err := GetPathParamAsInt64(c, "id")
		if err != nil {
			return err
		}

		if chat, err := dbR.GetChat(userPrincipalDto.UserId, chatId); err != nil {
			GetLogEntry(c.Request()).Errorf("Error get chats from db %v", err)
			return err
		} else {
			if chat == nil {
				return c.NoContent(http.StatusNotFound)
			}
			var chatDto *ChatDto
			if ids, err := dbR.GetParticipantIds(chatId); err != nil {
				return err
			} else {
				if users, err := restClient.GetUsers(ids); err != nil {
					GetLogEntry(c.Request()).Errorf("Error get participants for chat id=%v %v", chatId, err)
					chatDto = convertToDto(chat, ids, nil)
				} else {
					chatDto = convertToDto(chat, ids, users)
				}
			}

			GetLogEntry(c.Request()).Infof("Successfully returning %v chat", chatDto)
			return c.JSON(200, chatDto)
		}
	}
}

func convertToParticipant(user *name_nkonev_aaa.UserDto) Participant {
	var nullableAvatar = null.NewString(user.Avatar, user.Avatar != "")
	return Participant{
		Id:     user.Id,
		Login:  user.Login,
		Avatar: nullableAvatar,
	}
}

func convertToDto(c *db.Chat, participantIds []int64, users []*name_nkonev_aaa.UserDto) *ChatDto {
	var participants []Participant

	for _, u := range users {
		participant := convertToParticipant(u)
		participants = append(participants, participant)
	}

	return &ChatDto{
		Id:             c.Id,
		Name:           c.Title,
		ParticipantIds: participantIds,
		Participants:   participants,
	}
}

func CreateChat(dbR db.DB, node *centrifuge.Node, restClient client.RestClient) func(c echo.Context) error {
	return func(c echo.Context) error {
		var bindTo = new(CreateChatDto)
		if err := c.Bind(bindTo); err != nil {
			GetLogEntry(c.Request()).Errorf("Error during binding to dto %v", err)
			return err
		}
		if valid, err := ValidateAndRespondError(c, bindTo); err != nil || !valid {
			return err
		}

		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		var participantsForNotify []int64
		result, errOuter := db.TransactWithResult(dbR, func(tx *db.Tx) (interface{}, error) {
			id, err := tx.CreateChat(convertToCreatableChat(bindTo))
			if err != nil {
				return 0, err
			}
			if err := tx.AddParticipant(userPrincipalDto.UserId, id, true); err != nil {
				return 0, err
			}
			participantsForNotify = append(participantsForNotify, userPrincipalDto.UserId)
			for _, participantId := range bindTo.ParticipantIds {
				if participantId == userPrincipalDto.UserId {
					continue
				}
				if err := tx.AddParticipant(participantId, id, false); err != nil {
					return 0, err
				}
				participantsForNotify = append(participantsForNotify, participantId)
			}
			ids, err := tx.GetParticipantIdsTx(id)
			if err != nil {
				return nil, err
			}
			//responseDto := ChatDto{
			//	Id:             id,
			//	Name:           bindTo.Name,
			//	ParticipantIds: ids,
			//}
			var responseDto ChatDto
			if users, err := restClient.GetUsers(ids); err != nil {
				GetLogEntry(c.Request()).Errorf("Error get participants for chat id=%v %v", id, err)
				responseDto = ChatDto{
					Id:             id,
					Name:           bindTo.Name,
					ParticipantIds: ids,
				}
			} else {
				var participants []Participant
				for _, u := range users {
					participant := convertToParticipant(u)
					participants = append(participants, participant)
				}
				responseDto = ChatDto{
					Id:             id,
					Name:           bindTo.Name,
					ParticipantIds: ids,
					Participants:   participants,
				}
			}

			return responseDto, err
		})
		if errOuter != nil {
			GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
			return errOuter
		} else {
			typedResult := result.(ChatDto)
			// send start
			for _, participantId := range participantsForNotify {
				participantChannel := node.PersonalChannel(utils.Int64ToString(participantId))
				GetLogEntry(c.Request()).Infof("Sending notification about create the chat to participantChannel: %v", participantChannel)
				_, err := node.Publish(participantChannel, []byte(`{"messageChatId": `+utils.InterfaceToString(typedResult.Id)+`, "type": "chat_created"}`))
				if err != nil {
					GetLogEntry(c.Request()).Errorf("error publishing to personal channel: %s", err)
				}
			}
			// send end
			return c.JSON(http.StatusCreated, typedResult)
		}
	}
}

func convertToCreatableChat(d *CreateChatDto) *db.Chat {
	return &db.Chat{
		Title: d.Name,
	}
}

func DeleteChat(dbR db.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		chatId, err := GetPathParamAsInt64(c, "id")
		if err != nil {
			return err
		}

		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		errOuter := db.Transact(dbR, func(tx *db.Tx) error {
			if admin, err := tx.IsAdmin(userPrincipalDto.UserId, chatId); err != nil {
				return err
			} else if !admin {
				return errors.New(fmt.Sprintf("User %v is not admin of chat %v", userPrincipalDto.UserId, chatId))
			}
			if err := tx.DeleteChat(chatId); err != nil {
				return err
			}
			return c.JSON(http.StatusAccepted, &utils.H{"id": chatId})
		})
		if errOuter != nil {
			GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
		}
		return errOuter
	}
}

func EditChat(dbR db.DB, restClient client.RestClient) func(c echo.Context) error {
	return func(c echo.Context) error {
		var bindTo = new(EditChatDto)
		if err := c.Bind(bindTo); err != nil {
			GetLogEntry(c.Request()).Errorf("Error during binding to dto %v", err)
			return err
		}

		if valid, err := ValidateAndRespondError(c, bindTo); err != nil || !valid {
			return err
		}

		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		errOuter := db.Transact(dbR, func(tx *db.Tx) error {
			if admin, err := tx.IsAdmin(userPrincipalDto.UserId, bindTo.Id); err != nil {
				return err
			} else if !admin {
				return errors.New(fmt.Sprintf("User %v is not admin of chat %v", userPrincipalDto.UserId, bindTo.Id))
			}
			if err := tx.EditChat(bindTo.Id, bindTo.Name); err != nil {
				return err
			}
			// TODO re-think about case when non-admin is trying to edit
			if err := tx.DeleteParticipantsExcept(userPrincipalDto.UserId, bindTo.Id); err != nil {
				return err
			}
			for _, participantId := range bindTo.ParticipantIds {
				if participantId == userPrincipalDto.UserId {
					continue
				}
				if err := tx.AddParticipant(participantId, bindTo.Id, false); err != nil {
					return err
				}
			}

			ids, err := tx.GetParticipantIdsTx(bindTo.Id)
			if err != nil {
				return err
			}
			var responseDto ChatDto
			if users, err := restClient.GetUsers(ids); err != nil {
				GetLogEntry(c.Request()).Errorf("Error get participants for chat id=%v %v", bindTo.Id, err)
				responseDto = ChatDto{
					Id:             bindTo.Id,
					Name:           bindTo.Name,
					ParticipantIds: ids,
				}
			} else {
				var participants []Participant
				for _, u := range users {
					participant := convertToParticipant(u)
					participants = append(participants, participant)
				}
				responseDto = ChatDto{
					Id:             bindTo.Id,
					Name:           bindTo.Name,
					ParticipantIds: ids,
					Participants:   participants,
				}
			}

			return c.JSON(http.StatusAccepted, responseDto)
		})
		if errOuter != nil {
			GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
		}
		return errOuter
	}
}
