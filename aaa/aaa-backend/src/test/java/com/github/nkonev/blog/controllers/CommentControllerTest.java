package com.github.nkonev.blog.controllers;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.blog.AbstractUtTestRunner;
import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.TestConstants;
import com.github.nkonev.blog.converter.CommentConverter;
import com.github.nkonev.blog.dto.CommentDTO;
import com.github.nkonev.blog.entity.jdbc.Comment;
import com.github.nkonev.blog.repository.jdbc.CommentRepository;
import com.github.nkonev.blog.utils.PageUtils;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.security.test.context.support.WithUserDetails;
import org.springframework.test.annotation.Repeat;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders;

import static org.hamcrest.CoreMatchers.everyItem;
import static org.hamcrest.core.Is.is;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

public class CommentControllerTest extends AbstractUtTestRunner {

    private static final Logger LOGGER = LoggerFactory.getLogger(CommentControllerTest.class);
    
    @Autowired
    private ObjectMapper objectMapper;

    @Autowired
    private CommentRepository commentRepository;

    @Autowired
    private CommentConverter commentConverter;

    public static class CommentDtoBuilder {
        public static class Instance {
            private final CommentDTO commentDTO;
            {
                commentDTO = new CommentDTO(
                        0,
                        "default comment",
                        null,
                        null
                );
            }
            public CommentDTO build() {
                return commentDTO;
            }

            public Instance id(long id) {
                commentDTO.setId(id);
                return this;
            }

            public Instance text(String s) {
                commentDTO.setText(s);
                return this;
            }
        }
        public static Instance startBuilding() {
            return new Instance();
        }
    }
    
    public static final long POST_UNDER_TEST = 90;

    public static final long FOREIGN_COMMENT_ID = 20;

    public static final long POST_WITH_COMMENTS = 101; // select distinct c.post_id from posts.comment c;

    @org.junit.jupiter.api.Test
    public void testAnonymousCanGetCommentsAndItsLimiting() throws Exception {
        MvcResult getCommentsRequest = mockMvc.perform(
                MockMvcRequestBuilders.get(Constants.Urls.API+ Constants.Urls.POST+"/"+POST_WITH_COMMENTS+"/"+ Constants.Urls.COMMENT)
                .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.size()").value(PageUtils.DEFAULT_SIZE))
                .andExpect(jsonPath("$.data.[*].canEdit").value(everyItem(is(false))))
                .andExpect(jsonPath("$.data.[*].canDelete").value(everyItem(is(false))))
                .andReturn();
    }


    @Repeat(10)
    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void testUserCanAddAndUpdateAndCanDeleteComment() throws Exception {
        MvcResult addCommentRequest = mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.POST+"/"+POST_UNDER_TEST+"/"+ Constants.Urls.COMMENT)
                        .content(objectMapper.writeValueAsString(CommentDtoBuilder.startBuilding().build()))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.owner.login").value(TestConstants.USER_ALICE))
                .andExpect(jsonPath("$.canEdit").value(true))
                .andExpect(jsonPath("$.canDelete").value(true))
                .andExpect(jsonPath("$.createDateTime").isNotEmpty())
                .andExpect(jsonPath("$.editDateTime").isEmpty())
                .andReturn();
        String addStr = addCommentRequest.getResponse().getContentAsString();
        LOGGER.info(addStr);
        CommentDTO added = objectMapper.readValue(addStr, CommentDTO.class);

        // check Alice can update her comment
        final String updatedText = "updated text";
        added.setText(updatedText);
        MvcResult updateCommentRequest = mockMvc.perform(
                put(Constants.Urls.API+ Constants.Urls.POST+"/"+POST_UNDER_TEST+"/"+ Constants.Urls.COMMENT)
                        .content(objectMapper.writeValueAsString(added))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.text").value(updatedText))
                .andExpect(jsonPath("$.owner.login").value(TestConstants.USER_ALICE))
                .andExpect(jsonPath("$.canEdit").value(true))
                .andExpect(jsonPath("$.canDelete").value(true))
                .andExpect(jsonPath("$.createDateTime").isNotEmpty())
                .andExpect(jsonPath("$.editDateTime").isNotEmpty())
                .andReturn();
        LOGGER.info(updateCommentRequest.getResponse().getContentAsString());


        MvcResult deleteResult = mockMvc.perform(
                delete(Constants.Urls.API+ Constants.Urls.POST+"/"+POST_UNDER_TEST+"/"+ Constants.Urls.COMMENT+"/"+added.getId()).with(csrf())
        )
                .andExpect(status().isOk())
                .andReturn();
    }
    
    @org.junit.jupiter.api.Test
    public void testAnonymousCannotAddComment() throws Exception {
        MvcResult addCommentRequest = mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.POST+"/"+POST_UNDER_TEST+"/"+ Constants.Urls.COMMENT)
                        .content(objectMapper.writeValueAsString(CommentDtoBuilder.startBuilding().build()))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isUnauthorized())
                .andReturn();
        LOGGER.info(addCommentRequest.getResponse().getContentAsString());
    }

    @org.junit.jupiter.api.Test
    public void testAnonymousCannotUpdateComment() throws Exception {
        CommentDTO commentDTO = CommentDtoBuilder.startBuilding().id(FOREIGN_COMMENT_ID).build();

        MvcResult addCommentRequest = mockMvc.perform(
                put(Constants.Urls.API+ Constants.Urls.POST+"/"+POST_UNDER_TEST+"/"+ Constants.Urls.COMMENT)
                        .content(objectMapper.writeValueAsString(commentDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isUnauthorized())
                .andReturn();
        LOGGER.info(addCommentRequest.getResponse().getContentAsString());
    }

    @org.junit.jupiter.api.Test
    public void testAnonymousCannotDeleteComment() throws Exception {
        CommentDTO commentDTO = CommentDtoBuilder.startBuilding().id(FOREIGN_COMMENT_ID).build();

        MvcResult addCommentRequest = mockMvc.perform(
                delete(Constants.Urls.API+ Constants.Urls.POST+"/"+POST_UNDER_TEST+"/"+ Constants.Urls.COMMENT+"/"+FOREIGN_COMMENT_ID).with(csrf())
                        .content(objectMapper.writeValueAsString(commentDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isUnauthorized())
                .andReturn();
        LOGGER.info(addCommentRequest.getResponse().getContentAsString());
    }

    public CommentDTO getForeignComment(long id) {
        Comment comment = commentRepository.findById(id).orElseThrow(()->new IllegalArgumentException("comment not found during test"));
        return commentConverter.convertToDto(comment);
    }

    @WithUserDetails(TestConstants.USER_BOB)
    @org.junit.jupiter.api.Test
    public void testUserCannotUpdateForeignComment() throws Exception {

        MvcResult addCommentRequest = mockMvc.perform(
                put(Constants.Urls.API+ Constants.Urls.POST+"/"+POST_UNDER_TEST+"/"+ Constants.Urls.COMMENT)
                        .content(objectMapper.writeValueAsString(getForeignComment(FOREIGN_COMMENT_ID)))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isForbidden())
                .andReturn();
        String addStr = addCommentRequest.getResponse().getContentAsString();
        LOGGER.info(addStr);

    }

    @WithUserDetails(TestConstants.USER_BOB)
    @org.junit.jupiter.api.Test
    public void testUserCannotDeleteForeignComment() throws Exception {
        CommentDTO commentDTO = CommentDtoBuilder.startBuilding().id(FOREIGN_COMMENT_ID).build();

        MvcResult addCommentRequest = mockMvc.perform(
                delete(Constants.Urls.API+ Constants.Urls.POST+"/"+POST_UNDER_TEST+"/"+ Constants.Urls.COMMENT+"/"+FOREIGN_COMMENT_ID).with(csrf())
                        .content(objectMapper.writeValueAsString(commentDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isForbidden())
                .andReturn();
        LOGGER.info(addCommentRequest.getResponse().getContentAsString());
    }


    @WithUserDetails(TestConstants.USER_BOB)
    @org.junit.jupiter.api.Test
    public void testUserCannotRecreateExistsComment() throws Exception {

        CommentDTO commentDTO = CommentDtoBuilder.startBuilding().id(FOREIGN_COMMENT_ID).build();
        MvcResult addCommentRequest = mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.POST+"/"+POST_UNDER_TEST+"/"+ Constants.Urls.COMMENT)
                        .content(objectMapper.writeValueAsString(commentDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isBadRequest())
                .andReturn();
        String addStr = addCommentRequest.getResponse().getContentAsString();
        LOGGER.info(addStr);
    }

    @WithUserDetails(TestConstants.USER_ADMIN)
    @org.junit.jupiter.api.Test
    public void testAdminCanUpdateForeignComment() throws Exception {
        CommentDTO foreign = getForeignComment(FOREIGN_COMMENT_ID);

        final String title = "text updated by admin";
        foreign.setText(title);
        MvcResult updateCommentRequest = mockMvc.perform(
                put(Constants.Urls.API+ Constants.Urls.POST+"/"+POST_UNDER_TEST+"/"+ Constants.Urls.COMMENT)
                        .content(objectMapper.writeValueAsString(foreign))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.text").value(title))
                .andReturn();
        String addStr = updateCommentRequest.getResponse().getContentAsString();
        LOGGER.info(addStr);

    }

    @WithUserDetails(TestConstants.USER_ADMIN)
    @org.junit.jupiter.api.Test
    public void testAdminCanDeleteForeignComment() throws Exception {

        MvcResult deleteCommentRequest = mockMvc.perform(
                delete(Constants.Urls.API+ Constants.Urls.POST+"/"+POST_UNDER_TEST+"/"+ Constants.Urls.COMMENT+"/"+FOREIGN_COMMENT_ID).with(csrf())
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andReturn();
        LOGGER.info(deleteCommentRequest.getResponse().getContentAsString());
    }
}
