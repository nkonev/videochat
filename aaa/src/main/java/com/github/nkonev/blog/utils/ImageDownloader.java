package com.github.nkonev.blog.utils;

import com.github.nkonev.blog.controllers.AbstractImageUploadController;
import com.github.nkonev.blog.controllers.ImageOperations;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.ResponseEntity;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.io.InputStream;

@Service
public class ImageDownloader {

    @Autowired
    private RestTemplate restTemplate;

    private static final Logger LOGGER = LoggerFactory.getLogger(ImageDownloader.class);

    public String downloadImageAndSave(String url, ImageOperations saveTo) {
        LOGGER.info("Start downloading {} by {}", url, saveTo.getClass().getName());
        ResponseEntity<byte[]> responseEntity = restTemplate.getForEntity(url, byte[].class);
        byte[] body = responseEntity.getBody();
        try(InputStream is = new ByteArrayInputStream(body);) {
            AbstractImageUploadController.ImageResponse imageResponse = saveTo.insertImage(body.length, responseEntity.getHeaders().getContentType().toString(), is);
            LOGGER.info("Successfully downloaded and saved {} to {}", url, imageResponse.getRelativeUrl());
            return imageResponse.getRelativeUrl();
        } catch (IOException e) {
            LOGGER.error("Error during downloading and saving {}. Check server logs.", url);
            throw new RuntimeException(e);
        }
    }
}
