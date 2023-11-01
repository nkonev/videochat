package handlers

import (
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
	"strings"
	"time"
)

type BlogHandler struct {
	db              *db.DB
	notificator     services.Events
	policy          *services.SanitizerPolicy
	stripTagsPolicy *services.StripTagsPolicy
	restClient      *client.RestClient
}

func NewBlogHandler(db *db.DB, notificator services.Events, policy *services.SanitizerPolicy, stripTagsPolicy *services.StripTagsPolicy, restClient *client.RestClient) *BlogHandler {
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

func getSize(size int, isSearch bool) int {
	if isSearch {
		return viper.GetInt("blogSearchSize")
	} else {
		return size
	}
}

func getOffset(offset int, isSearch bool) int {
	if isSearch {
		return 0
	} else {
		return offset
	}
}

func (h *BlogHandler) GetBlogPosts(c echo.Context) error {

	page := utils.FixPageString(c.QueryParam("page"))
	size := utils.FixSizeString(c.QueryParam("size"))
	offset := utils.GetOffset(page, size)
	searchString := c.QueryParam("searchString")
	searchString = strings.TrimSpace(searchString)

	isSearch := false

	if len(searchString) != 0 {
		isSearch = true
	}

	// get chats where blog=true
	blogs, err := h.db.GetBlogPostsByLimitOffset(getSize(size, isSearch), getOffset(offset, isSearch))
	if err != nil {
		return err
	}

	var blogIds []int64 = make([]int64, 0)
	for _, blog := range blogs {
		blogIds = append(blogIds, blog.Id)
	}

	// get their message where blog_post=true for sake to make preview
	posts, err := h.db.BlogPosts(blogIds)
	if err != nil {
		return err
	}
	var response = make([]*BlogPostPreviewDto, 0)
	for _, blog := range blogs {

		blogPost := &BlogPostPreviewDto{
			Id:             blog.Id,
			CreateDateTime: blog.CreateDateTime,
			Title:          blog.Title,
		}

		for _, post := range posts {
			if post.ChatId == blog.Id {
				mbImage := h.tryGetFirstImage(post.Text)
				if mbImage != nil {
					tmpVar, err := h.getFileParam(*mbImage)
					if err != nil {
						Logger.Warnf("Unagle to get file key: %v", err)
						break
					}
					tmp2Var := h.getPreviewUrl(tmpVar)

					blogPost.ImageUrl = tmp2Var
				} else {
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

	if isSearch {
		search, err := h.performSearchAndPaging(searchString, response, size, offset)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, search)
	} else {
		return c.JSON(http.StatusOK, response)
	}

}

func (h *BlogHandler) GetBlogPage(c echo.Context) error {
	itemId, err := utils.ParseInt64(c.QueryParam("id"))
	if err != nil {
		return err
	}
	size := utils.FixSizeString(c.QueryParam("size"))
	searchString := c.QueryParam("searchString")
	searchString = strings.TrimSpace(searchString)

	return db.Transact(h.db, func(tx *db.Tx) error {
		rowNumber, err := tx.GetBlogRowNumber(itemId, searchString)
		if err != nil {
			return err
		}

		var returnPage = (rowNumber + 1) / size

		return c.JSON(http.StatusOK, &utils.H{"page": returnPage})
	})
}

func (h *BlogHandler) getPreviewUrl(aKey string) *string {
	var previewUrl *string = nil

	respUrl := url.URL{}
	respUrl.Path = "/api/storage/public/download/embed/preview"
	previewMinioKey := ""
	previewMinioKey = utils.SetImagePreviewExtension(aKey)
	if previewMinioKey != "" {
		query := respUrl.Query()
		query.Set(utils.FileParam, previewMinioKey)

		respUrl.RawQuery = query.Encode()

		tmp := respUrl.String()
		previewUrl = &tmp
	} else {
		Logger.Errorf("Unable to make previewUrl for %v", previewMinioKey)
	}

	return previewUrl
}

func (h *BlogHandler) performSearchAndPaging(searchString string, searchable []*BlogPostPreviewDto, size, offset int) ([]*BlogPostPreviewDto, error) {
	searchString = strings.ToLower(searchString)

	var intermediateList = make([]*BlogPostPreviewDto, 0)

	for _, blogPostPreviewDto := range searchable {
		if strings.Contains(strings.ToLower(blogPostPreviewDto.Title), searchString) ||
			(blogPostPreviewDto.Preview != nil && strings.Contains(strings.ToLower(*blogPostPreviewDto.Text), searchString)) {
			intermediateList = append(intermediateList, blogPostPreviewDto)
		}
	}

	var list = make([]*BlogPostPreviewDto, 0)
	var counter = 0
	var respCounter = 0

	for _, objInfo := range intermediateList {
		if counter >= offset {
			list = append(list, objInfo)
			respCounter++
			if respCounter >= size {
				break
			}
		}
		counter++
	}

	return list, nil
}

func (h *BlogHandler) tryGetFirstImage(text string) *string {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
	if err != nil {
		Logger.Warnf("Unagle to get image: %v", err)
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
	tmp := h.stripTagsPolicy.Sanitize(text)
	runes := []rune(tmp)
	sizeToCut := viper.GetInt("blogPreviewMaxTextSize")
	textLen := len(runes)
	size := utils.Min(sizeToCut, textLen)
	ret := string(runes[:size])
	if textLen > sizeToCut {
		ret += "..."
	}
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
		return c.NoContent(http.StatusNotFound)
	}
	if !chatBasic.IsBlog {
		GetLogEntry(c.Request().Context()).Infof("This chat %v is not blog", blogId)
		return c.NoContent(http.StatusUnauthorized)
	}

	response := BlogPostResponse{
		ChatId:         chatBasic.Id,
		Title:          chatBasic.Title,
		CreateDateTime: chatBasic.CreateDateTime,
	}

	post, err := h.getBlogPost(blogId)
	if err != nil {
		return err
	}
	if post != nil {
		response.OwnerId = &post.OwnerId
		response.MessageId = &post.MessageId
		patchedText := h.patchStorageUrlToPublic(post.Text)
		response.Text = &patchedText

		var participantIdSet = map[int64]bool{}
		participantIdSet[post.OwnerId] = true
		var users = getUsersRemotelyOrEmpty(participantIdSet, h.restClient, c)

		if len(users) == 1 {
			user := users[post.OwnerId]
			response.Owner = user
		}
	}

	return c.JSON(http.StatusOK, response)
}

func (h *BlogHandler) getBlogPost(blogId int64) (*db.BlogPost, error) {
	posts, err := h.db.BlogPosts([]int64{blogId})
	if err != nil {
		return nil, err
	}
	if len(posts) == 1 {
		post := posts[0]
		return post, nil
	}
	return nil, nil
}

func (h *BlogHandler) GetBlogPostComments(c echo.Context) error {
	blogId, err := utils.ParseInt64(c.Param("id"))
	if err != nil {
		return err
	}

	chatBasic, err := h.db.GetChatBasic(blogId)
	if err != nil {
		return err
	}
	if chatBasic == nil {
		return c.NoContent(http.StatusNotFound)
	}
	if !chatBasic.IsBlog {
		GetLogEntry(c.Request().Context()).Infof("This chat %v is not blog", blogId)
		return c.NoContent(http.StatusUnauthorized)
	}

	startingFromItemId, err := utils.ParseInt64(c.QueryParam("startingFromItemId"))
	if err != nil {
		return err
	}
	size := utils.FixSizeString(c.QueryParam("size"))
	reverse := utils.GetBoolean(c.QueryParam("reverse"))

	post, err := h.getBlogPost(blogId)
	if err != nil {
		return err
	}
	if post == nil {
		GetLogEntry(c.Request().Context()).Infof("Messages of this chat %v does no have blog posts", blogId)
		return c.NoContent(http.StatusNoContent)
	}
	messages, err := h.db.GetComments(blogId, post.MessageId, size, startingFromItemId, reverse)
	if err != nil {
		return err
	}

	var ownersSet = map[int64]bool{}
	var chatsPreSet = map[int64]bool{}
	for _, message := range messages {
		populateSets(message, ownersSet, chatsPreSet)
	}
	chatsSet, err := h.db.GetChatsBasic(chatsPreSet, NonExistentUser)
	if err != nil {
		return err
	}
	var owners = getUsersRemotelyOrEmpty(ownersSet, h.restClient, c)
	messageDtos := make([]*dto.DisplayMessageDto, 0)
	for _, cc := range messages {
		msg := convertToMessageDto(cc, owners, chatsSet, NonExistentUser)
		msg.Text = h.patchStorageUrlToPublic(msg.Text)
		messageDtos = append(messageDtos, msg)
	}

	GetLogEntry(c.Request().Context()).Infof("Successfully returning %v messages", len(messageDtos))
	return c.JSON(http.StatusOK, messageDtos)
}

func (h *BlogHandler) patchStorageUrlToPublic(text string) string {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
	if err != nil {
		Logger.Warnf("Unagle to read html: %v", err)
		return ""
	}

	wlArr := []string{"", viper.GetString("baseUrl")}

	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		maybeImage := s.First()
		if maybeImage != nil {
			src, srcExists := maybeImage.Attr("src")
			if srcExists && utils.ContainsUrl(wlArr, src) {
				newurl, err := h.makeUrlPublic(src, "", false)
				if err != nil {
					Logger.Warnf("Unagle to change url: %v", err)
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
				newurl, err := h.makeUrlPublic(src, "", true)
				if err != nil {
					Logger.Warnf("Unagle to change url: %v", err)
					return
				}
				maybeVideo.SetAttr("src", newurl)
			}

			poster, posterExists := maybeVideo.Attr("poster")
			if posterExists && utils.ContainsUrl(wlArr, src) {
				newurl, err := h.makeUrlPublic(poster, "/embed/preview", false)
				if err != nil {
					Logger.Warnf("Unagle to change url: %v", err)
					return
				}
				maybeVideo.SetAttr("poster", newurl)
			}
		}
	})

	ret, err := doc.Find("html").Find("body").Html()
	if err != nil {
		Logger.Warnf("Unagle to write html: %v", err)
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

func (h *BlogHandler) makeUrlPublic(src string, additionalSegment string, addTime bool) (string, error) {
	parsed, err := url.Parse(src)
	if err != nil {
		return "", err
	}

	parsed.Path = "/api" + utils.UrlStoragePublicGetFile + additionalSegment

	if addTime {
		query := parsed.Query()
		query.Set("time", utils.Int64ToString(time.Now().Unix()))
		parsed.RawQuery = query.Encode()
	}

	newurl := parsed.String()
	return newurl, nil
}
