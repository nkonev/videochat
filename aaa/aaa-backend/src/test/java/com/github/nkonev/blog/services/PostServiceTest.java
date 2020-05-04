package com.github.nkonev.blog.services;

import com.github.nkonev.blog.AbstractUtTestRunner;
import com.github.nkonev.blog.entity.elasticsearch.IndexPost;
import com.github.nkonev.blog.repository.elasticsearch.IndexPostRepository;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;

public class PostServiceTest extends AbstractUtTestRunner {

    @Autowired
    private IndexPostRepository indexPostRepository;

    @Autowired
    private PostService postService;

    @Test
    public void testCleanOldElasticsearchGarbage(){
        IndexPost post = new IndexPost();
        post.setId(20000L);
        post.setText("trash text");
        post.setTitle("trash title");

        post=indexPostRepository.save(post);

        postService.refreshFulltextIndex(true);

        Assertions.assertFalse(indexPostRepository.findById(post.getId()).isPresent());
    }
}

