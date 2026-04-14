package cqrs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"nkonev.name/chat/config"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/sanitizer"
	"nkonev.name/chat/utils"

	"github.com/PuerkitoBio/goquery"
	"github.com/jackc/pgtype"

	"github.com/georgysavva/scany/v2/sqlscan"
)

func (m *CommonProjection) refreshBlog(ctx context.Context, co db.CommonOperations, chatId int64, createdTime time.Time, blogAboutP *bool) (*int64, error) {
	var blogInpData = struct {
		ChatId      int64   `db:"chat_id"`
		MessageId   *int64  `db:"message_id"`
		ChatAvatar  *string `db:"chat_avatar"`
		MessageText *string `db:"message_text"`
		BlogAbout   bool    `db:"blog_about"`
	}{}

	err := sqlscan.Get(ctx, co, &blogInpData, `
	with blog_message as (
		select m.* from message m where m.chat_id = $1 and m.blog_post = true order by create_date_time desc limit 1
	)
	select 
		cc.id as chat_id,
		m.id as message_id,
		coalesce(cc.avatar_big, cc.avatar) as chat_avatar,
		m.content as message_text,
		coalesce(b.blog_about, false) as blog_about
	from chat_common cc
	left join blog_message m on cc.id = m.chat_id
	left join blog b on cc.id = b.id
	where cc.id = $1
	`, chatId)
	if err != nil {
		return nil, err
	}

	var previousBlogAbout *int64

	var blogAboutVar = blogInpData.BlogAbout
	if blogAboutP != nil {
		blogAboutVar = *blogAboutP

		if blogAboutVar {
			err = sqlscan.Get(ctx, co, &previousBlogAbout, "select id from blog where blog_about limit 1")
			if errors.Is(err, sql.ErrNoRows) {
				// ok
			} else if err != nil {
				return nil, err
			}

			if previousBlogAbout != nil {
				_, err = co.ExecContext(ctx, "update blog set blog_about = false where blog_about = true")
				if err != nil {
					return nil, err
				}
			}
		}
	}

	imageUrl := getBlogPostImage(ctx, m.lgr, blogInpData.MessageText, blogInpData.ChatAvatar, blogInpData.ChatId, blogInpData.MessageId)

	_, errInner := co.ExecContext(ctx, `
				with blog_message as (
					select m.* from message m where m.chat_id = $1 and m.blog_post = true limit 1
				),
				input_data as (
					select 
						 c.id as chat_id
						,m.owner_id
						,m.id as message_id
						,c.title
						,m.content as post
						,left(strip_tags(m.content), $2) as post_preview
						,m.file_item_uuid
						,cast ($3 as timestamp) as create_date_time
						,cast ($4 as text) as image_url
					from chat_common c 
					left join blog_message m on c.id = m.chat_id
					where c.id = $1
				)
				insert into blog(
					id, 
					blog_about,
					owner_id,
					message_id,
					title, 
					image_url,
					post, 
					preview,
					file_item_uuid,
					create_date_time,
					update_date_time
				)
				select 
				     idt.chat_id
					,cast($5 as boolean)
					,idt.owner_id
					,idt.message_id
					,idt.title
					,idt.image_url
					,idt.post
					,idt.post_preview
					,idt.file_item_uuid
					,idt.create_date_time
					,null
				from input_data idt
				on conflict(id) do update set
					  blog_about = excluded.blog_about
					, owner_id = excluded.owner_id
					, message_id = excluded.message_id
					, title = excluded.title
					, image_url = excluded.image_url
					, post = excluded.post
					, preview = excluded.preview
					, file_item_uuid = excluded.file_item_uuid
					, update_date_time = cast ($3 as timestamp)
			`, chatId, m.cfg.Cqrs.Projections.BlogView.MaxTextPreviewSize, createdTime, imageUrl, blogAboutVar)
	if errInner != nil {
		return nil, errInner
	}
	return previousBlogAbout, nil
}

func (m *CommonProjection) removeBlog(ctx context.Context, tx *db.Tx, chatId int64) error {
	_, errInner := tx.ExecContext(ctx, `
				delete from blog where id = $1
			`, chatId)
	if errInner != nil {
		return errInner
	}
	return nil
}

func CanMakeMessageBlogPost(isChatAdmin, tetATet, messageIsBlogPost, chatIsBlog, bloggingIsAllowed bool) bool {
	return bloggingIsAllowed && CanEditChat(isChatAdmin, tetATet) && !messageIsBlogPost && isBlogInternal(chatIsBlog)
}

func isBlogInternal(chatIsBlog bool) bool {
	return chatIsBlog
}

func (m *CommonProjection) OnMessageBlogPostMade(ctx context.Context, event *MessageBlogPostMade) error {
	errOuter := db.Transact(ctx, m.db, func(tx *db.Tx) error {
		chatExists, errInner := m.checkChatExists(ctx, tx, event.ChatId)
		if errInner != nil {
			return errInner
		}
		if !chatExists {
			m.lgr.InfoContext(ctx, "Skipping MessageBlogPostMade because there is no chat", logger.AttributeChatId, event.ChatId)
			return nil
		}

		messageExists, errInner := m.checkMessageExists(ctx, tx, event.ChatId, event.MessageId)
		if errInner != nil {
			return errInner
		}
		if !messageExists {
			m.lgr.InfoContext(ctx, "Skipping MessageBlogPostMade because there is no message", logger.AttributeChatId, event.ChatId, logger.AttributeMessageId, event.MessageId)
			return nil
		}

		// unset previous
		_, errInner = tx.ExecContext(ctx, "update message set blog_post = false where chat_id = $1 and id in (select id from message where chat_id = $1 and blog_post = true)", event.ChatId)
		if errInner != nil {
			return errInner
		}

		_, errInner = tx.ExecContext(ctx, "update message set blog_post = $3 where chat_id = $1 and id = $2", event.ChatId, event.MessageId, event.BlogPost)
		if errInner != nil {
			return errInner
		}

		_, errInner = m.refreshBlog(ctx, tx, event.ChatId, event.AdditionalData.CreatedAt, nil)
		if errInner != nil {
			return errInner
		}
		return nil
	})

	return errOuter
}

func (m *CommonProjection) isChatBlog(ctx context.Context, co db.CommonOperations, chatId int64) (bool, error) {
	var blog bool
	err := sqlscan.Get(ctx, co, &blog, "select exists(select * from blog where id = $1)", chatId)
	if err != nil {
		return false, err
	}
	return blog, nil
}

func (m *CommonProjection) isMessageBlogPost(ctx context.Context, co db.CommonOperations, chatId, messageId int64) (bool, error) {
	var blog bool
	err := sqlscan.Get(ctx, co, &blog, "select exists(select * from message where chat_id = $1 and id = $2 and blog_post = true order by id desc limit 1)", chatId, messageId)
	if err != nil {
		return false, err
	}
	return blog, nil
}

func (m *CommonProjection) GetCurrentBlogPostMessage(ctx context.Context, co db.CommonOperations, chatId int64) (*int64, error) {
	var id int64
	err := sqlscan.Get(ctx, co, &id, "select id from message where chat_id = $1 and blog_post = true order by id desc limit 1", chatId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return nil, nil
		}
		return nil, err
	}
	return &id, nil
}

type blogAbout struct {
	Id    int64  `db:"id"`
	Title string `db:"title"`
}

func (m *CommonProjection) gebBlogAbout(ctx context.Context, co db.CommonOperations) (*blogAbout, error) {
	b := blogAbout{}

	errInner := sqlscan.Get(ctx, co, &b, `
		SELECT 
			ch.id,
			ch.title
		FROM blog ch 
		WHERE 
			ch.blog_about IS TRUE
		ORDER BY id LIMIT 1
	`)
	if errors.Is(errInner, sql.ErrNoRows) {
		return nil, nil
	} else if errInner != nil {
		return nil, errInner
	}
	return &b, nil
}

func (m *EnrichingProjection) GetBlogsEnriched(ctx context.Context, size int32, offset int64, orderBy BlogOrderBy, reverseOrder bool, searchString string) (*dto.BlogPostsDTO, error) {
	searchString = sanitizer.TrimAmdSanitize(m.policy, searchString)

	blogs, count, b, err := m.cp.GetBlogs(ctx, size, offset, orderBy, reverseOrder, searchString)
	if err != nil {
		m.lgr.ErrorContext(ctx, "Error getting blogs", logger.AttributeError, err)
		return nil, err
	}

	userIds := getUserIdsFromBlogs(blogs)
	users, err := m.aaaRestClient.GetUsers(ctx, userIds)
	if err != nil {
		m.lgr.WarnContext(ctx, "unable to get users")
	}
	blogsEnriched := enrichBlogs(ctx, m.lgr, m.cfg, blogs, utils.ToMap(users))

	pagesCount := count / int64(size)
	if count%int64(size) > 0 {
		pagesCount++
	}

	bh := dto.BlogHeader{}
	if b != nil {
		bh.AboutPostId = &b.Id
		bh.AboutPostTitle = &b.Title
	}

	return &dto.BlogPostsDTO{
		Header:     bh,
		Items:      blogsEnriched,
		Count:      count,
		PagesCount: pagesCount,
	}, nil
}

func (m *EnrichingProjection) GetBlogsEnrichedForSeo(ctx context.Context, size int32, offset int64) (*dto.SeoBlogPosts, error) {
	type dbDto struct {
		BlogId     int64     `db:"blog_id"`
		UpdateDate time.Time `db:"update_date"`
	}
	lst := []dbDto{}
	err := sqlscan.Select(ctx, m.cp.db, &lst, `
			select
				id as blog_id, 
				coalesce(update_date_time, create_date_time) as update_date
			from blog
			limit $1 offset $2
		`, size, offset)
	if err != nil {
		return nil, err
	}

	res := make([]dto.BlogSeoItem, 0)
	for _, bl := range lst {
		res = append(res, dto.BlogSeoItem{
			ChatId:       bl.BlogId,
			LastModified: bl.UpdateDate,
		})
	}

	return &dto.SeoBlogPosts{
		Items: res,
	}, err
}

func getUserIdsFromBlogs(chats []BlogListViewDto) []int64 {
	m := map[int64]struct{}{}

	for _, ch := range chats {
		if ch.OwnerId != nil {
			m[*ch.OwnerId] = struct{}{}
		}
	}

	r := []int64{}

	for k, _ := range m {
		r = append(r, k)
	}
	return r
}

func enrichBlogs(
	ctx context.Context,
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	blogs []BlogListViewDto,
	users map[int64]*dto.User,
) []*dto.BlogPostPreviewDto {
	res := make([]*dto.BlogPostPreviewDto, 0, len(blogs))
	for _, ch := range blogs {
		var u *dto.User
		if ch.OwnerId != nil {
			u = users[*ch.OwnerId]
		}

		var postP *string
		if ch.Post != nil && ch.MessageId != nil {
			post := PatchStorageUrlToPublic(ctx, lgr, cfg, *ch.Post, ch.Id, *ch.MessageId)
			postP = &post
		}

		che := dto.BlogPostPreviewDto{
			Id:             ch.Id,
			Title:          ch.Title,
			CreateDateTime: ch.CreateDateTime,
			OwnerId:        ch.OwnerId,
			Owner:          u,
			MessageId:      ch.MessageId,
			Text:           postP,
			Preview:        ch.Preview,
			ImageUrl:       ch.Image,
		}

		res = append(res, &che)
	}
	return res
}

type BlogListViewDto struct {
	Id             int64     `db:"id"`
	OwnerId        *int64    `db:"owner_id"`
	MessageId      *int64    `db:"message_id"`
	Title          string    `db:"title"`
	Post           *string   `db:"post"`
	Preview        *string   `db:"preview"`
	CreateDateTime time.Time `db:"create_date_time"`
	Image          *string   `db:"image_url"`
}

type BlogOrderBy int16

const BlogOrderByCreateDateTime BlogOrderBy = 1
const BlogOrderByUpdateDateTime BlogOrderBy = 2

func (m *CommonProjection) makeBlogSearch(queryArgsInput []any, searchString string) (string, []any) {
	searchClause := ""
	queryArgs := slices.Clone(queryArgsInput)

	if len(searchString) > 0 {
		searchClause = " and ("

		queryArgs = append(queryArgs, searchString)
		searchClause += fmt.Sprintf(`exists( 
			select 1 from (select * from (select unnest(tsvector_to_array(b.fts_all_content))) t(av)) inq 
			where
				   ( inq.av %% plainto_tsquery('russian', $%d)::text )
			    or ( cyrillic_transliterate(inq.av) %% cyrillic_transliterate(plainto_tsquery('russian', $%d)::text) ) 
		) `, len(queryArgs), len(queryArgs))

		searchClause += " ) "
	}

	return searchClause, queryArgs
}

func (m *CommonProjection) GetBlogs(ctx context.Context, size int32, offset int64, orderBy BlogOrderBy, reverseOrder bool, searchString string) ([]BlogListViewDto, int64, *blogAbout, error) {
	itemsQueryArgs := []any{size, offset}
	countQueryArgs := []any{}

	order := "desc"
	if reverseOrder {
		order = "asc"
	}

	itemsSearchClause := ""
	itemsSearchClause, itemsQueryArgs = m.makeBlogSearch(itemsQueryArgs, searchString)

	countSearchClause := ""
	countSearchClause, countQueryArgs = m.makeBlogSearch(countQueryArgs, searchString)

	type postsWithCount struct {
		blogListViewDto []BlogListViewDto
		count           int64
		blogAbout       *blogAbout
	}

	orderByCaluse := ""
	switch orderBy {
	case BlogOrderByCreateDateTime:
		orderByCaluse = "b.create_date_time"
	case BlogOrderByUpdateDateTime:
		orderByCaluse = "b.update_date_time"
	default:
		return nil, 0, nil, fmt.Errorf("Unknown order by: %v", orderBy)
	}

	pwc, errOuter := db.TransactWithResult(ctx, m.db, func(tx *db.Tx) (*postsWithCount, error) {
		ma := []BlogListViewDto{}

		errInner := sqlscan.Select(ctx, tx, &ma, fmt.Sprintf(`
			select 
				b.id,
				b.owner_id,
				b.message_id,
				b.title,
				b.post,
				b.image_url,
				b.preview,
				b.create_date_time
			from blog b
			where true %s
			order by %s %s
			limit $1 offset $2
		`, itemsSearchClause, orderByCaluse, order), itemsQueryArgs...)
		if errInner != nil {
			return nil, errInner
		}

		var count int64
		errInner = sqlscan.Get(ctx, tx, &count, fmt.Sprintf("select count(*) from blog b where true %s", countSearchClause), countQueryArgs...)
		if errInner != nil {
			return nil, errInner
		}

		b, errInner := m.gebBlogAbout(ctx, tx)
		if errInner != nil {
			return nil, errInner
		}

		return &postsWithCount{
			blogListViewDto: ma,
			count:           count,
			blogAbout:       b,
		}, nil
	})

	if errOuter != nil {
		return nil, 0, nil, errOuter
	}

	return pwc.blogListViewDto, pwc.count, pwc.blogAbout, nil
}

func (m *EnrichingProjection) GetBlogEnriched(ctx context.Context, blogId int64) (*dto.WrappedBlogPostResponse, error) {
	type dbDto struct {
		blog         *BlogPostViewDto
		chatBasic    *dto.ChatBasic
		reactionsMap map[int64][]dto.ReactionDto
		reactions    []dto.ReactionDto
		blogAbout    *blogAbout
	}
	dd, errOuter := db.TransactWithResult(ctx, m.cp.db, func(tx *db.Tx) (*dbDto, error) {
		blog, chatBasic, b, errInn := m.cp.GetBlog(ctx, tx, blogId)
		if errInn != nil {
			m.lgr.ErrorContext(ctx, "Error getting blog", logger.AttributeError, errInn)
			return nil, errInn
		}

		if blog == nil {
			return nil, nil
		}

		var reactions []dto.ReactionDto
		var reactionsMap map[int64][]dto.ReactionDto

		if blog.MessageId != nil {
			reactionsMap, errInn = m.getReactions(ctx, tx, blogId, []int64{*blog.MessageId})
			if errInn != nil {
				return nil, fmt.Errorf("Got error during enriching messages with reactions: %v", errInn)
			}

			reactions = reactionsMap[*blog.MessageId]
		}

		return &dbDto{
			blog:         blog,
			chatBasic:    chatBasic,
			reactions:    reactions,
			reactionsMap: reactionsMap,
			blogAbout:    b,
		}, nil
	})

	if errOuter != nil {
		return nil, errOuter
	}
	if dd == nil {
		return nil, nil
	}

	var usersSet = map[int64]bool{}
	var chatsPreSet = map[int64]bool{}
	if dd.blog != nil && dd.blog.MessageId != nil && dd.blog.OwnerId != nil {
		err := populateSets(*dd.blog.MessageId, *dd.blog.OwnerId, nil, nil, usersSet, chatsPreSet, dd.blog.Id, dd.reactionsMap)
		if err != nil {
			return nil, err
		}
	}

	users, err := m.aaaRestClient.GetUsers(ctx, utils.SetMapIdBoolToSlice(usersSet))
	if err != nil {
		m.lgr.WarnContext(ctx, "unable to get users")
	}

	usersMap := utils.ToMap(users)

	blogEnriched := enrichBlog(ctx, m.lgr, m.cfg, dd.blog, usersMap, dd.reactions)

	bh := dto.BlogHeader{}
	if dd.blogAbout != nil {
		bh.AboutPostId = &dd.blogAbout.Id
		bh.AboutPostTitle = &dd.blogAbout.Title
	}

	return &dto.WrappedBlogPostResponse{
		Header:          bh,
		Post:            *blogEnriched,
		CanWriteMessage: dd.chatBasic.RegularParticipantCanWriteMessage,
	}, nil
}

func enrichBlog(
	ctx context.Context,
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	blog *BlogPostViewDto,
	users map[int64]*dto.User,
	reactions []dto.ReactionDto,
) *dto.BlogPostResponse {
	if blog == nil {
		return nil
	}

	var u *dto.User
	ownerIdP := blog.OwnerId
	if ownerIdP != nil {
		u = users[*ownerIdP]
	}

	var postP *string

	if blog.Post != nil && blog.MessageId != nil {
		post := PatchStorageUrlToPublic(ctx, lgr, cfg, *blog.Post, blog.Id, *blog.MessageId)
		postP = &post
	}

	res := dto.BlogPostResponse{
		ChatId:         blog.Id,
		Title:          blog.Title,
		OwnerId:        blog.OwnerId,
		Owner:          u,
		MessageId:      blog.MessageId,
		Text:           postP,
		CreateDateTime: blog.CreateDateTime,
		Reactions:      make([]dto.Reaction, 0),
		Preview:        blog.Preview,
		FileItemUuid:   blog.FileItemUuid,
	}

	if blog.MessageId != nil {
		res.Reactions = makeReactions(users, reactions)
	}

	return &res
}

type BlogPostViewDto struct {
	Id             int64     `db:"id"`
	OwnerId        *int64    `db:"owner_id"`
	MessageId      *int64    `db:"message_id"`
	Title          string    `db:"title"`
	Post           *string   `db:"post"`
	Preview        *string   `db:"preview"`
	CreateDateTime time.Time `db:"create_date_time"`
	FileItemUuid   *string   `db:"file_item_uuid"`
}

func (m *CommonProjection) GetBlog(ctx context.Context, co db.CommonOperations, blogId int64) (*BlogPostViewDto, *dto.ChatBasic, *blogAbout, error) {
	var bld BlogPostViewDto
	err := sqlscan.Get(ctx, co, &bld, `
		select 
		    b.id,
			b.owner_id,
			b.message_id,
		    b.title,
		    b.post,
		    b.preview,
		    b.file_item_uuid,
		    b.create_date_time
		from blog b
		where b.id = $1
		order by b.create_date_time desc 
	`, blogId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return nil, nil, nil, nil
		}
		return nil, nil, nil, err
	}

	cb, err := m.GetChatBasic(ctx, co, blogId)
	if err != nil {
		return nil, nil, nil, err
	}

	if cb == nil {
		return nil, nil, nil, nil
	}

	b, errInner := m.gebBlogAbout(ctx, co)
	if errInner != nil {
		return nil, nil, nil, errInner
	}

	return &bld, cb, b, nil
}

func (m *CommonProjection) getBlogPostMessageId(ctx context.Context, co db.CommonOperations, blogId int64) (*int64, error) {
	var messageId *int64
	err := sqlscan.Get(ctx, co, &messageId, "select message_id from blog where id = $1", blogId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return nil, nil
		}
		return nil, err
	}
	return messageId, nil
}

func (m *CommonProjection) getComments(ctx context.Context, co db.CommonOperations, blogId, postMessageId int64, size int32, offset int64, reverseOrder bool) ([]dto.CommentViewDto, error) {
	type commentViewDto struct {
		Id             int64        `db:"id"`
		OwnerId        int64        `db:"owner_id"`
		Content        string       `db:"content"`
		Embed          pgtype.JSONB `db:"embed"`
		FileItemUuid   *string      `db:"file_item_uuid"`
		CreateDateTime time.Time    `db:"create_date_time"`
		UpdateDateTime *time.Time   `db:"update_date_time"` // for sake compatibility
	}

	mar := []dto.CommentViewDto{}
	ma := []commentViewDto{}

	order := "asc"
	if reverseOrder {
		order = "desc"
	}

	err := sqlscan.Select(ctx, co, &ma, fmt.Sprintf(`
		select 
		    id, 
		    owner_id,
		    content,
		    embed,
		    file_item_uuid,
		    create_date_time,
		    update_date_time
		from message 
		where chat_id = $1 and id > $2
		order by id %s
		limit $3 offset $4
	`, order), blogId, postMessageId, size, offset)

	if err != nil {
		return mar, err
	}

	for i, mm := range ma {
		mc := dto.CommentViewDto{
			Id:             mm.Id,
			OwnerId:        mm.OwnerId,
			Content:        mm.Content,
			CreateDateTime: mm.CreateDateTime,
			UpdateDateTime: mm.UpdateDateTime,
			FileItemUuid:   mm.FileItemUuid,
		}

		embeddable, err := makeEmbedddable(mm.Embed)
		if err != nil {
			return mar, fmt.Errorf("error during mapping on index %d: %w", i, err)
		}
		mc.Embed = embeddable

		mar = append(mar, mc)
	}

	return mar, nil
}

func (m *EnrichingProjection) GetCommentsEnriched(ctx context.Context, blogId int64, size int32, offset int64, reverseOrder bool) (*dto.CommentsWrapper, error) {
	type commentsWithData struct {
		comments                []dto.CommentViewDto
		chatsBehalfUserByChatId map[int64]*dto.BasicChatDtoExtended
		usersSet                map[int64]bool
		postMessageId           int64
		count                   int64
		reactions               map[int64][]dto.ReactionDto
	}

	cwd, errOuter := db.TransactWithResult(ctx, m.cp.db, func(tx *db.Tx) (*commentsWithData, error) {
		postMessageId, errInn := m.cp.getBlogPostMessageId(ctx, tx, blogId)
		if postMessageId == nil {
			return &commentsWithData{
				comments:                make([]dto.CommentViewDto, 0),
				chatsBehalfUserByChatId: make(map[int64]*dto.BasicChatDtoExtended),
				usersSet:                make(map[int64]bool),
				postMessageId:           dto.NoId,
			}, nil
		}

		if errInn != nil {
			m.lgr.ErrorContext(ctx, "Error getting blog post message id", logger.AttributeError, errInn)
			return nil, errInn
		}

		comments, errInn := m.cp.getComments(ctx, tx, blogId, *postMessageId, size, offset, reverseOrder)
		if errInn != nil {
			m.lgr.ErrorContext(ctx, "Error getting blog comments", logger.AttributeError, errInn)
			return nil, errInn
		}

		messageIds := make([]int64, 0)
		for _, message := range comments {
			messageIds = append(messageIds, message.Id)
		}

		reactions, err := m.getReactions(ctx, tx, blogId, messageIds)
		if err != nil {
			return nil, fmt.Errorf("Got error during enriching messages with reactions: %v", err)
		}

		var usersSet = map[int64]bool{}
		var chatsPreSet = map[int64]bool{}
		for _, co := range comments {
			errInn = populateSets(co.Id, co.OwnerId, nil, co.Embed, usersSet, chatsPreSet, blogId, reactions)
			if errInn != nil {
				return nil, errInn
			}
		}

		behalfUserId := int64(dto.NonExistentUser)
		chatsByUserByChatId, errInn := m.cp.GetChatsBasicExtended(ctx, tx, utils.SetMapIdBoolToSlice(chatsPreSet), []int64{behalfUserId})
		if errInn != nil {
			m.lgr.ErrorContext(ctx, "Error getting chat basic", logger.AttributeError, errInn)
			return nil, errInn
		}

		chatsBehalfUserByChatId := chatsByUserByChatId[behalfUserId]

		var count int64
		errInn = sqlscan.Get(ctx, tx, &count, "SELECT count(*) FROM message m WHERE m.chat_id = $1 AND m.id > $2", blogId, postMessageId)
		if errInn != nil {
			m.lgr.ErrorContext(ctx, "Error getting comment count", logger.AttributeError, errInn)
			return nil, errInn
		}

		return &commentsWithData{
			comments:                comments,
			chatsBehalfUserByChatId: chatsBehalfUserByChatId,
			usersSet:                usersSet,
			postMessageId:           *postMessageId,
			count:                   count,
			reactions:               reactions,
		}, nil
	})
	if errOuter != nil {
		return nil, errOuter
	}

	res := make([]dto.CommentViewEnrichedDto, 0, len(cwd.comments))

	users, err := m.aaaRestClient.GetUsers(ctx, utils.SetMapIdBoolToSlice(cwd.usersSet))
	if err != nil {
		m.lgr.WarnContext(ctx, "unable to get users")
	}
	usersMap := utils.ToMap(users)

	for _, co := range cwd.comments {
		me := dto.CommentViewEnrichedDto{
			Id:             co.Id,
			OwnerId:        co.OwnerId,
			Content:        PatchStorageUrlToPublic(ctx, m.lgr, m.cfg, co.Content, blogId, co.Id),
			FileItemUuid:   co.FileItemUuid,
			CreateDateTime: co.CreateDateTime,
			UpdateDateTime: co.UpdateDateTime,
			Owner:          usersMap[co.OwnerId],
		}

		embed, err := makeEmbed(co.Embed, usersMap, cwd.chatsBehalfUserByChatId)
		if err != nil {
			return nil, err
		}

		if embed != nil {
			embed.Text = PatchStorageUrlToPublic(ctx, m.lgr, m.cfg, embed.Text, blogId, co.Id)
			me.EmbedMessage = embed
		}

		rl := cwd.reactions[co.Id]
		me.Reactions = makeReactions(usersMap, rl)

		res = append(res, me)
	}

	count := cwd.count

	pagesCount := count / int64(size)
	if count%int64(size) > 0 {
		pagesCount++
	}

	return &dto.CommentsWrapper{
		Items:      res,
		Count:      count,
		PagesCount: pagesCount,
	}, nil
}

func PatchStorageUrlToPublic(
	ctx context.Context,
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	text string,
	overrideChatId,
	overrideMessageId int64,
) string {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
	if err != nil {
		lgr.WarnContext(ctx, "Unable to read html", logger.AttributeError, err)
		return ""
	}

	wlArr := []string{"", cfg.FrontendUrl} // if our own server (storage)

	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		maybeImage := s.First()
		if maybeImage != nil {
			original, originalExists := maybeImage.Attr("data-original")
			if originalExists { // we have 2 tags - preview (small, tag attr) and original (data-original attr)
				if utils.ContainsUrl(ctx, lgr, wlArr, original) { // original
					newurl, err := makeUrlPublic(original, "", overrideChatId, overrideMessageId)
					if err != nil {
						lgr.WarnContext(ctx, "Unable to change url", logger.AttributeError, err)
						return
					}
					maybeImage.SetAttr("data-original", newurl)
				}

				src, srcExists := maybeImage.Attr("src") // preview
				if srcExists && utils.ContainsUrl(ctx, lgr, wlArr, src) {
					newurl, err := makeUrlPublic(src, utils.UrlStorageEmbedPreview, overrideChatId, overrideMessageId)
					if err != nil {
						lgr.WarnContext(ctx, "Unable to change url", logger.AttributeError, err)
						return
					}
					maybeImage.SetAttr("src", newurl)
				}
			}
		}
	})

	ret, err := doc.Find("html").Find("body").Html()
	if err != nil {
		lgr.WarnContext(ctx, "Unable to write html", logger.AttributeError, err)
		return ""
	}
	return ret
}

func getBlogPostImage(ctx context.Context, lgr *logger.LoggerWrapper, messageText, chatAvatar *string, chatId int64, messageId *int64) *string {
	if !(messageText == nil || messageId == nil) {
		mbImage := tryGetFirstImage(ctx, lgr, *messageText)
		if mbImage != nil {
			fileParam, err := getFileParam(*mbImage)
			if err != nil {
				lgr.WarnContext(ctx, "Unable to get file key", logger.AttributeError, err)
				return nil
			}
			if len(fileParam) > 0 {
				dumbUrl := url.URL{}
				query := dumbUrl.Query()
				query.Set(utils.FileParam, utils.SetImagePreviewExtension(fileParam))
				dumbUrl.RawQuery = query.Encode()

				publicPreviewUrl, err := makeUrlPublic(dumbUrl.String(), utils.UrlStorageEmbedPreview, chatId, *messageId)
				if err != nil {
					lgr.WarnContext(ctx, "Unable to to change url", logger.AttributeError, err)
					return nil
				}
				return &publicPreviewUrl
			}
		}
	}

	return chatAvatar
}

func tryGetFirstImage(ctx context.Context, lgr *logger.LoggerWrapper, text string) *string {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
	if err != nil {
		lgr.WarnContext(ctx, "Unable to get image", logger.AttributeError, err)
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

func getFileParam(src string) (string, error) {
	parsed, err := url.Parse(src)
	if err != nil {
		return "", err
	}
	fileParam := parsed.Query().Get(utils.FileParam)
	return fileParam, nil
}

const OverrideMessageId = "overrideMessageId"
const OverrideChatId = "overrideChatId"

// see also storage/services/files.go :: makeUrlPublic
func makeUrlPublic(src string, additionalSegment string, overrideChatId, overrideMessageId int64) (string, error) {
	if strings.HasPrefix(src, "/images/covers/") { // don't touch built-in default urls (used for video-by-link, audio)
		return src, nil
	}

	// we add time in order not to cache the video itself
	parsed, err := url.Parse(src)
	if err != nil {
		return "", err
	}

	parsed.Path = utils.UrlStoragePublicGetFile + additionalSegment

	query := parsed.Query()

	query.Set(OverrideMessageId, utils.ToString(overrideMessageId))
	query.Set(OverrideChatId, utils.ToString(overrideChatId))

	parsed.RawQuery = query.Encode()

	newurl := parsed.String()
	return newurl, nil
}
