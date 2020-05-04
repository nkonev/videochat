package com.github.nkonev.blog.controllers;

import java.io.InputStream;

public interface ImageOperations {
    AbstractImageUploadController.ImageResponse insertImage(
            long contentLength,
            String contentType,
            InputStream inputStream
    );
}
