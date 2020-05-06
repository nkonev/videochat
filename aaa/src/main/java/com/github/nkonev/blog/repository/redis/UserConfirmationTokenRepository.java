package com.github.nkonev.blog.repository.redis;

import com.github.nkonev.blog.entity.redis.UserConfirmationToken;
import org.springframework.data.repository.CrudRepository;
import org.springframework.stereotype.Repository;

@Repository
public interface UserConfirmationTokenRepository extends CrudRepository<UserConfirmationToken, String> {
}
