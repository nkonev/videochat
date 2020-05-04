package com.github.nkonev.blog.repository.jdbc;

import com.github.nkonev.blog.entity.jdbc.Comment;
import org.springframework.data.jdbc.repository.query.Modifying;
import org.springframework.data.jdbc.repository.query.Query;
import org.springframework.data.repository.CrudRepository;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import java.util.List;

@Repository
public interface CommentRepository extends CrudRepository<Comment, Long> {

    @Query("select * from posts.comment c where c.post_id = :postId order by id asc limit :limit offset :offset")
    List<Comment> findCommentByPostIdOrderByIdAsc(@Param("limit")long limit, @Param("offset")long offset, @Param("postId")long postId);

    @Query("delete from posts.comment c where c.post_id = :postId")
    @Modifying
    void deleteByPostId(@Param("postId")long postId);

    @Query("select count(id) from posts.comment c where c.post_id = :postId")
    long countByPostId(@Param("postId")long postId);

    @Modifying
    @Query(value = "UPDATE posts.comment SET owner_id = :toUserId WHERE owner_id = :fromUserId")
    void moveToAnotherUser(@Param("fromUserId") long fromUserId, @Param("toUserId") long toUserId);
}
