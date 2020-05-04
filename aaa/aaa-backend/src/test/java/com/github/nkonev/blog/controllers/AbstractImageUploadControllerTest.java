package com.github.nkonev.blog.controllers;

import com.github.nkonev.blog.AbstractUtTestRunner;
import com.github.nkonev.blog.TestConstants;
import com.google.common.net.HttpHeaders;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import org.springframework.mock.web.MockMultipartFile;
import org.springframework.security.test.context.support.WithUserDetails;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders;

import java.net.URI;

import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

public abstract class AbstractImageUploadControllerTest extends AbstractUtTestRunner {

    private static final Logger LOGGER = LoggerFactory.getLogger(AbstractImageUploadControllerTest.class);

    protected abstract String postTemplate();

    protected String postImage(String postUrlTemplate, MockMultipartFile mpf) throws Exception {
        MvcResult mvcResult = mockMvc.perform(
                MockMvcRequestBuilders.multipart(postUrlTemplate)
                        .file(mpf).with(csrf())
        )
                .andExpect(status().isOk())
                .andReturn()
                ;
        AbstractImageUploadController.ImageResponse imageResponse = objectMapper.readValue(mvcResult.getResponse().getContentAsString(), AbstractImageUploadController.ImageResponse.class);
        String urlResponse = imageResponse.getUrl();

        LOGGER.info("responsed image url: {}", urlResponse);

        MvcResult result = mockMvc.perform(
                MockMvcRequestBuilders.get(urlResponse)
        )
                .andExpect(status().isOk())
                .andExpect(header().string(HttpHeaders.CONTENT_TYPE, mpf.getContentType()))
                .andReturn()
                ;
        byte[] content = result.getResponse().getContentAsByteArray();

        Assertions.assertArrayEquals(mpf.getBytes(), content);

        return urlResponse;
    }

    @WithUserDetails(TestConstants.USER_NIKITA)
    @org.junit.jupiter.api.Test
    public void postImage() throws Exception {

        byte[] img0 = {(byte)0xFF, (byte)0x01, (byte)0x1A};
        MockMultipartFile mf0 = new MockMultipartFile(ImagePostTitleUploadController.IMAGE_PART, "lol-content.png", "image/png;charset=UTF-8", img0);
        String url0 = postImage(postTemplate(), mf0);

        byte[] img1 = {(byte)0xAA, (byte)0xBB, (byte)0xCC, (byte)0xDD, (byte)0xCC};
        MockMultipartFile mf1 = new MockMultipartFile(ImagePostTitleUploadController.IMAGE_PART, "lol-content.png", "image/png;charset=UTF-8", img1);
        String url1 = postImage(postTemplate(), mf1);

        Assertions.assertNotEquals(url0, url1);

        assertDeletedCount();
    }

    protected abstract void assertDeletedCount();

    protected abstract int clearAbandonedImage();

    @WithUserDetails(TestConstants.USER_NIKITA)
    @org.junit.jupiter.api.Test
    public void postImageAndTwiceGet() throws Exception {

        byte[] img1 = {(byte)0xAA, (byte)0xBB, (byte)0xCC, (byte)0xDD, (byte)0xCC};
        MockMultipartFile mf1 = new MockMultipartFile(ImagePostTitleUploadController.IMAGE_PART, "lol-content.png", "image/png", img1);


        MvcResult mvcResult = mockMvc.perform(
                MockMvcRequestBuilders.multipart(postTemplate())
                        .file(mf1)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andReturn()
                ;
        AbstractImageUploadController.ImageResponse imageResponse = objectMapper.readValue(mvcResult.getResponse().getContentAsString(), AbstractImageUploadController.ImageResponse.class);
        String urlResponse = imageResponse.getUrl();

        LOGGER.info("responsed image url: {}", urlResponse);

        RequestEntity requestEntity1 = RequestEntity.get(URI.create(urlResponse)).build();
        ResponseEntity<byte[]> re1 = testRestTemplate.exchange(requestEntity1, byte[].class);
        Assertions.assertEquals(200, re1.getStatusCodeValue());

        String etag = re1.getHeaders().getFirst(org.springframework.http.HttpHeaders.ETAG);
        Assertions.assertNotNull(etag);

        RequestEntity requestEntity2 = RequestEntity.get(URI.create(urlResponse)).header(org.springframework.http.HttpHeaders.IF_NONE_MATCH, etag).build();
        ResponseEntity<byte[]> re2 = testRestTemplate.exchange(requestEntity2, byte[].class);
        Assertions.assertEquals(304, re2.getStatusCodeValue());
    }

    @Test
    public void putImageUnauthorized() throws Exception {
        byte[] img0 = {(byte)0xFF, (byte)0x01, (byte)0x1A};
        MockMultipartFile mf0 = new MockMultipartFile(ImagePostTitleUploadController.IMAGE_PART, "lol-content.png", "image/png", img0);
        MvcResult mvcResult = mockMvc.perform(
                MockMvcRequestBuilders.multipart(postTemplate())
                        .file(mf0).with(csrf())
        )
                .andExpect(status().isUnauthorized())
                .andReturn()
                ;
    }

}
