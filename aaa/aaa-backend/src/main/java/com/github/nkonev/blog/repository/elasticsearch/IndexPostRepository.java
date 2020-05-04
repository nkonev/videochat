package com.github.nkonev.blog.repository.elasticsearch;

import com.github.nkonev.blog.entity.elasticsearch.IndexPost;
import org.springframework.data.elasticsearch.repository.ElasticsearchRepository;
import org.springframework.stereotype.Repository;

@Repository
public interface IndexPostRepository extends ElasticsearchRepository<IndexPost, Long> {

}
