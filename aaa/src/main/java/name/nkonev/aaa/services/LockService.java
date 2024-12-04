package name.nkonev.aaa.services;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.UUID;

@Service
public class LockService {

    @Autowired
    private StringRedisTemplate stringRedisTemplate;

    public LockInstance lock(String lockKey, Duration timeout) {
        boolean wasSet = stringRedisTemplate.opsForValue().setIfAbsent(lockKey, UUID.randomUUID().toString(), timeout);

        return new LockInstance(wasSet, lockKey, stringRedisTemplate);
    }

    public static class LockInstance implements AutoCloseable {

        private final boolean wasSet;

        private final String lockKey;

        private final StringRedisTemplate stringRedisTemplate;

        public LockInstance(boolean wasSet, String lockKey, StringRedisTemplate stringRedisTemplate) {
            this.wasSet = wasSet;
            this.lockKey = lockKey;
            this.stringRedisTemplate = stringRedisTemplate;
        }

        @Override
        public void close() {
            stringRedisTemplate.delete(lockKey);
        }

        public boolean isWasSet() {
            return wasSet;
        }
    }
}
