package com.github.nkonev.blog.services;

import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.converter.PostConverter;
import com.github.nkonev.blog.dto.*;
import com.github.nkonev.blog.entity.elasticsearch.IndexPost;
import com.github.nkonev.blog.entity.jdbc.Post;
import com.github.nkonev.blog.entity.jdbc.UserAccount;
import com.github.nkonev.blog.exception.BadRequestException;
import com.github.nkonev.blog.exception.DataNotFoundException;
import com.github.nkonev.blog.repository.elasticsearch.IndexPostRepository;
import com.github.nkonev.blog.repository.jdbc.CommentRepository;
import com.github.nkonev.blog.repository.jdbc.PostRepository;
import com.github.nkonev.blog.repository.jdbc.UserAccountRepository;
import com.github.nkonev.blog.security.BlogSecurityService;
import com.github.nkonev.blog.security.permissions.PostPermissions;
import com.github.nkonev.blog.utils.PageUtils;
import org.elasticsearch.index.query.AbstractQueryBuilder;
import org.elasticsearch.index.query.QueryBuilders;
import org.elasticsearch.search.fetch.subphase.highlight.HighlightBuilder;
import org.elasticsearch.search.sort.FieldSortBuilder;
import org.elasticsearch.search.sort.SortOrder;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.domain.PageRequest;
import org.springframework.data.domain.Pageable;
import org.springframework.data.elasticsearch.core.ElasticsearchRestTemplate;
import org.springframework.data.elasticsearch.core.SearchHit;
import org.springframework.data.elasticsearch.core.SearchHits;
import org.springframework.data.elasticsearch.core.mapping.IndexCoordinates;
import org.springframework.data.elasticsearch.core.query.FetchSourceFilter;
import org.springframework.data.elasticsearch.core.query.NativeSearchQuery;
import org.springframework.data.elasticsearch.core.query.NativeSearchQueryBuilder;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.jdbc.core.ResultSetExtractor;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;
import org.springframework.lang.Nullable;
import org.springframework.security.access.hierarchicalroles.RoleHierarchy;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.stereotype.Service;
import org.springframework.util.Assert;
import org.springframework.util.StringUtils;
import javax.validation.constraints.NotNull;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.time.LocalDateTime;
import java.util.*;
import java.util.concurrent.TimeUnit;
import java.util.stream.Collectors;
import static com.github.nkonev.blog.entity.elasticsearch.IndexPost.*;
import static com.github.nkonev.blog.utils.TimeUtil.getNowUTC;
import static org.elasticsearch.index.query.QueryBuilders.*;

@Service
public class PostService {

    private static final ResultSetExtractor<PostDTO> POST_DTO_RESULT_SET_EXTRACTOR = rs -> {
        if (!rs.next()) {
            return null;
        }
        String title = rs.getString("title");
        long id = rs.getLong("id");
        PostDTO postDTO = new PostDTO();
        postDTO.setId(id);
        postDTO.setTitle(title);
        return postDTO;
    };

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private PostConverter postConverter;

    @Autowired
    private PostRepository postRepository;

    @Autowired
    private IndexPostRepository indexPostRepository;

    @Autowired
    private ElasticsearchRestTemplate elasticsearchTemplate;

    @Autowired
    private NamedParameterJdbcTemplate jdbcTemplate;

    @Autowired
    private CommentRepository commentRepository;

    @Autowired
    private WebSocketService webSocketService;

    @Autowired
    private SeoCacheListenerProxy seoCacheListenerProxy;

    @Autowired
    private SeoCacheService seoCacheService;

    @Autowired
    private RoleHierarchy roleHierarchy;

    @Autowired
    private BlogSecurityService blogSecurityService;

    @Value("${custom.elasticsearch.get.all.index.ids.chunk.size:20}")
    private int chunkSize;

    @Value("${custom.elasticsearch.search.field.text.slop:24}")
    private int searchFieldTextSlop;

    @Value("${custom.elasticsearch.search.field.title.slop:8}")
    private int searchFieldTitleSlop;

    @Value("${custom.elasticsearch.highlight.field.text.num.of.fragments:5}")
    private int highlightFieldTextNumOfFragments;

    @Value("${custom.elasticsearch.highlight.field.title.num.of.fragments:1}")
    private int highlightFieldTitleNumOfFragments;

    @Value("${custom.elasticsearch.highlight.field.text.num.of.fragments:150}")
    private int highlightFieldTextFragmentSize;

    @Value("${custom.elasticsearch.highlight.field.title.num.of.fragments:150}")
    private int highlightFieldTitleFragmentSize;

    @Value(Constants.ELASTICSEARCH_REFRESH_ON_START_KEY_TIMEOUT)
    private int timeout;

    @Value(Constants.ELASTICSEARCH_REFRESH_ON_START_KEY_TIMEUNIT)
    private TimeUnit timeUnit;

    @Value("${custom.elasticsearch.refresh.batch.size:100}")
    private int refreshBatchSize;

    @Autowired
    private RedisTemplate<String, String> redisTemplate;

    private static final Logger LOGGER = LoggerFactory.getLogger(PostService.class);

    static class Tuple2<T1, T2> {
        private T1 t1;
        private T2 t2;

        public Tuple2(T1 t1, T2 t2) {
            this.t1 = t1;
            this.t2 = t2;
        }

        public T1 getT1() {
            return t1;
        }

        public T2 getT2() {
            return t2;
        }
    }

    private static class PostRowMapper implements RowMapper<PostDTO> {

        private boolean setTitle, setPost;

        public String getBaseSql() {
            return "select " +
                    "p.id, " +
                    "p.title_img, " +
                    "p.draft, " +
                    (setTitle ? "p.title, " : "") +
                    (setPost ? "p.text, "  : "") +
                    "p.create_date_time," +
                    "p.edit_date_time," +
                    "u.id as owner_id," +
                    "u.username as owner_login," +
                    "u.avatar as owner_avatar, " +
                    "(select count(*) from posts.comment c where c.post_id = p.id) as comment_count " +
                    "  from posts.post p " +
                    "    join auth.users u on p.owner_id = u.id ";
        }

        public PostRowMapper(boolean setTitle, boolean setPost) {
            this.setTitle = setTitle;
            this.setPost = setPost;
        }

        @Override
        public PostDTO mapRow(ResultSet resultSet, int i) throws SQLException {
            return new PostDTO(
                    resultSet.getLong("id"),
                    setTitle ? resultSet.getString("title") : null,
                    setPost ? resultSet.getString("text") : null,
                    resultSet.getString("title_img"),
                    resultSet.getObject("create_date_time", LocalDateTime.class),
                    resultSet.getObject("edit_date_time", LocalDateTime.class),
                    resultSet.getInt("comment_count"),
                    new OwnerDTO(
                            resultSet.getLong("owner_id"),
                            resultSet.getString("owner_login"),
                            resultSet.getString("owner_avatar")
                    ),
                    resultSet.getBoolean("draft")
            );
        }
    }

    private final PostRowMapper rowMapperWithoutTextTitle = new PostRowMapper(false, false);
    private final PostRowMapper rowMapper = new PostRowMapper(true, true);

    public PostDTOWithAuthorization addPost(UserAccountDetailsDTO userAccount, @NotNull PostDTO postDTO){
        Assert.notNull(userAccount, "UserAccountDetailsDTO can't be null");
        if (postDTO.getId() != 0) {
            throw new BadRequestException("id cannot be set");
        }
        Post fromWeb = postConverter.convertToPost(postDTO, null);
        fromWeb.setCreateDateTime(getNowUTC());
        UserAccount ua = userAccountRepository.findById(userAccount.getId()).orElseThrow(()->new IllegalArgumentException("User account not found")); // Hibernate caches it
        fromWeb.setOwnerId(ua.getId());
        Post saved = postRepository.save(fromWeb);
        indexPostRepository.save(postConverter.toElasticsearchPost(saved));

        webSocketService.sendInsertPostEvent(postDTO);
        seoCacheListenerProxy.rewriteCachedPage(saved.getId());
        seoCacheListenerProxy.rewriteCachedIndex();

        return postConverter.convertToDto(saved, userAccount);
    }

    public PostDTOWithAuthorization updatePost(UserAccountDetailsDTO userAccount, @NotNull PostDTO postDTO) {
        Assert.notNull(userAccount, "UserAccountDetailsDTO can't be null");
        Post found = postRepository.findById(postDTO.getId()).orElseThrow(()->new IllegalArgumentException("Post with id " + postDTO.getId() + " not found"));
        Post updatedEntity = postConverter.convertToPost(postDTO, found);
        updatedEntity.setEditDateTime(getNowUTC());
        Post saved = postRepository.save(updatedEntity);
        indexPostRepository.save(postConverter.toElasticsearchPost(saved));

        webSocketService.sendUpdatePostEvent(postDTO);
        seoCacheListenerProxy.rewriteCachedPage(saved.getId());
        seoCacheListenerProxy.rewriteCachedIndex();

        return postConverter.convertToDto(saved, userAccount);
    }

    private PostDTO convertToPostDTOWithCleanTags(Post post) {
        PostDTO postDTO = postConverter.convertToPostDTO(post);
        IndexPost byId = indexPostRepository
                .findById(post.getId())
                .orElseThrow(()->new DataNotFoundException("post "+post.getId()+" not found in fulltext store"));
        postDTO.setText(byId.getText());
        return postDTO;
    }

    public List<PostDTO> getMyPosts(int page, int size, @Nullable UserAccountDetailsDTO userAccount) {
        page = PageUtils.fixPage(page);
        size = PageUtils.fixSize(size);
        PageRequest springDataPage = PageRequest.of(PageUtils.fixPage(page), PageUtils.fixSize(size));

        return postRepository
                .findMyPosts(springDataPage.getPageSize(), springDataPage.getOffset(), Optional.ofNullable(userAccount).map(u-> userAccount.getId()).orElse(null))
                .stream()
                .map(this::convertToPostDTOWithCleanTags)
                .collect(Collectors.toList());
    }

    private AbstractQueryBuilder noDraftFilterElasticsearch(UserAccountDetailsDTO userAccountDetailsDTO){
        if (userAccountDetailsDTO==null) {
            return termQuery(FIELD_DRAFT, false);
        } else {
            if (roleHierarchy.getReachableGrantedAuthorities(userAccountDetailsDTO.getAuthorities()).contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name()))){
                return matchAllQuery();
            } else {
                return boolQuery()
                        .should(termQuery(FIELD_DRAFT, false))
                        .should(termQuery(FIELD_OWNER_ID, userAccountDetailsDTO.getId()));
            }
        }
    }

    private Tuple2<String, Map<String, Object>> noDraftFilterJdbc(UserAccountDetailsDTO userAccountDetailsDTO){
        Map<String, Object> countParams = new HashMap<>();
        if (userAccountDetailsDTO==null) {
            countParams.put("currentUserId", null);
            countParams.put("isAdmin", false);
        } else {
            countParams.put("currentUserId", userAccountDetailsDTO.getId());
            countParams.put("isAdmin", roleHierarchy.getReachableGrantedAuthorities(userAccountDetailsDTO.getAuthorities()).contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name())));
        }
        return new Tuple2<>("(p.draft = FALSE OR ((:currentUserId\\:\\:bigint) = p.owner_id) OR :isAdmin = TRUE)", countParams);
    }

    private PostDTOExtended convertToDtoExtended(PostDTO saved, UserAccountDetailsDTO userAccount, Tuple2<String, Map<String, Object>> noDraftFilterJdbc) {
        Assert.notNull(saved, "Post can't be null");

        var params = new HashMap<String, Object>();
        params.put("postId", saved.getId());
        params.putAll(noDraftFilterJdbc.getT2());

        String sqlLeft = "SELECT p.id, p.title FROM posts.post p WHERE p.id < :postId AND "+noDraftFilterJdbc.getT1()+" ORDER BY id DESC LIMIT 1";
        String sqlright = "SELECT p.id, p.title FROM posts.post p WHERE p.id > :postId AND "+noDraftFilterJdbc.getT1()+" ORDER BY id ASC LIMIT 1";
        PostDTO left = jdbcTemplate.query(sqlLeft, params, POST_DTO_RESULT_SET_EXTRACTOR);
        PostDTO right = jdbcTemplate.query(sqlright, params, POST_DTO_RESULT_SET_EXTRACTOR);

        return new PostDTOExtended(
                saved.getId(),
                saved.getTitle(),
                (saved.getText()),
                saved.getTitleImg(),
                saved.getOwner(),
                blogSecurityService.hasPostPermission(saved, userAccount, PostPermissions.EDIT),
                blogSecurityService.hasPostPermission(saved, userAccount, PostPermissions.DELETE),
                left != null ? new PostPreview(left.getId(), left.getTitle()) : null,
                right != null ? new PostPreview(right.getId(), right.getTitle()) : null,
                saved.getCreateDateTime(),
                saved.getEditDateTime(),
                saved.isDraft()
        );
    }

    public Optional<PostDTOExtended> findById(long postId, @Nullable UserAccountDetailsDTO userAccountDetailsDTO){
        Tuple2<String, Map<String, Object>> tuple2 = noDraftFilterJdbc(userAccountDetailsDTO);
        var params = new HashMap<String, Object>();
        params.put("postId", postId);
        params.putAll(tuple2.getT2());

        var postsResult = jdbcTemplate.query(
                rowMapper.getBaseSql() + " WHERE p.id = :postId AND " + tuple2.getT1(),
                params,
                rowMapper
        );

        if (postsResult.isEmpty()){
            return Optional.empty();
        } else {
            return Optional.of(convertToDtoExtended(postsResult.get(0), userAccountDetailsDTO, tuple2));
        }
    }

    public Wrapper<PostDTO> getPosts(int page, int size, String searchString, @Nullable UserAccountDetailsDTO currentUser){
        page = PageUtils.fixPage(page);
        size = PageUtils.fixSize(size);
        searchString = StringUtils.trimWhitespace(searchString);

        List<PostDTO> postsResult;

        if (StringUtils.isEmpty(searchString)) {
            PageRequest pageRequest = PageRequest.of(page, size);

            NativeSearchQuery searchQuery = new NativeSearchQueryBuilder()
                    .withSort(new FieldSortBuilder(FIELD_ID).order(SortOrder.DESC))
                    .withQuery(boolQuery()
                            .must(noDraftFilterElasticsearch(currentUser))
                    )
                    .withPageable(pageRequest)
                    .build();
            // https://stackoverflow.com/questions/37049764/how-to-provide-highlighting-with-spring-data-elasticsearch/37163711#37163711
            postsResult = getPostDTOS(searchQuery);
        } else {
            PageRequest pageRequest = PageRequest.of(page, size);

            // need for correct highlight source field e. g. forceSource(true)
            final String fastVectorHighlighter = "fvh";

            NativeSearchQuery searchQuery = new NativeSearchQueryBuilder()
                    .withSort(new FieldSortBuilder(FIELD_ID).order(SortOrder.DESC))
                    .withQuery(boolQuery()
                            .must(
                                    boolQuery()
                                            .should(matchPhrasePrefixQuery(FIELD_TEXT, searchString).slop(searchFieldTextSlop))
                                            .should(matchPhrasePrefixQuery(FIELD_TITLE, searchString).slop(searchFieldTitleSlop))
                                            // https://www.elastic.co/guide/en/elasticsearch/reference/current/multi-fields.html
                                            .should(QueryBuilders.queryStringQuery("*"+org.apache.lucene.queryparser.classic.QueryParser.escape(searchString)+"*").field(FIELD_TEXT_STD).field(FIELD_TITLE_STD).analyzeWildcard(true))
                            )

                            .must(noDraftFilterElasticsearch(currentUser))
                    )
                    .withHighlightFields(
                            new HighlightBuilder.Field(FIELD_TEXT).matchedFields(FIELD_TEXT, FIELD_TEXT_STD).forceSource(true).highlighterType(fastVectorHighlighter).preTags("<b>").postTags("</b>").numOfFragments(highlightFieldTextNumOfFragments).fragmentSize(highlightFieldTextFragmentSize),
                            new HighlightBuilder.Field(FIELD_TITLE).matchedFields(FIELD_TITLE, FIELD_TITLE_STD).forceSource(true).highlighterType(fastVectorHighlighter).preTags("<u>").postTags("</u>").numOfFragments(highlightFieldTitleNumOfFragments).fragmentSize(highlightFieldTitleFragmentSize)
                    )
                    .withPageable(pageRequest)
                    .build();
            // https://stackoverflow.com/questions/37049764/how-to-provide-highlighting-with-spring-data-elasticsearch/37163711#37163711
            postsResult = getPostDTOS(searchQuery);
        }

        NativeSearchQuery countQuery = new NativeSearchQueryBuilder()
                .withQuery(boolQuery().must(noDraftFilterElasticsearch(currentUser)))
                .build();
        long totalCount = elasticsearchTemplate.count(countQuery, IndexCoordinates.of(INDEX));

        return new Wrapper<>(postsResult, totalCount);
    }

    private List<PostDTO> getPostDTOS(NativeSearchQuery searchQuery) {
        List<PostDTO> postsResult;
        SearchHits<IndexPost> fulltextResult = elasticsearchTemplate.search(searchQuery, IndexPost.class, IndexCoordinates.of(INDEX));

        postsResult = new ArrayList<>();
        for (SearchHit<IndexPost> fulltextPost0: fulltextResult.getSearchHits()){

            final IndexPost fulltextPost = fulltextPost0.getContent();
            var params = new HashMap<String, Object>();
            params.put("postId", fulltextPost.getId());
            LOGGER.debug("Will search in postgres by id="+fulltextPost.getId());
            PostDTO postDTO = jdbcTemplate.queryForObject(
                    rowMapperWithoutTextTitle.getBaseSql() + " where p.id = :postId",
                    params,
                    rowMapperWithoutTextTitle
            );
            if (postDTO == null){
                throw new DataNotFoundException("post not found in db");
            }

            final String text = getHighlightOrText(fulltextPost0, FIELD_TEXT, fulltextPost0.getContent().getText());
            postDTO.setText(text);
            final String title = getHighlightOrText(fulltextPost0, FIELD_TITLE, fulltextPost0.getContent().getTitle());
            postDTO.setTitle(title);

            postsResult.add(postDTO);
        }
        return postsResult;
    }

    private String getHighlightOrText(SearchHit<IndexPost> fulltextPost0, String fieldText, String nonHighlightedText) {
        return Optional.ofNullable(fulltextPost0.getHighlightFields().get(fieldText)).map(strings -> String.join("... ", strings)).orElse(nonHighlightedText);
    }

    public Wrapper<PostDTO> findByOwnerId(Pageable springDataPage, Long userId, UserAccountDetailsDTO userAccountDetailsDTO) {
        int limit = springDataPage.getPageSize();
        long offset = springDataPage.getOffset();

        Tuple2<String, Map<String, Object>> tuple2 = noDraftFilterJdbc(userAccountDetailsDTO);
        var params = new HashMap<String, Object>();
        params.put("offset", offset);
        params.put("limit", limit);
        params.put("userId", userId);
        params.putAll(tuple2.getT2());

        var postsResult = jdbcTemplate.query(
                rowMapper.getBaseSql() +
                        " WHERE u.id = :userId AND " + tuple2.getT1() +
                        "  ORDER BY p.id DESC " +
                        "limit :limit offset :offset\n",
                params,
                rowMapper
        );

        List<PostDTO> list = postsResult.stream()
                .peek(PostConverter::cleanTags)
                .collect(Collectors.toList());
        var countParams = new HashMap<String, Object>();
        countParams.put("userId", userId);
        countParams.putAll(tuple2.getT2());
        long count = jdbcTemplate.queryForObject("SELECT COUNT(*) FROM posts.post p WHERE p.owner_id = :userId AND "+tuple2.getT1(), countParams, long.class);
        return new Wrapper<>(list, count);
    }

    public void deletePost(UserAccountDetailsDTO userAccount, long postId) {
        Assert.notNull(userAccount, "UserAccountDetailsDTO can't be null");
        commentRepository.deleteByPostId(postId);
        postRepository.deleteById(postId);
        indexPostRepository.deleteById(postId);

        webSocketService.sendDeletePostEvent(postId);
        seoCacheService.removeAllPagesCache(postId);
        seoCacheListenerProxy.rewriteCachedIndex();
    }



    private String getKey() {
        return "elasticsearch:"+ IndexPost.INDEX+":building";
    }

    private void refreshFulltextIndex() {
        LOGGER.info("Starting refreshing elasticsearch index {}", IndexPost.INDEX);
        final Collection<Long> postIds = postRepository.findPostIds();

        final Collection<IndexPost> toSave = new ArrayList<>();
        for (Long id: postIds) {
            Optional<com.github.nkonev.blog.entity.jdbc.Post> post = postRepository.findById(id);
            if (post.isPresent()) {
                com.github.nkonev.blog.entity.jdbc.Post jpaPost = post.get();
                LOGGER.debug("Converting PostgreSQL -> Elasticsearch post id={}", id);
                IndexPost indexPost = postConverter.toElasticsearchPost(jpaPost);
                toSave.add(indexPost);

                if (toSave.size() > refreshBatchSize-1){
                    saveToElasticsearch(toSave);
                }
            }
        }
        saveToElasticsearch(toSave);

        removeOrphansFromIndex();
        LOGGER.info("Finished refreshing elasticsearch index {}", IndexPost.INDEX);
    }

    private void saveToElasticsearch(Collection<IndexPost> toSave) {
        if (!toSave.isEmpty()) {
            indexPostRepository.saveAll(toSave);
        }
        LOGGER.info("Flushed {} items to elasticsearch index", toSave.size());
        toSave.clear();
    }

    public void removeOrphansFromIndex() {
        List<Long> toDeleteFromIndex = new ArrayList<>();
        for(int page=0; ;page++) {
            PageRequest pageRequest = PageRequest.of(page, chunkSize);
            final String[] includes = {"_"};
            NativeSearchQuery searchQuery = new NativeSearchQueryBuilder()
                    .withQuery(matchAllQuery())
                    .withPageable(pageRequest)
                    .withSourceFilter(new FetchSourceFilter(includes, null))
                    .build();
            List<String> idsString = elasticsearchTemplate
                    .search(searchQuery, IndexPost.class, IndexCoordinates.of(INDEX))
                    .map(SearchHit::getId).toList();
            LOGGER.info("Get {} index ids", idsString.size());
            if (idsString.isEmpty()) {
                break;
            }
            idsString.stream().map(Long::valueOf).forEach(id -> {
                if (!postRepository.existsById(id)){
                    toDeleteFromIndex.add(id);
                }
            });
        }
        LOGGER.info("Found {} orphan posts in index", toDeleteFromIndex.size());
        for (Long id: toDeleteFromIndex) {
            LOGGER.info("Deleting orphan post id={} from index", id);
            indexPostRepository.deleteById(id);
        }
    }

    public void refreshFulltextIndex(boolean ignoreInProgress){
        final String key = getKey();
        final boolean wasSet = Boolean.TRUE.equals(redisTemplate.opsForValue().setIfAbsent(key, "true"));

        if (wasSet || ignoreInProgress) {
            LOGGER.info("Probe is successful, so we'll refresh elasticsearch index");
            redisTemplate.expire(key, timeout, timeUnit);

            refreshFulltextIndex();

            redisTemplate.delete(key);
            LOGGER.info("Successful delete probe");
        } else {
            LOGGER.info("Probe isn't successful, so we won't refresh elasticsearch index");
        }

    }
}
