package com.github.nkonev.blog.controllers;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.blog.AbstractUtTestRunner;
import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.TestConstants;
import com.github.nkonev.blog.config.CustomConfig;
import com.github.nkonev.blog.dto.*;
import com.github.nkonev.blog.repository.jdbc.PostRepository;
import com.github.nkonev.blog.repository.jdbc.UserAccountRepository;
import com.github.nkonev.blog.services.PostService;
import com.github.nkonev.blog.services.SeoCacheService;
import com.github.nkonev.blog.utils.PageUtils;
import org.hamcrest.Matchers;
import org.hamcrest.core.StringStartsWith;
import org.junit.jupiter.api.*;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.http.HttpMethod;
import org.springframework.http.MediaType;
import org.springframework.security.authentication.AuthenticationCredentialsNotFoundException;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.test.context.support.WithUserDetails;
import org.springframework.test.web.client.ExpectedCount;
import org.springframework.test.web.client.MockRestServiceServer;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders;

import java.util.ArrayList;
import java.util.Collection;
import java.util.List;
import java.util.function.Predicate;

import static com.github.nkonev.blog.utils.SeoCacheKeyUtils.RENDERTRON_HTML;
import static com.github.nkonev.blog.utils.SeoCacheKeyUtils.getRedisKeyForIndex;
import static com.github.nkonev.blog.utils.SeoCacheKeyUtils.getRedisKeyHtmlForPost;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.client.match.MockRestRequestMatchers.method;
import static org.springframework.test.web.client.match.MockRestRequestMatchers.requestTo;
import static org.springframework.test.web.client.response.MockRestResponseCreators.withSuccess;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.xpath;

public class PostControllerTest extends AbstractUtTestRunner {

    private static final Logger LOGGER = LoggerFactory.getLogger(PostControllerTest.class);

    @Autowired
    private PostController postController;

    @Autowired
    private PostRepository postRepository;

    @Autowired
    private ObjectMapper objectMapper;

    @Autowired
    private RedisTemplate<String, String> redisTemplate;

    @Autowired
    private SeoCacheService seoCacheService;

    private MockRestServiceServer mockServer;

    @Autowired
    private CustomConfig customConfig;

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private PostService postService;

    @BeforeEach
    public void setUp() {
        mockServer = MockRestServiceServer.createServer(restTemplate);

        // removed that posts those orphaned in elasticsearch due transaction rollback
        postService.removeOrphansFromIndex();
    }

    @AfterEach
    public void tearDown(){
        redisTemplate.delete(RENDERTRON_HTML+"*");
    }

    public static class PostDtoBuilder {
        public static class Instance {
            private final PostDTO postDTO;
            {
                postDTO = new PostDTO(
                        0,
                        "default new post title",
                        "default new post text",
                        "/logo_mono.png",
                        null,
                        null,
                        new OwnerDTO(),
                        false
                );
            }
            public PostDTO build() {
                return postDTO;
            }

            public Instance id(long id) {
                postDTO.setId(id);
                return this;
            }

            public Instance text(String s) {
                postDTO.setText(s);
                return this;
            }

            public Instance titleImg(String o) {
                postDTO.setTitleImg(o);
                return this;
            }
        }

        public static Instance startBuilding() {
            return new Instance();
        }
    }

    public static final long FOREIGN_POST = 50;
    public static final long DRAFT_POST = 84;

    private final Predicate<PostDTO> hasDraftPostPredicate = postDTO -> postDTO.getId() == DRAFT_POST;

    @Test
    public void testAnonymousCanGetPostsAndItsLimiting() throws Exception {
        MvcResult getPostsRequest = mockMvc.perform(
                MockMvcRequestBuilders.get(Constants.Urls.API+ Constants.Urls.POST)
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.size()").value(PageUtils.DEFAULT_SIZE))
                .andExpect(jsonPath("$.data[0].commentCount").value(1))
                .andExpect(jsonPath("$.data[1].commentCount").value(0))
                .andExpect(jsonPath("$.data[2].commentCount").value(501))
                .andReturn();
        String getStr = getPostsRequest.getResponse().getContentAsString();
        LOGGER.info(getStr);
    }

    @Test
    public void testTrimmed() throws Exception {
        MvcResult getPostsRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"?searchString= ")
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.size()").value(PageUtils.DEFAULT_SIZE))
                .andReturn();
    }

    @Test
    public void testFulltextSearch() throws Exception {
        MvcResult getPostsRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"?searchString=рыбами")
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.size()").value(PageUtils.DEFAULT_SIZE))
                .andExpect(jsonPath("$.data[0].title").value("generated_post_100"))
                .andExpect(jsonPath("$.data[0].text").value("Lorem Ipsum - это текст-\"<b>рыба</b>\", часто используемый в печати и вэб-дизайне. Lorem Ipsum является стандартной \"<b>рыбой</b>\" для текстов на латинице с начала XVI"))
                .andReturn();
    }

    @DisplayName("Префиксный поиск")
    @Test
    public void testPrefixSearch() throws Exception {
        MvcResult getPostsRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"?searchString=исп")
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.size()").value(PageUtils.DEFAULT_SIZE))
                .andExpect(jsonPath("$.data[0].title").value("generated_post_100"))
                .andExpect(jsonPath("$.data[0].text").value("Lorem Ipsum - это текст-\"рыба\", часто <b>используемый</b> в печати и вэб-дизайне. Lorem Ipsum является стандартной \"рыбой\" для текстов на латинице с начала XVI... безымянный печатник создал большую коллекцию размеров и форм шрифтов, <b>используя</b> Lorem Ipsum для распечатки образцов. Lorem Ipsum не только успешно пережил... программы электронной вёрстки типа Aldus PageMaker, в шаблонах которых <b>используется</b> Lorem Ipsum."))
                .andReturn();
    }

    @Test
    public void testSlopSearch() throws Exception {
        MvcResult getPostsRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"?searchString=является начать")
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.size()").value(PageUtils.DEFAULT_SIZE))
                .andExpect(jsonPath("$.data[0].title").value("generated_post_100"))
                .andExpect(jsonPath("$.data[0].text").value("используемый в печати и вэб-дизайне. Lorem Ipsum <b>является</b> стандартной \"рыбой\" для текстов на латинице с <b>начала</b> XVI века. В то время некий безымянный печатник"))
                .andReturn();
    }

    @Test
    public void testContainsSearch() throws Exception {
        MvcResult getPostsRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"?searchString=psum")
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.size()").value(PageUtils.DEFAULT_SIZE))
                .andExpect(jsonPath("$.data[0].title").value("generated_post_100"))
                .andExpect(jsonPath("$.data[0].text").value("Lorem <b>Ipsum</b> - это текст-\"рыба\", часто используемый в печати и вэб-дизайне. Lorem <b>Ipsum</b> является стандартной \"рыбой\" для текстов на латинице с начала XVI... большую коллекцию размеров и форм шрифтов, используя Lorem <b>Ipsum</b> для распечатки образцов. Lorem <b>Ipsum</b> не только успешно пережил без заметных изменений пять... Lorem <b>Ipsum</b> в 60-х годах и, в более недавнее время, программы электронной вёрстки типа Aldus PageMaker, в шаблонах которых используется Lorem <b>Ipsum</b>."))
                .andReturn();
    }

    @Test
    public void testContainsSearch2() throws Exception {
        MvcResult getPostsRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"?searchString=CommitFailedException")
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.size()").value(1))
                .andExpect(jsonPath("$.data[0].title").value("Hi from kafka"))
                .andExpect(jsonPath("$.data[0].text").value("Consumer has failed with exception: <b>org.apache.kafka.clients.consumer.CommitFailedException</b>: Commit cannot be completed due to group rebalance class com... messagehub.consumer.Consumer is shutting down. <b>org.apache.kafka.clients.consumer.CommitFailedException</b>: Commit cannot be completed due to group rebalance"))
                .andReturn();
    }


    @WithUserDetails(TestConstants.USER_ADMIN)
    @Test
    public void testFulltextSearchHostPort() throws Exception {
        final String newPostRendered = "<body>Post Rendered</body>";
        mockServer.expect(requestTo(new StringStartsWith(true, "http://rendertron.example.com:3000/"+customConfig.getBaseUrl()+"/post/")))
                .andExpect(method(HttpMethod.GET))
                .andRespond(withSuccess(newPostRendered, MediaType.TEXT_HTML));

        final String newIndexRendered = "<body>Index Rendered</body>";
        mockServer.expect(requestTo("http://rendertron.example.com:3000/"+customConfig.getBaseUrl()))
                .andExpect(method(HttpMethod.GET))
                .andRespond(withSuccess(newIndexRendered, MediaType.TEXT_HTML));

        UserAccountDetailsDTO userAccountDetailsDTO = (UserAccountDetailsDTO) SecurityContextHolder.getContext().getAuthentication().getPrincipal();
        postController.updatePost(userAccountDetailsDTO, new PostDTO(50L, "edited for search host port", "A new host for test www.google.com:80 with port too", "", null, null, null, false));

        MvcResult getPostsRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"?searchString=www.google.com:80")
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andDo(
                        mvcResult -> {
                            LOGGER.info(mvcResult.getResponse().getContentAsString());
                        }
                )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.size()").value(1))
                .andExpect(jsonPath("$.data[0].title").value("edited for search host port"))
                .andExpect(jsonPath("$.data[0].text").value("A new host for test <b>www.google.com:80</b> with port too"))
                .andReturn();
    }


    @Test
    public void test404() throws Exception {
        MvcResult getPostsRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"/1005001")
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andExpect(status().isNotFound())
                .andExpect(jsonPath("$.error").value("data not found"))
                .andExpect(jsonPath("$.message").value("Post 1005001 not found"))
                .andReturn();
    }


    @Test
    public void test404xml() throws Exception {
        MvcResult getPostsRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"/1005001")
                        .accept(MediaType.APPLICATION_XML_VALUE)
        )
                .andExpect(status().isNotFound())
                .andExpect(xpath("/BlogError/error").string("data not found"))
                .andExpect(xpath("/BlogError/message").string("Post 1005001 not found"))
                .andReturn();
    }


    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void testUserCanAddAndUpdateAndCannotDeletePost() throws Exception {
        final String newPostRendered = "<body>Post Rendered</body>";
        mockServer.expect(ExpectedCount.times(2), requestTo(new StringStartsWith(true, "http://rendertron.example.com:3000/"+customConfig.getBaseUrl()+"/post/")))
                .andExpect(method(HttpMethod.GET))
                .andRespond(withSuccess(newPostRendered, MediaType.TEXT_HTML));

        final String newIndexRendered = "<body>Index Rendered</body>";
        mockServer.expect(ExpectedCount.times(2), requestTo("http://rendertron.example.com:3000/"+customConfig.getBaseUrl()))
                .andExpect(method(HttpMethod.GET))
                .andRespond(withSuccess(newIndexRendered, MediaType.TEXT_HTML));


        final String oldDataIndex = "<html>bad old index data</html>";
        redisTemplate.opsForValue().set(getRedisKeyForIndex(), oldDataIndex);

        MvcResult addPostRequest = mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.POST)
                        .content(objectMapper.writeValueAsString(PostDtoBuilder.startBuilding().build()))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.owner.login").value(TestConstants.USER_ALICE))
                .andExpect(jsonPath("$.canEdit").value(true))
                .andExpect(jsonPath("$.canDelete").value(false))
                .andExpect(jsonPath("$.createDateTime").isNotEmpty())
                .andExpect(jsonPath("$.editDateTime").isEmpty())
                .andReturn();

        Assertions.assertFalse(oldDataIndex.equals(redisTemplate.opsForValue().get(getRedisKeyForIndex())));
        Assertions.assertEquals(newIndexRendered, redisTemplate.opsForValue().get(getRedisKeyForIndex()));

        long id = objectMapper.readValue(addPostRequest.getResponse().getContentAsString(), PostDTOWithAuthorization.class).getId();
        Assertions.assertTrue(redisTemplate.hasKey(getRedisKeyHtmlForPost(id)));

        String addStr = addPostRequest.getResponse().getContentAsString();
        LOGGER.info(addStr);
        PostDTO added = objectMapper.readValue(addStr, PostDTO.class);

        // check post present in my posts
        MvcResult getMyPostsRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+ Constants.Urls.MY)
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andExpect(status().isOk())
                .andReturn();
        String strListPosts = getMyPostsRequest.getResponse().getContentAsString();
        LOGGER.info(strListPosts);
        List<PostDTO> posts = objectMapper.readValue(strListPosts, new TypeReference<List<PostDTO>>(){});
        Assertions.assertTrue(posts.stream().anyMatch(postDTO -> postDTO.getTitle().equals("default new post title")), "I should can see my created post");

        // check foreign post not present in my posts
        Assertions.assertFalse(posts.stream().anyMatch(postDTO -> postDTO.getTitle().startsWith("generated_post")), "foreign post shouldn't be in my posts");

        // check Alice can update her post
        final String updatedTitle = "updated title";
        final String oldCachedPost = "<html>old post data</html>";
        added.setTitle(updatedTitle);
        redisTemplate.opsForValue().set(getRedisKeyHtmlForPost(added.getId()), oldCachedPost);
        MvcResult updatePostRequest = mockMvc.perform(
                put(Constants.Urls.API+ Constants.Urls.POST)
                        .content(objectMapper.writeValueAsString(added))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.title").value(updatedTitle))
                .andExpect(jsonPath("$.owner.login").value(TestConstants.USER_ALICE))
                .andExpect(jsonPath("$.canEdit").value(true))
                .andExpect(jsonPath("$.canDelete").value(false))
                .andExpect(jsonPath("$.createDateTime").isNotEmpty())
                .andExpect(jsonPath("$.editDateTime").isNotEmpty())
                .andReturn();
        LOGGER.info(updatePostRequest.getResponse().getContentAsString());
        Assertions.assertFalse(oldCachedPost.equals(redisTemplate.opsForValue().get(getRedisKeyHtmlForPost(added.getId()))));
        Assertions.assertEquals(newPostRendered, redisTemplate.opsForValue().get(getRedisKeyHtmlForPost(id)));

        MvcResult deleteResult = mockMvc.perform(
                delete(Constants.Urls.API+ Constants.Urls.POST+"/"+added.getId()).with(csrf())
        )
                .andExpect(status().isForbidden())
                .andReturn();

        mockServer.verify();
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void testPostWithoutTitleImageWillSetAutoFromContent() throws Exception {
        final String newPostRendered = "<body>Post Rendered</body>";
        mockServer.expect(requestTo(new StringStartsWith(true, "http://rendertron.example.com:3000/"+customConfig.getBaseUrl()+"/post/")))
                .andExpect(method(HttpMethod.GET))
                .andRespond(withSuccess(newPostRendered, MediaType.TEXT_HTML));

        final String newIndexRendered = "<body>Index Rendered</body>";
        mockServer.expect(requestTo("http://rendertron.example.com:3000/"+customConfig.getBaseUrl()))
                .andExpect(method(HttpMethod.GET))
                .andRespond(withSuccess(newIndexRendered, MediaType.TEXT_HTML));

        MvcResult addPostRequest = mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.POST)
                        .content(objectMapper.writeValueAsString(PostDtoBuilder.startBuilding().text("Lorem ipsum <img src=\"/api/image/post/content/uuid.jpeg\" />").titleImg(null).build()))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.titleImg").value("/api/image/post/content/uuid.jpeg"))
                .andReturn();
    }

    @Test
    public void testAnonymousCannotAddPostUnit() throws Exception {
        Assertions.assertThrows(AuthenticationCredentialsNotFoundException.class, () -> {
            postController.addPost(null, PostDtoBuilder.startBuilding().build());
        });
    }

    @Test
    public void testAnonymousCannotAddPost() throws Exception {
        MvcResult addPostRequest = mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.POST)
                        .content(objectMapper.writeValueAsString(PostDtoBuilder.startBuilding().build()))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isUnauthorized())
                .andReturn();
        LOGGER.info(addPostRequest.getResponse().getContentAsString());
    }

    @Test
    public void testAnonymousCannotUpdatePost() throws Exception {
        final long foreignPostId = FOREIGN_POST;
        PostDTO postDTO = PostDtoBuilder.startBuilding().id(foreignPostId).build();

        MvcResult addPostRequest = mockMvc.perform(
                put(Constants.Urls.API+ Constants.Urls.POST)
                        .content(objectMapper.writeValueAsString(postDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isUnauthorized())
                .andReturn();
        LOGGER.info(addPostRequest.getResponse().getContentAsString());
    }

    @Test
    public void testAnonymousCannotSeeDraftPost() throws Exception {
        MvcResult addPostRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"/"+DRAFT_POST)
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andExpect(status().isNotFound())
                .andReturn();
        LOGGER.info(addPostRequest.getResponse().getContentAsString());
    }

    @Test
    public void testAnonymousCannotSeeDraftPostInLeft() throws Exception {
        MvcResult addPostRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"/"+(DRAFT_POST+1))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andDo(result -> {
                    LOGGER.info("Response: {} {}", result.getResponse().getStatus(), result.getResponse().getContentAsString());
                })
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.left.id").value(Matchers.not((int)DRAFT_POST)))
                .andReturn();
        LOGGER.info(addPostRequest.getResponse().getContentAsString());
    }

    @Test
    public void testAnonymousCannotSeeDraftPostInRight() throws Exception {
        MvcResult addPostRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"/"+(DRAFT_POST-1))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andDo(result -> {
                    LOGGER.info("Response: {} {}", result.getResponse().getStatus(), result.getResponse().getContentAsString());
                })
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.right.id").value(Matchers.not((int)DRAFT_POST)))
                .andReturn();
        LOGGER.info(addPostRequest.getResponse().getContentAsString());
    }

    private List<PostDTO> getAllPosts(String urlPrefix) throws Exception {
        final int pageSize = 20;
        final List<PostDTO> allPosts = new ArrayList<>();
        int lastSize;
        int p = 0;
        do {
            MvcResult getPostRequest = mockMvc.perform(
                    get(urlPrefix + "?page=" + p + "&size=" + pageSize)
            )
                    .andExpect(status().isOk())
                    .andReturn();
            String getStr = getPostRequest.getResponse().getContentAsString();
            LOGGER.info(getStr);
            Wrapper<PostDTO> wrapper = objectMapper.readValue(getStr, new TypeReference<Wrapper<PostDTO>>() {});
            Collection<PostDTO> list = wrapper.getData();
            lastSize = list.size();
            allPosts.addAll(list);
            ++p;
        } while (lastSize == pageSize);
        return allPosts;
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void testUserCannotSeeForeignDrafts() throws Exception {
        Assertions.assertFalse(getAllPosts(Constants.Urls.API + Constants.Urls.POST).stream().anyMatch(hasDraftPostPredicate));
    }

    @WithUserDetails(TestConstants.USER_NIKITA)
    @Test
    public void testUserCanSeeHisDrafts() throws Exception {
        Assertions.assertTrue(getAllPosts(Constants.Urls.API + Constants.Urls.POST).stream().anyMatch(hasDraftPostPredicate));
    }

    private Long getUserId(String userName){
        return userAccountRepository.findByUsername(userName).orElseThrow().getId();
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void testUserCannotSeeForeignDraftsInUsersPosts() throws Exception {
        Long id = getUserId(TestConstants.USER_ALICE);
        Assertions.assertFalse(getAllPosts(Constants.Urls.API+Constants.Urls.USER + "/" + id + Constants.Urls.POSTS).stream().anyMatch(hasDraftPostPredicate));
    }

    @Test
    public void testAnonymousCannotSeeForeignDraftsInUsersPosts() throws Exception {
        Long id = getUserId(TestConstants.USER_ALICE);
        Assertions.assertFalse(getAllPosts(Constants.Urls.API+Constants.Urls.USER + "/" + id + Constants.Urls.POSTS).stream().anyMatch(hasDraftPostPredicate));
    }

    @WithUserDetails(TestConstants.USER_NIKITA)
    @Test
    public void testUserCanSeeHisDraftsInUsersPosts() throws Exception {
        Long id = getUserId(TestConstants.USER_NIKITA);
        Assertions.assertTrue(getAllPosts(Constants.Urls.API+Constants.Urls.USER + "/" + id + Constants.Urls.POSTS).stream().anyMatch(hasDraftPostPredicate));
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void testUserCannotSeeForeignDraft() throws Exception {
        MvcResult getPostRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"/"+DRAFT_POST)
        )
                .andExpect(status().isNotFound())
                .andReturn();
    }

    @Test
    public void testAnonymousCannotSeeForeignDraft() throws Exception {
        MvcResult getPostRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"/"+DRAFT_POST)
        )
                .andExpect(status().isNotFound())
                .andReturn();
    }


    @Test
    public void testAnonymousCannotDeletePost() throws Exception {
        final long foreignPostId = FOREIGN_POST;
        PostDTO postDTO = PostDtoBuilder.startBuilding().id(foreignPostId).build();

        MvcResult addPostRequest = mockMvc.perform(
                delete(Constants.Urls.API+ Constants.Urls.POST+"/"+foreignPostId).with(csrf())
                        .content(objectMapper.writeValueAsString(postDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isUnauthorized())
                .andReturn();
        LOGGER.info(addPostRequest.getResponse().getContentAsString());
    }



    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void testUserCannotUpdateForeignPost() throws Exception {
        final long foreignPostId = FOREIGN_POST;

        MvcResult getPostRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"/"+foreignPostId)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.canEdit").value(false))
                .andExpect(jsonPath("$.canDelete").value(false))
                .andReturn();
        String getStr = getPostRequest.getResponse().getContentAsString();
        LOGGER.info(getStr);
        PostDTO foreign = objectMapper.readValue(getStr, PostDTO.class);


        MvcResult addPostRequest = mockMvc.perform(
                put(Constants.Urls.API+ Constants.Urls.POST)
                        .content(objectMapper.writeValueAsString(foreign))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isForbidden())
                .andReturn();
        String addStr = addPostRequest.getResponse().getContentAsString();
        LOGGER.info(addStr);

    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void testUserCannotDeleteForeignPost() throws Exception {
        final long foreignPostId = FOREIGN_POST;
        PostDTO postDTO = PostDtoBuilder.startBuilding().id(foreignPostId).build();

        MvcResult addPostRequest = mockMvc.perform(
                delete(Constants.Urls.API+ Constants.Urls.POST+"/"+foreignPostId).with(csrf())
                        .content(objectMapper.writeValueAsString(postDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isForbidden())
                .andReturn();
        LOGGER.info(addPostRequest.getResponse().getContentAsString());
    }


    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void testUserCannotRecreateExistsPost() throws Exception {
        final long foreignPostId = FOREIGN_POST;

        PostDTO postDTO = PostDtoBuilder.startBuilding().id(foreignPostId).build();
        MvcResult addPostRequest = mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.POST)
                        .content(objectMapper.writeValueAsString(postDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isBadRequest())
                .andReturn();
        String addStr = addPostRequest.getResponse().getContentAsString();
        LOGGER.info(addStr);
    }

    @WithUserDetails(TestConstants.USER_ADMIN)
    @Test
    public void testAdminCanUpdateForeignPost() throws Exception {
        final String newPostRendered = "<body>Post Rendered</body>";
        mockServer.expect(requestTo(new StringStartsWith(true, "http://rendertron.example.com:3000/"+customConfig.getBaseUrl()+"/post/")))
                .andExpect(method(HttpMethod.GET))
                .andRespond(withSuccess(newPostRendered, MediaType.TEXT_HTML));

        final String newIndexRendered = "<body>Index Rendered</body>";
        mockServer.expect(requestTo("http://rendertron.example.com:3000/"+customConfig.getBaseUrl()))
                .andExpect(method(HttpMethod.GET))
                .andRespond(withSuccess(newIndexRendered, MediaType.TEXT_HTML));


        final long foreignPostId = FOREIGN_POST;

        MvcResult getPostRequest = mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.POST+"/"+foreignPostId)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.canEdit").value(true))
                .andExpect(jsonPath("$.canDelete").value(true))
                .andReturn();
        String getStr = getPostRequest.getResponse().getContentAsString();
        LOGGER.info(getStr);
        PostDTO foreign = objectMapper.readValue(getStr, PostDTO.class);

        final String title = "title updated by admin";
        foreign.setTitle(title);
        MvcResult updatePostRequest = mockMvc.perform(
                put(Constants.Urls.API+ Constants.Urls.POST)
                        .content(objectMapper.writeValueAsString(foreign))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.title").value(title))
                .andReturn();
        String addStr = updatePostRequest.getResponse().getContentAsString();
        LOGGER.info(addStr);
        mockServer.verify();
    }

    @WithUserDetails(TestConstants.USER_ADMIN)
    @Test
    public void testAdminCanDeleteForeignPost() throws Exception {
        final long foreignPostId = FOREIGN_POST;

        // add some comments
        {
            MvcResult addCommentRequest = mockMvc.perform(
                    post(Constants.Urls.API+ Constants.Urls.POST+"/"+foreignPostId+"/"+ Constants.Urls.COMMENT)
                            .content(objectMapper.writeValueAsString(CommentControllerTest.CommentDtoBuilder.startBuilding().build()))
                            .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                            .with(csrf())
            )
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.owner.login").value(TestConstants.USER_ADMIN))
                    .andExpect(jsonPath("$.canEdit").value(true))
                    .andExpect(jsonPath("$.canDelete").value(true))
                    .andReturn();
        }

        final String newIndexRendered = "<body>Index Rendered</body>";
        mockServer.expect(requestTo("http://rendertron.example.com:3000/"+customConfig.getBaseUrl()))
                .andExpect(method(HttpMethod.GET))
                .andRespond(withSuccess(newIndexRendered, MediaType.TEXT_HTML));

        MvcResult deletePostRequest = mockMvc.perform(
                delete(Constants.Urls.API+ Constants.Urls.POST+"/"+foreignPostId).with(csrf())
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andReturn();

        LOGGER.info(deletePostRequest.getResponse().getContentAsString());
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void xssText() throws Exception {
        final String newPostRendered = "<body>Post Rendered</body>";
        mockServer.expect(requestTo(new StringStartsWith(true, "http://rendertron.example.com:3000/"+customConfig.getBaseUrl()+"/post/")))
                .andExpect(method(HttpMethod.GET))
                .andRespond(withSuccess(newPostRendered, MediaType.TEXT_HTML));

        final String newIndexRendered = "<body>Index Rendered</body>";
        mockServer.expect(requestTo("http://rendertron.example.com:3000/"+customConfig.getBaseUrl()))
                .andExpect(method(HttpMethod.GET))
                .andRespond(withSuccess(newIndexRendered, MediaType.TEXT_HTML));


        MvcResult addPostRequest = mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.POST)
                        .content(objectMapper.writeValueAsString(PostDtoBuilder.startBuilding().text("Harmless <script>alert('XSS')</script>text").build()))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.owner.login").value(TestConstants.USER_ALICE))
                .andExpect(jsonPath("$.text").value("Harmless text"))
                .andReturn();
        String addStr = addPostRequest.getResponse().getContentAsString();
        LOGGER.info(addStr);
        PostDTO added = objectMapper.readValue(addStr, PostDTO.class);

    }
}
