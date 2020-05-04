package com.github.nkonev.blog.controllers;

import com.github.nkonev.blog.TestConstants;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.mock.web.MockMultipartFile;
import org.springframework.security.test.context.support.WithUserDetails;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders;
import java.util.Random;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

public class ImagePostContentUploadControllerTest extends AbstractImageUploadControllerTest {

	private static final Logger LOGGER = LoggerFactory.getLogger(ImagePostContentUploadControllerTest.class);

	private static final String POST_TEMPLATE = ImagePostContentUploadController.POST_TEMPLATE;
	private static final String GET_TEMPLATE = ImagePostContentUploadController.GET_TEMPLATE;

	@Autowired
	private ImagePostContentUploadController imagePostContentUploadController;

	@WithUserDetails(TestConstants.USER_NIKITA)
	@org.junit.jupiter.api.Test
	public void putImageWithWrongContentType() throws Exception {
		byte[] img0 = {(byte)0xFF, (byte)0x01, (byte)0x1A};
		MockMultipartFile mf0 = new MockMultipartFile(ImagePostTitleUploadController.IMAGE_PART, "lol-content.png", "image/ololo", img0);

		MvcResult mvcResult = mockMvc.perform(
				MockMvcRequestBuilders.multipart(POST_TEMPLATE)
						.file(mf0).with(csrf())
		)
				.andExpect(status().isUnsupportedMediaType())
				.andExpect(jsonPath("$.error").value("unsupported media type"))
				.andExpect(jsonPath("$.message").value("Incompatible content type. Allowed: [image/png, image/jpg, image/jpeg]"))
				.andReturn()
				;
	}

	@WithUserDetails(TestConstants.USER_NIKITA)
	@Test
	public void putImageWithVeryBigSize() throws Exception {

		// in application.yml 4 Mb allowed. We try to POST 5 Mb
		byte[] img0 = new byte[1024 * 1024 * 5];
		new Random().nextBytes(img0);

		MockMultipartFile mf0 = new MockMultipartFile(ImagePostTitleUploadController.IMAGE_PART, "lol-content.png", "image/ololo", img0);

		MvcResult mvcResult = mockMvc.perform(
				MockMvcRequestBuilders.multipart(POST_TEMPLATE)
						.file(mf0).with(csrf())
		)
				.andExpect(status().isPayloadTooLarge())
				.andExpect(jsonPath("$.error").value("payload too large"))
				.andExpect(jsonPath("$.message").value("Image must be <= 4194304 bytes"))
				.andReturn()
				;
	}


	@Test
	public void getUnexistingImage() throws Exception {

		MvcResult result = mockMvc.perform(
				MockMvcRequestBuilders.get(GET_TEMPLATE, "a979054b-8c9d-4df8-983e-6abc57c2aed6", "png")
		)
				.andExpect(status().isNotFound())
				.andExpect(jsonPath("$.error").value("data not found"))
				.andExpect(jsonPath("$.message").value("post content image with id 'a979054b-8c9d-4df8-983e-6abc57c2aed6' not found"))
				.andReturn()
				;
	}

	@Override
	protected String postTemplate() {
		return POST_TEMPLATE;
	}

	@Override
	protected int clearAbandonedImage() {
		return imagePostContentUploadController.clearPostContentImages();
	}

	@Override
	protected void assertDeletedCount() {
		Assertions.assertTrue(clearAbandonedImage() == 0);
	}
}