package com.github.nkonev.blog.controllers;

import com.github.nkonev.blog.AbstractUtTestRunner;
import com.github.nkonev.blog.Constants;
import org.springframework.http.MediaType;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders;

import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@TestPropertySource(properties = {
        "custom.applications[0].title=firstapp",
        "custom.applications[0].srcUrl=http://example.com/",
        "server.port=9083",
        "management.server.port=3013",
        "spring.flyway.drop-first=false",
        "custom.elasticsearch.refresh-on-start=false",
        "custom.elasticsearch.drop-first=false"
})
public class ApplicationControllerTest extends AbstractUtTestRunner {
    @org.junit.jupiter.api.Test
    public void testAnonymousCanGetCommentsAndItsLimiting() throws Exception {
        MvcResult getCommentsRequest = mockMvc.perform(
                MockMvcRequestBuilders.get(Constants.Urls.API+ Constants.Urls.APPLICATION)
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.size()").value(1))
                .andExpect(jsonPath("$[0].id").value(0))
                .andExpect(jsonPath("$[0].title").value("firstapp"))
                .andExpect(jsonPath("$[0].srcUrl").value("http://example.com/"))
                .andReturn();
    }
}
