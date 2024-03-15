package com.github.nkonev.aaa.repository.redis;

import com.github.nkonev.aaa.entity.redis.ChangeEmailConfirmationToken;
import org.springframework.data.repository.CrudRepository;
import org.springframework.stereotype.Repository;

import java.util.UUID;

@Repository
public interface ChangeEmailConfirmationTokenRepository extends CrudRepository<ChangeEmailConfirmationToken, UUID> {

}
