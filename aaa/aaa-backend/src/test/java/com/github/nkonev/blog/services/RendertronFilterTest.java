package com.github.nkonev.blog.services;

import com.github.nkonev.blog.AbstractUtTestRunner;
import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.config.CustomConfig;
import org.hamcrest.core.StringContains;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.http.HttpMethod;
import org.springframework.http.MediaType;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.client.ExpectedCount;
import org.springframework.test.web.client.MockRestServiceServer;
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders;

import static com.github.nkonev.blog.utils.SeoCacheKeyUtils.RENDERTRON_HTML;
import static org.hamcrest.Matchers.not;
import static org.springframework.test.web.client.match.MockRestRequestMatchers.method;
import static org.springframework.test.web.client.match.MockRestRequestMatchers.requestTo;
import static org.springframework.test.web.client.response.MockRestResponseCreators.withSuccess;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@TestPropertySource(properties = {
        "custom.rendertron.enable=true",
        "custom.rendertron.crawlerUserAgents=YaBot",
        "custom.seo.script=classpath:seo.html",
        "server.port=9082",
        "management.server.port=3012",
        "spring.flyway.drop-first=false",
        "custom.elasticsearch.refresh-on-start=false",
        "custom.elasticsearch.drop-first=false"
})
public class RendertronFilterTest extends AbstractUtTestRunner {

    private static final Logger LOGGER = LoggerFactory.getLogger(RendertronFilterTest.class);

    private MockRestServiceServer mockServer;


    @Autowired
    private RedisTemplate<String, String> redisTemplate;

    @Autowired
    private CustomConfig customConfig;

    @BeforeEach
    public void setUp() {
        mockServer = MockRestServiceServer.createServer(restTemplate);
    }

    @AfterEach
    public void tearDown(){
        redisTemplate.delete(RENDERTRON_HTML+"*");
    }

    @Test
    public void testSeoHtmlInjectionBeforeClosingHeadWorks() throws Exception {
        final String newIndexRendered = "<head></head><body>Index Rendered</body>";
        mockServer.expect(ExpectedCount.once(), requestTo("http://rendertron.example.com:3000/"+customConfig.getBaseUrl()))
                .andExpect(method(HttpMethod.GET))
                .andRespond(withSuccess(newIndexRendered, MediaType.TEXT_HTML));


        mockMvc.perform(
                MockMvcRequestBuilders.get(Constants.Urls.ROOT)
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .accept(MediaType.TEXT_HTML)
                .header("User-Agent", "YaBot")
        ).andDo(result -> {
            LOGGER.info("result body={}", result.getResponse().getContentAsString());
        })
                .andExpect(status().isOk())
                .andExpect(content().string(new StringContains(true, "Index Rendered")))
                .andExpect(content().string(new StringContains(true, "<script>console.log(\"Seo\");</script>")))
                .andReturn();

        mockServer.verify();
    }

    @Test
    public void testSeoScriptInjectionWorksWhenNonBot() throws Exception {

        mockMvc.perform(
                MockMvcRequestBuilders.get(Constants.Urls.ROOT)
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .accept(MediaType.TEXT_HTML)
                        .header("User-Agent", "Common User Browser")
        ).andDo(result -> {
            LOGGER.info("result body={}", result.getResponse().getContentAsString());
        })
                .andExpect(status().isOk())
                .andExpect(content().string(new StringContains(true, "<div id=\"app-container\"></div>")))
                .andExpect(content().string(new StringContains(true, "<script>console.log(\"Seo\");</script>")))
                .andReturn();

    }


    @Test
    public void testSeoScriptInjectionNotWorksWhenBot() throws Exception {
        mockServer.expect(ExpectedCount.never(), requestTo("http://rendertron.example.com:3000/http://127.0.0.1:9082"))
                .andExpect(method(HttpMethod.GET));

        mockMvc.perform(
                MockMvcRequestBuilders.get(Constants.Urls.ROOT)
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .accept(MediaType.TEXT_HTML)
                        .header("User-Agent", "HeadlessChrome/65.1.2.3.4")
        ).andDo(result -> {
            LOGGER.info("result body={}", result.getResponse().getContentAsString());
        })
                .andExpect(status().isOk())
                .andExpect(content().string(new StringContains(true, "<div id=\"app-container\"></div>")))
                .andExpect(content().string(not(new StringContains(true, "<script>console.log(\"Seo\");</script>"))))
                .andReturn();

        mockServer.verify();

    }

}
