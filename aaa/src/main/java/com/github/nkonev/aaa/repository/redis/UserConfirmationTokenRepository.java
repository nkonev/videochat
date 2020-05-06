package com.github.nkonev.aaa.repository.redis;

import com.github.nkonev.aaa.entity.redis.UserConfirmationToken;
import org.springframework.data.repository.CrudRepository;
import org.springframework.stereotype.Repository;

@Repository
public interface UserConfirmationTokenRepository extends CrudRepository<UserConfirmationToken, String> {
}
