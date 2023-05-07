package handlers

import (
	"github.com/labstack/echo/v4"
	"nkonev.name/chat/db"
	"nkonev.name/chat/services"
)

type BlogHandler struct {
	db          *db.DB
	notificator services.Events
}

func NewBlogHandler(db *db.DB, notificator services.Events) *BlogHandler {
	return &BlogHandler{
		db:          db,
		notificator: notificator,
	}
}

func (h BlogHandler) CreateBlog(c echo.Context) error {
	// auth check
	// db.CreateChat (blog=true)
	// db.InsertMessage
}

func (h BlogHandler) DeleteBlog(c echo.Context) error {
	// auth check
	// db.DeleteChat
}

func (h BlogHandler) GetBlogs(c echo.Context) error {
	// auth check
	// get chats where blog=true
	// get their message where blog_post=true for sake to make preview
}

func (h BlogHandler) GetComments(c echo.Context) error {
	// auth check
	// get messages where blog_post=false for sake to make preview
}

func (h BlogHandler) MakeBlog(c echo.Context) error {
	// auth check
	// set chat's blog=true
	// find the message where blog_post=true
}
