package com.github.nkonev.blog.controllers;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.blog.AbstractUtTestRunner;
import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.TestConstants;
import com.github.nkonev.blog.dto.SettingsDTO;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.mock.web.MockMultipartFile;
import org.springframework.mock.web.MockPart;
import org.springframework.security.test.context.support.WithUserDetails;
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders;

import java.nio.charset.Charset;
import java.nio.charset.StandardCharsets;

import static com.github.nkonev.blog.controllers.SettingsController.DTO_PART;
import static com.github.nkonev.blog.controllers.SettingsController.IMAGE_PART;
import static org.hamcrest.core.Is.is;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

public class SettingsControllerTest extends AbstractUtTestRunner {

    @Autowired
    private ObjectMapper objectMapper;

    private static final Logger LOGGER = LoggerFactory.getLogger(SettingsControllerTest.class);

    @Test
    public void getConfig() throws Exception {
        mockMvc.perform(
                get(Constants.Urls.API+ Constants.Urls.CONFIG)
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.titleTemplate").value("%s | nkonev's blog"))
                .andExpect(jsonPath("$.header").value("Блог Конева Никиты"))
                .andExpect(jsonPath("$.canShowSettings").value(false))
                .andExpect(jsonPath("$.removeImageBackground").doesNotExist())
                .andReturn();

    }

    private MockMultipartFile makeMultipartTextPart(String requestPartName,
                                                    byte[] value, String contentType) throws Exception {

        return new MockMultipartFile(requestPartName, "", contentType,
                value);
    }

    @WithUserDetails(TestConstants.USER_ADMIN)
    @Test
    public void setConfig() throws Exception {
        SettingsDTO s = new SettingsDTO();
        s.setTitleTemplate("bloggo");
        s.setHeader("Header");
        s.setSubHeader("Sub");
        s.setBackgroundColor("green");

        byte[] img0 = {(byte)0xFF, (byte)0x01, (byte)0x1A};
        MockMultipartFile imgPart = new MockMultipartFile(IMAGE_PART, "lol-content.png", "image/png", img0);

        mockMvc.perform(multipart(Constants.Urls.API + Constants.Urls.CONFIG)
                .file(makeMultipartTextPart(DTO_PART, objectMapper.writeValueAsBytes(s), MediaType.APPLICATION_JSON_VALUE))
                .file(imgPart)
                .with(csrf())
        )
                .andDo(result -> {
                    LOGGER.info("Set SettingsDTO response: {}", result.getResponse().getContentAsString());
                })
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.titleTemplate").value("bloggo"))
                .andExpect(jsonPath("$.header").value("Header"))
                .andExpect(jsonPath("$.subHeader").value("Sub"))
                .andExpect(jsonPath("$.backgroundColor").value("green"))
                .andExpect(jsonPath("$.imageBackground").isNotEmpty())
                .andReturn();

    }

}