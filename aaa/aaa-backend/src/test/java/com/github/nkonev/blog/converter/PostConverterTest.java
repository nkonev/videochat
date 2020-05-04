package com.github.nkonev.blog.converter;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

public class PostConverterTest {

    @Test
    public void testGetYouTubeId() {
        String youtubeVideoId = PostConverter.getYouTubeVideoId("https://www.youtube.com/embed/eoDsxos6xhM?showinfo=0");
        Assertions.assertEquals("eoDsxos6xhM", youtubeVideoId);
    }
}
