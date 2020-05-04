package com.github.nkonev.blog;

import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;
import org.springframework.mock.web.MockHttpServletResponse;
import org.springframework.test.web.servlet.MvcResult;

import java.io.BufferedWriter;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;


/**
 * Created by nik on 28.05.17.
 */
public class Swagger2MarkupTest extends AbstractUtTestRunner {

    @Test
    public void createSpringfoxSwaggerJson() throws Exception {

        String outputDir = TestConstants.SWAGGER_DIR;
        MvcResult mvcResult = this.mockMvc.perform(get(TestConstants.SPRINGFOX_DOCS_URL)
                .accept(MediaType.APPLICATION_JSON))
                .andExpect(status().isOk())
                .andReturn();

        MockHttpServletResponse response = mvcResult.getResponse();
        String swaggerJson = response.getContentAsString();
        Files.createDirectories(Paths.get(outputDir));
        try (BufferedWriter writer = Files.newBufferedWriter(Paths.get(outputDir, TestConstants.SWAGGER_JSON), StandardCharsets.UTF_8)){
            writer.write(swaggerJson);
        }
    }
}
