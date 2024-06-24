package name.nkonev.aaa.repository.spring.jdbc;

import name.nkonev.aaa.entity.jdbc.UserAccount;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.DataClassRowMapper;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;
import org.springframework.stereotype.Repository;
import org.springframework.util.StringUtils;

import java.util.List;
import java.util.Map;

@Repository
public class UserListViewRepository {

    private static final Logger LOGGER = LoggerFactory.getLogger(UserListViewRepository.class);

    @Autowired
    private NamedParameterJdbcTemplate jdbcTemplate;

    private final RowMapper<UserAccount> rowMapper = DataClassRowMapper.newInstance(UserAccount.class);

    // copy-paste from chat/db/message.go::getMessagesCommon
    public List<UserAccount> getUsers(int limit, long startingFromItemId, boolean reverse, boolean hasHash, String searchString) {
        List<UserAccount> list;
        if (hasHash) {
            // has hash means that frontend's page has message hash
            // it means we need to calculate page/2 to the top and to the bottom
            // to respond page containing from two halves
            var leftLimit = limit / 2;
            var rightLimit = limit / 2;

            if (leftLimit == 0) {
                leftLimit = 1;
            }
            if (rightLimit == 0) {
                rightLimit = 1;
            }

            Long leftMessageId, rightMessageId;
            var searchStringPercents = "";
            if (StringUtils.hasLength(searchString)) {
                searchStringPercents = "%" + searchString + "%";
            }

            if (StringUtils.hasLength(searchString)) {
                leftMessageId = jdbcTemplate.queryForObject("""
                        SELECT MIN(inn.id) FROM (SELECT u.id FROM user_account u WHERE u.id > 0 AND u.id <= :startingFromItemId AND u.username ILIKE :searchStringPercents ORDER BY u.id DESC LIMIT :leftLimit) inn
                    """,
                    Map.of(
                        "startingFromItemId", startingFromItemId,
                        "leftLimit", leftLimit,
                        "searchStringPercents", searchStringPercents
                    ),
                    Long.class
                );
            } else {
                leftMessageId = jdbcTemplate.queryForObject("""
                        SELECT MIN(inn.id) FROM (SELECT u.id FROM user_account u WHERE u.id > 0 AND u.id <= :startingFromItemId ORDER BY u.id DESC LIMIT :leftLimit) inn
                    """,
                    Map.of(
                        "startingFromItemId", startingFromItemId,
                        "leftLimit", leftLimit
                    ),
                    Long.class
                );
            }

            if (StringUtils.hasLength(searchString)) {
                rightMessageId = jdbcTemplate.queryForObject("""
                        SELECT MAX(inn.id) + 1 FROM (SELECT u.id FROM user_account u WHERE u.id > 0 AND u.id >= :startingFromItemId AND u.username ILIKE :searchStringPercents ORDER BY u.id ASC LIMIT :rightLimit) inn
                    """,
                    Map.of(
                        "startingFromItemId", startingFromItemId,
                        "rightLimit", rightLimit,
                        "searchStringPercents", searchStringPercents
                    ),
                    Long.class
                );
            } else {
                rightMessageId = jdbcTemplate.queryForObject("""
                        SELECT MAX(inn.id) + 1 FROM (SELECT u.id FROM user_account u WHERE u.id > 0 AND u.id >= :startingFromItemId ORDER BY u.id ASC LIMIT :rightLimit) inn
                    """,
                    Map.of(
                        "startingFromItemId", startingFromItemId,
                        "rightLimit", rightLimit
                    ),
                    Long.class
                );
            }

            if (leftMessageId == null || rightMessageId == null) {
                LOGGER.info("Got leftMessageId={}, rightMessageId={} for startingFromItemId={}, reverse={}, searchString={}, fallback to simple", leftMessageId, rightMessageId, startingFromItemId, reverse, searchString);
                list = getUsersSimple(limit, 0, reverse, searchString);
            } else {

                var order = "asc";
                if (reverse) {
                    order = "desc";
                }

                if (StringUtils.hasLength(searchString)) {
                    list = jdbcTemplate.query("""
                                SELECT u.* FROM user_account u
                                WHERE
                                u.id > 0 AND
                                u.id >= :leftMessageId
                                AND u.id <= :rightMessageId
                                AND u.username ILIKE :searchStringPercents
                                ORDER BY u.id %s
                                LIMIT :limit
                            """.formatted(order),
                        Map.of(
                            "limit", limit,
                            "leftMessageId", leftMessageId,
                            "rightMessageId", rightMessageId,
                            "searchStringPercents", searchStringPercents
                        ),
                        rowMapper
                    );
                } else {
                    list = jdbcTemplate.query(
                        """
                                SELECT u.* FROM user_account u
                                WHERE
                                u.id > 0 AND
                                u.id >= :leftMessageId
                                AND u.id <= :rightMessageId
                                ORDER BY u.id %s
                                LIMIT :limit
                            """.formatted(order),
                        Map.of(
                            "limit", limit,
                            "leftMessageId", leftMessageId,
                            "rightMessageId", rightMessageId
                        ),
                        rowMapper
                    );
                }
            }
        } else {
            // otherwise, startingFromItemId is used as the top or the bottom limit of the portion
            list = getUsersSimple(limit, startingFromItemId, reverse, searchString);
        }

        return list;
    }

    private List<UserAccount> getUsersSimple(int limit, long startingFromItemId, boolean reverse, String searchString) {
        List<UserAccount> list;
        var order = "asc";
        var nonEquality = "u.id > :startingFromItemId";
        if (reverse) {
            order = "desc";
            nonEquality = "u.id < :startingFromItemId";
        }
        if (StringUtils.hasLength(searchString)) {
            var searchStringPercents = "%" + searchString + "%";
            list = jdbcTemplate.query("""
                    SELECT u.* FROM user_account u
                    WHERE
                    u.id > 0 AND
                    %s
                    AND u.username ILIKE :searchStringPercents
                    ORDER BY u.id %s
                    LIMIT :limit
                    """.formatted(nonEquality, order),
                Map.of(
                    "limit", limit,
                    "startingFromItemId", startingFromItemId,
                    "searchStringPercents", searchStringPercents
                ),
                rowMapper
            );
        } else {
            list = jdbcTemplate.query("""
                    SELECT u.* FROM user_account u
                    WHERE
                    u.id > 0 AND
                    %s
                    ORDER BY u.id %s
                    LIMIT :limit
                    """.formatted(nonEquality, order),
                Map.of(
                    "limit", limit,
                    "startingFromItemId", startingFromItemId
                ),
                rowMapper
            );
        }
        return list;
    }
}
