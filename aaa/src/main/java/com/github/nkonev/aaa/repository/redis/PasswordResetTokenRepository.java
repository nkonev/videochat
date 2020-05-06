package com.github.nkonev.aaa.repository.redis;

import com.github.nkonev.aaa.entity.redis.PasswordResetToken;
import org.springframework.data.repository.CrudRepository;
import org.springframework.stereotype.Repository;
import java.util.UUID;

@Repository
public interface PasswordResetTokenRepository extends CrudRepository<PasswordResetToken, UUID> {

}
