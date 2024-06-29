package handlers

import (
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
	"net/url"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
	"nkonev.name/chat/utils"
	"sort"
	"strings"
	"time"
)

type BlogHandler struct {
	db              *db.DB
	notificator     *services.Events
	policy          *services.SanitizerPolicy
	stripTagsPolicy *services.StripTagsPolicy
	restClient      *client.RestClient
}

func NewBlogHandler(db *db.DB, notificator *services.Events, policy *services.SanitizerPolicy, stripTagsPolicy *services.StripTagsPolicy, restClient *client.RestClient) *BlogHandler {
	return &BlogHandler{
		db:              db,
		notificator:     notificator,
		policy:          policy,
		stripTagsPolicy: stripTagsPolicy,
		restClient:      restClient,
	}
}

type BlogPostPreviewDto struct {
	Id             int64     `json:"id"` // chatId
	Title          string    `json:"title"`
	CreateDateTime time.Time `json:"createDateTime"`
	OwnerId        *int64    `json:"ownerId"`
	Owner          *dto.User `json:"owner"`
	MessageId      *int64    `json:"messageId"`
	Text           *string   `json:"-"`
	Preview        *string   `json:"preview"`
	ImageUrl       *string   `json:"imageUrl"`
}

func (h *BlogHandler) getPostsWoUsers(ctx context.Context, blogs []*db.Blog) ([]*BlogPostPreviewDto, error) {
	var response = make([]*BlogPostPreviewDto, 0)

	var blogIds []int64 = make([]int64, 0)
	for _, blog := range blogs {
		blogIds = append(blogIds, blog.Id)
	}

	// get their message where blog_post=true for sake to make preview
	posts, err := h.db.GetBlogPostsByChatIds(blogIds)
	if err != nil {
		return response, err
	}
	for _, blog := range blogs {

		blogPost := &BlogPostPreviewDto{
			Id:             blog.Id,
			CreateDateTime: blog.CreateDateTime,
			Title:          blog.Title,
		}

		for _, post := range posts {
			if post.ChatId == blog.Id {
				mbImage := h.tryGetFirstImage(ctx, post.Text)
				if mbImage != nil {
					fileParam, err := h.getFileParam(*mbImage)
					if err != nil {
						GetLogEntry(ctx).Warnf("Unagle to get file key: %v", err)
						break
					}
					if len(fileParam) > 0 {
						dumbUrl := url.URL{}
						query := dumbUrl.Query()
						query.Set(utils.FileParam, utils.SetImagePreviewExtension(fileParam))
						dumbUrl.RawQuery = query.Encode()

						publicPreviewUrl, err := makeUrlPublic(dumbUrl.String(), utils.UrlStorageEmbedPreview, false, post.MessageId)
						if err != nil {
							GetLogEntry(ctx).Warnf("Unagle to change url: %v", err)
							break
						}
						blogPost.ImageUrl = &publicPreviewUrl
					}
				}

				if blogPost.ImageUrl == nil {
					blogPost.ImageUrl = blog.Avatar.Ptr()
				}

				t := post.Text
				blogPost.Text = &t
				blogPost.Preview = h.cutText(post.Text)
				oid := post.OwnerId
				blogPost.OwnerId = &oid
				mid := post.MessageId
				blogPost.MessageId = &mid
				break
			}
		}

		response = append(response, blogPost)
	}
	return response, nil
}

func (h *BlogHandler) GetBlogPosts(c echo.Context) error {

	size := utils.FixSizeString(c.QueryParam("size"))
	page := utils.FixPageString(c.QueryParam("page"))
	searchString := c.QueryParam("searchString")

	var response = make([]*BlogPostPreviewDto, 0)

	isSearch := false
	if len(searchString) != 0 {
		isSearch = true
	}

	var count int64

	if !isSearch {
		portionOffset := utils.GetOffset(page, size)
		blogs, err := h.db.GetBlogPostsByLimitOffset(false, size, portionOffset)
		if err != nil {
			return err
		}

		count, err = h.db.CountBlogs()
		if err != nil {
			return err
		}

		response, err = h.getPostsWoUsers(c.Request().Context(), blogs)
		if err != nil {
			return err
		}
	} else { // search

		offset := page * size

		var offsetCounter = 0

		shouldIterate := true
		for portionPage := 0; shouldIterate; portionPage++ {
			portionOffset := utils.GetOffset(portionPage, size)
			// get chats where blog=true
			blogs, err := h.db.GetBlogPostsByLimitOffset(false, size, portionOffset)
			if err != nil {
				return err
			}

			portion, err := h.getPostsWoUsers(c.Request().Context(), blogs)
			if err != nil {
				return err
			}
			if len(portion) < size {
				shouldIterate = false
			}

			searched, err := h.performSearch(searchString, portion)
			if err != nil {
				return err
			}

			count += int64(len(searched))

			for _, sp := range searched {
				if offsetCounter >= offset {
					if len(response) < size {
						response = append(response, sp)
					}
				}
				offsetCounter++
			}
		}
	}

	var participantIdSet = map[int64]bool{}
	for _, respDto := range response {
		if respDto.OwnerId != nil {
			participantIdSet[*respDto.OwnerId] = true
		}
	}
	var users = getUsersRemotelyOrEmpty(participantIdSet, h.restClient, c)
	for _, respDto := range response {
		if respDto.OwnerId != nil {
			respDto.Owner = users[*respDto.OwnerId]
		}
	}

	pagesCount := count / int64(size)
	if count % int64(size) > 0 {
		pagesCount++
	}

	return c.JSON(http.StatusOK, &BlogPostsDTO{
		Items: response,
		Count: count,
		PagesCount: pagesCount,
	})
}

type BlogPostsDTO struct {
	Items []*BlogPostPreviewDto `json:"items"`
	Count int64 `json:"count"`
	PagesCount int64 `json:"pagesCount"`
}

type BlogSeoItem struct {
	ChatId int64 `json:"chatId"`
	LastModified time.Time `json:"lastModified"`
}

func (h *BlogHandler) GetAllBlogPostsForSeo(c echo.Context) error {

	page := utils.FixPageString(c.QueryParam("page"))
	size := utils.FixSizeString(c.QueryParam("size"))

	posts, err := h.db.GetBlogPostsByLimitOffset(false, size, page*size)
	if err != nil {
		return err
	}

	var chatIds = make([]int64, 0)
	for _, post := range posts {
		chatIds = append(chatIds, post.Id)
	}

	dates, err := h.db.GetBlobPostModifiedDates(chatIds)
	if err != nil {
		return err
	}

	res := make([]BlogSeoItem, 0)
	for chatId, aDate := range dates {
		res = append(res, BlogSeoItem{
			ChatId:       chatId,
			LastModified: aDate,
		})
	}

	sort.SliceStable(res, func(i, j int) bool {
		return res[i].LastModified.Unix() > res[j].LastModified.Unix()
	})

	return c.JSON(http.StatusOK, res)
}

func appendKeepingN(input []*BlogPostPreviewDto, post *BlogPostPreviewDto, sizeToKeep int) ([]*BlogPostPreviewDto, int) {
	var response []*BlogPostPreviewDto = make([]*BlogPostPreviewDto, 0)
	if len(input) < sizeToKeep {
		for _, pp := range input {
			response = append(response, pp)
		}
	} else {
		// copy last sizeToKeep - 1 items to response // + 1 gives us the space for post
		leftIdx := len(input) + 1 - sizeToKeep
		tmpResponse := input[leftIdx:]
		for _, pp := range tmpResponse {
			response = append(response, pp)
		}
	}
	response = append(response, post)
	return response, len(response)
}

func (h *BlogHandler) performSearch(searchString string, searchable []*BlogPostPreviewDto) ([]*BlogPostPreviewDto, error) {
	searchString = strings.ToLower(searchString)

	var intermediateList = make([]*BlogPostPreviewDto, 0)

	for _, blogPostPreviewDto := range searchable {
		if strings.Contains(strings.ToLower(blogPostPreviewDto.Title), searchString) ||
			(blogPostPreviewDto.Preview != nil && strings.Contains(strings.ToLower(*blogPostPreviewDto.Text), searchString)) {
			intermediateList = append(intermediateList, blogPostPreviewDto)
		}
	}

	return intermediateList, nil
}

func (h *BlogHandler) tryGetFirstImage(ctx context.Context, text string) *string {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
	if err != nil {
		GetLogEntry(ctx).Warnf("Unagle to get image: %v", err)
		return nil
	}

	maybeImage := doc.Find("img").First()
	if maybeImage != nil {
		src, exists := maybeImage.Attr("src")
		if exists {
			return &src
		}
	}
	maybeVideo := doc.Find("video").First()
	if maybeVideo != nil {
		src, exists := maybeVideo.Attr("poster")
		if exists {
			return &src
		}
	}

	return nil
}

func (h *BlogHandler) cutText(text string) *string {
	ret := stripTagsAndCut(h.stripTagsPolicy, viper.GetInt("blogPreviewMaxTextSize"), text)
	return &ret
}

type BlogPostResponse struct {
	ChatId         int64     `json:"chatId"`
	Title          string    `json:"title"`
	OwnerId        *int64    `json:"ownerId"`
	Owner          *dto.User `json:"owner"`
	MessageId      *int64    `json:"messageId"`
	Text           *string   `json:"text"`
	CreateDateTime time.Time `json:"createDateTime"`
	Reactions 	   []dto.Reaction `json:"reactions"`
	Preview        *string   `json:"preview"`
}

func (h *BlogHandler) GetBlogPost(c echo.Context) error {
	blogId, err := utils.ParseInt64(c.Param("id"))
	if err != nil {
		return err
	}

	chatBasic, err := h.db.GetChatBasic(blogId)
	if err != nil {
		return err
	}
	if chatBasic == nil {
		return c.NoContent(http.StatusNoContent)
	}
	if !chatBasic.IsBlog {
		GetLogEntry(c.Request().Context()).Infof("This chat %v is not blog", blogId)
		return c.NoContent(http.StatusNoContent)
	}

	response := BlogPostResponse{
		ChatId:         chatBasic.Id,
		Title:          chatBasic.Title,
		CreateDateTime: chatBasic.CreateDateTime,
	}

	var post *db.BlogPost
	posts, err := h.db.GetBlogPostsByChatIds([]int64{blogId})
	if err != nil {
		return err
	}
	if len(posts) == 1 {
		post = posts[0]
	} else {
		GetLogEntry(c.Request().Context()).Infof("By blog id %v found not 1 message - %v", blogId, len(posts))
	}

	if post != nil {
		response.OwnerId = &post.OwnerId
		response.MessageId = &post.MessageId
		patchedText := PatchStorageUrlToPublic(c.Request().Context(), post.Text, post.MessageId)
		response.Text = &patchedText

		var participantIdSet = map[int64]bool{}
		participantIdSet[post.OwnerId] = true

		reactions, err := h.db.GetReactionsOnMessage(chatBasic.Id, post.MessageId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Infof("For blog id %v unable to get reactions: %v", blogId, err)
			return err
		}

		takeOnAccountReactions(participantIdSet, reactions) // adds reaction' users to participantIdSet

		var users = getUsersRemotelyOrEmpty(participantIdSet, h.restClient, c)

		user := users[post.OwnerId]
		response.Owner = user

		response.Reactions = convertReactions(reactions, users)

		response.Preview = h.cutText(post.Text)
	}

	return c.JSON(http.StatusOK, response)
}

func (h *BlogHandler) GetBlogPostComments(c echo.Context) error {
	blogId, err := utils.ParseInt64(c.Param("id"))
	if err != nil {
		return err
	}

	messageDtos := make([]*dto.DisplayMessageDto, 0)

	chatBasic, err := h.db.GetChatBasic(blogId)
	if err != nil {
		return err
	}
	if chatBasic == nil {
		return c.NoContent(http.StatusNoContent)
	}
	if !chatBasic.IsBlog {
		GetLogEntry(c.Request().Context()).Infof("This chat %v is not blog", blogId)
		return c.NoContent(http.StatusNoContent)
	}

	size := utils.FixSizeString(c.QueryParam("size"))
	page := utils.FixPageString(c.QueryParam("page"))
	portionOffset := utils.GetOffset(page, size)

	postMessageId, err := h.db.GetBlogPostMessageId(blogId)
	if err != nil {
		return err
	}

	var count int64
	count, err = h.db.CountComments(blogId, postMessageId)
	if err != nil {
		return err
	}

	messages, err := h.db.GetComments(blogId, postMessageId, size, portionOffset, false)
	if err != nil {
		return err
	}

	var ownersSet = map[int64]bool{}
	var chatsPreSet = map[int64]bool{}
	for _, message := range messages {
		populateSets(message, ownersSet, chatsPreSet, true)
	}
	chatsSet, err := h.db.GetChatsBasic(chatsPreSet, NonExistentUser)
	if err != nil {
		return err
	}
	var users = getUsersRemotelyOrEmpty(ownersSet, h.restClient, c)
	for _, cc := range messages {
		msg := convertToMessageDtoWithoutPersonalized(c.Request().Context(), cc, users, chatsSet)
		msg.Text = PatchStorageUrlToPublic(c.Request().Context(), msg.Text, msg.Id)
		messageDtos = append(messageDtos, msg)
	}

	pagesCount := count / int64(size)
	if count % int64(size) > 0 {
		pagesCount++
	}

	GetLogEntry(c.Request().Context()).Infof("Successfully returning %v messages", len(messageDtos))
	return c.JSON(http.StatusOK, &utils.H{"items": messageDtos, "count": count, "pagesCount": pagesCount})
}

// see also message.go :: patchStorageUrlToPreventCachingVideo
func PatchStorageUrlToPublic(ctx context.Context, text string, messageId int64) string {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
	if err != nil {
		GetLogEntry(ctx).Warnf("Unagle to read html: %v", err)
		return ""
	}

	wlArr := []string{"", viper.GetString("frontendUrl")}

	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		maybeImage := s.First()
		if maybeImage != nil {
			src, srcExists := maybeImage.Attr("src")
			if srcExists && utils.ContainsUrl(wlArr, src) {
				newurl, err := makeUrlPublic(src, "", false, messageId)
				if err != nil {
					GetLogEntry(ctx).Warnf("Unagle to change url: %v", err)
					return
				}
				maybeImage.SetAttr("src", newurl)
			}
		}
	})

	doc.Find("video").Each(func(i int, s *goquery.Selection) {
		maybeVideo := s.First()
		if maybeVideo != nil {
			src, srcExists := maybeVideo.Attr("src")
			if srcExists && utils.ContainsUrl(wlArr, src) {
				newurl, err := makeUrlPublic(src, "", true, messageId) // large video file doesn't fit in cache well, so in order not to cache it we add time
				if err != nil {
					GetLogEntry(ctx).Warnf("Unagle to change url: %v", err)
					return
				}
				maybeVideo.SetAttr("src", newurl)
			}

			poster, posterExists := maybeVideo.Attr("poster")
			if posterExists && utils.ContainsUrl(wlArr, src) {
				newurl, err := makeUrlPublic(poster, utils.UrlStorageEmbedPreview, false, messageId)
				if err != nil {
					GetLogEntry(ctx).Warnf("Unagle to change url: %v", err)
					return
				}
				maybeVideo.SetAttr("poster", newurl)
			}
		}
	})

	doc.Find("audio").Each(func(i int, s *goquery.Selection) {
		maybeVideo := s.First()
		if maybeVideo != nil {
			src, srcExists := maybeVideo.Attr("src")
			if srcExists && utils.ContainsUrl(wlArr, src) {
				newurl, err := makeUrlPublic(src, "", true, messageId)
				if err != nil {
					GetLogEntry(ctx).Warnf("Unagle to change url: %v", err)
					return
				}
				maybeVideo.SetAttr("src", newurl)
			}
		}
	})

	ret, err := doc.Find("html").Find("body").Html()
	if err != nil {
		GetLogEntry(ctx).Warnf("Unagle to write html: %v", err)
		return ""
	}
	return ret
}

func (h *BlogHandler) getFileParam(src string) (string, error) {
	parsed, err := url.Parse(src)
	if err != nil {
		return "", err
	}
	fileParam := parsed.Query().Get(utils.FileParam)
	return fileParam, nil
}

func makeUrlPublic(src string, additionalSegment string, addTime bool, messageId int64) (string, error) {
	// we add time in order not to cache the video itself
	parsed, err := url.Parse(src)
	if err != nil {
		return "", err
	}

	parsed.Path = utils.UrlApiPrefix + utils.UrlStoragePublicGetFile + additionalSegment

	query := parsed.Query()

	if addTime {
		addTimeToUrlValues(&query)
	}

	query.Set("messageId", utils.Int64ToString(messageId))

	parsed.RawQuery = query.Encode()

	newurl := parsed.String()
	return newurl, nil
}
