package com.github.nkonev.blog.repository.jdbc;

import com.github.nkonev.blog.entity.jdbc.Post;
import org.springframework.data.jdbc.repository.query.Modifying;
import org.springframework.data.jdbc.repository.query.Query;
import org.springframework.data.repository.CrudRepository;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import org.springframework.validation.annotation.Validated;
import java.util.Collection;
import java.util.List;

@Validated
@Repository
public interface PostRepository extends CrudRepository<Post, Long> {

    @Query("select p.* from posts.post p where p.owner_id = :ownerId order by id limit :limit offset :offset")
    List<Post> findMyPosts(@Param("limit")long limit, @Param("offset")long offset, @Param("ownerId")Long ownerId);

    @Query("select p.id from posts.post p")
    Collection<Long> findPostIds();

    @Modifying
    @Query(value = "UPDATE posts.post SET owner_id = :toUserId WHERE owner_id = :fromUserId")
    void moveToAnotherUser(@Param("fromUserId") long fromUserId, @Param("toUserId") long toUserId);
}
