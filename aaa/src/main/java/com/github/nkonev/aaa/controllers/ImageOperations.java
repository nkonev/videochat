package com.github.nkonev.aaa.controllers;

import java.io.InputStream;

public interface ImageOperations {
    AbstractImageUploadController.ImageResponse insertImage(
        long contentLength,
        String contentType,
        InputStream inputStream
    );
}
