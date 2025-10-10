package dto

import "time"

// list view
type BlogPostsDTO struct {
	Header     BlogHeader            `json:"header"`
	Items      []*BlogPostPreviewDto `json:"items"`
	Count      int64                 `json:"count"`
	PagesCount int64                 `json:"pagesCount"`
}

type BlogPostPreviewDto struct {
	Id             int64     `json:"id"` // chatId
	Title          string    `json:"title"`
	CreateDateTime time.Time `json:"createDateTime"`
	OwnerId        *int64    `json:"ownerId"`
	Owner          *User     `json:"owner"`
	MessageId      *int64    `json:"messageId"`
	Text           *string   `json:"-"`
	Preview        *string   `json:"preview"`
	ImageUrl       *string   `json:"imageUrl"`
}

// post view
type WrappedBlogPostResponse struct {
	Header          BlogHeader       `json:"header"`
	Post            BlogPostResponse `json:"post"`
	CanWriteMessage bool             `json:"canWriteMessage"`
}

type BlogPostResponse struct {
	ChatId         int64      `json:"chatId"`
	Title          string     `json:"title"`
	OwnerId        *int64     `json:"ownerId"`
	Owner          *User      `json:"owner"`
	MessageId      *int64     `json:"messageId"`
	Text           *string    `json:"text"`
	CreateDateTime time.Time  `json:"createDateTime"`
	Reactions      []Reaction `json:"reactions"`
	Preview        *string    `json:"preview"`
	FileItemUuid   *string    `json:"fileItemUuid"`
}

type CommentViewDto struct {
	Id             int64
	OwnerId        int64
	Content        string
	Embed          Embeddable
	FileItemUuid   *string
	CreateDateTime time.Time
	UpdateDateTime *time.Time
}

type CommentViewEnrichedDto struct {
	Id             int64                 `json:"id"`
	OwnerId        int64                 `json:"ownerId"`
	Content        string                `json:"text"`
	EmbedMessage   *EmbedMessageResponse `json:"embedMessage"`
	FileItemUuid   *string               `json:"fileItemUuid"`
	CreateDateTime time.Time             `json:"createDateTime"`
	UpdateDateTime *time.Time            `json:"editDateTime"` // for sake compatibility
	Owner          *User                 `json:"owner"`
	Reactions      []Reaction            `json:"reactions"`
}

type CommentsWrapper struct {
	Items      []CommentViewEnrichedDto `json:"items"`
	Count      int64                    `json:"count"`
	PagesCount int64                    `json:"pagesCount"`
}

type CanCreateBlogDto struct {
	CanCreateBlog bool `json:"canCreateBlog"`
}

type BlogHeader struct {
	AboutPostId    *int64  `json:"aboutPostId"`
	AboutPostTitle *string `json:"aboutPostTitle"`
}

type BlogSeoItem struct {
	ChatId       int64     `json:"chatId"`
	LastModified time.Time `json:"lastModified"`
}

type SeoBlogPosts struct {
	Items []BlogSeoItem `json:"items"`
}
