package name.nkonev.aaa.repository.redis;

import name.nkonev.aaa.entity.redis.ChangeEmailConfirmationToken;
import org.springframework.data.repository.CrudRepository;
import org.springframework.stereotype.Repository;

import java.util.UUID;

@Repository
public interface ChangeEmailConfirmationTokenRepository extends CrudRepository<ChangeEmailConfirmationToken, UUID> {

}
