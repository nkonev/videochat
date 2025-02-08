package name.nkonev.aaa.repository.spring.jdbc;

import name.nkonev.aaa.entity.jdbc.UserAccount;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
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

    @Autowired
    private RowMapper<UserAccount> userAccountRowMapper;

    private record MinMax(Long leftId, Long rightId) { }

    private final RowMapper<MinMax> mmRowMapper = (rs, rowNum) -> {
        Long min = (Long) rs.getObject("minid");
        Long max = (Long) rs.getObject("maxid");
        return new MinMax(min, max);
    };

    private static final String USERNAME_SEARCH = """ 
        (
            (u.username ilike :searchStringPercents)
            or (cyrillic_transliterate(u.username) ilike '%' || cyrillic_transliterate(:searchStringPercents) || '%')
        )
    """;

    private long getSafeDefaultUserId(boolean reverse) {
        if (reverse) {
            return 0;
        } else {
            return Long.MAX_VALUE;
        }
    }

    // copy-paste from chat/db/message.go::getMessagesCommon
    public List<UserAccount> getUsers(int limit, Long startingFromItemId0, boolean includeStartingFrom, boolean reverse, String searchString) {

        final long startingFromItemId;
        if (startingFromItemId0 == null) {
            startingFromItemId = getSafeDefaultUserId(reverse);
        } else {
            startingFromItemId = startingFromItemId0;
        }

        // startingFromItemId is used as the top or the bottom limit of the portion
        var list = getUsersSimple(limit, startingFromItemId, includeStartingFrom, reverse, searchString);

        return list;
    }

    // implements keyset pagination
    private List<UserAccount> getUsersSimple(int limit, long startingFromItemId, boolean includeStartingFrom, boolean reverse, String searchString) {
        List<UserAccount> list;
        var order = "";
        var nonEquality = "";
        if (reverse) {
            order = "asc";
            var s = "";
            if (includeStartingFrom) {
                s = ">=";
            } else {
                s = ">";
            }
            nonEquality = "u.id " + s + " :startingFromItemId";
        } else {
            order = "desc";
            var s = "";
            if (includeStartingFrom) {
                s = "<=";
            } else {
                s = "<";
            }
            nonEquality = "u.id " + s + " :startingFromItemId";
        }
        if (StringUtils.hasLength(searchString)) {
            var searchStringPercents = "%" + searchString + "%";
            list = jdbcTemplate.query("""
                    SELECT u.* FROM user_account u
                    WHERE
                    u.id > 0 AND
                    %s
                    AND %s
                    ORDER BY u.id %s
                    LIMIT :limit
                    """.formatted(nonEquality, USERNAME_SEARCH, order),
                Map.of(
                    "limit", limit,
                    "startingFromItemId", startingFromItemId,
                    "searchStringPercents", searchStringPercents
                ),
                userAccountRowMapper
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
                userAccountRowMapper
            );
        }
        return list;
    }

    private static final String PREFIX_COUNT = "select count(u.*) from user_account u where u.id > 0 ";

    private static final String PREFIX = "select u.* from user_account u where u.id > 0 ";

    private static final String PAGINATION_SUFFIX = " order by u.id limit :limit offset :offset";

    public List<UserAccount> findPage(int pageSize, int offset) {
        return jdbcTemplate.query(
            PREFIX + PAGINATION_SUFFIX,
            Map.of(
                "limit", pageSize,
                "offset", offset
            ),
            userAccountRowMapper);
    }

    private static final String AND_USERNAME_ILIKE = " and " + USERNAME_SEARCH;

    public List<UserAccount> findByUsernameContainsIgnoreCase(int pageSize, long offset, String searchString) {
        return jdbcTemplate.query(
            PREFIX + AND_USERNAME_ILIKE + PAGINATION_SUFFIX,
            Map.of(
                "searchStringPercents", searchString,
                "limit", pageSize,
                "offset", offset
            ),
            userAccountRowMapper);
    }

    public long findByUsernameContainsIgnoreCaseCount(String searchString) {
        return jdbcTemplate.queryForObject(
            PREFIX_COUNT + AND_USERNAME_ILIKE,
            Map.of(
                "searchStringPercents", searchString
            ),
            long.class);
    }

    private static final String AND_USERNAME_ILIKE_AND_ID_IN = " and " + USERNAME_SEARCH + " and u.id in (:userIds) ";

    public List<UserAccount> findByUsernameContainsIgnoreCaseAndIdIn(int pageSize, long offset, String searchString, List<Long> includingUserIds) {
        return jdbcTemplate.query(
            PREFIX + AND_USERNAME_ILIKE_AND_ID_IN + PAGINATION_SUFFIX,
            Map.of(
                "searchStringPercents", searchString,
                "userIds", includingUserIds,
                "limit", pageSize,
                "offset", offset
            ),
            userAccountRowMapper);
    }

    public long findByUsernameContainsIgnoreCaseAndIdInCount(String searchString, List<Long> includingUserIds) {
        return jdbcTemplate.queryForObject(
            PREFIX_COUNT + AND_USERNAME_ILIKE_AND_ID_IN,
            Map.of(
                "searchStringPercents", searchString,
                "userIds", includingUserIds
            ),
            long.class);
    }

    private static final String AND_USERNAME_ILIKE_AND_ID_NOT_IN = " and " + USERNAME_SEARCH + " and u.id not in (:userIds) ";

    public List<UserAccount> findByUsernameContainsIgnoreCaseAndIdNotIn(int pageSize, long offset, String searchString, List<Long> excludingUserIds) {
        return jdbcTemplate.query(
            PREFIX + AND_USERNAME_ILIKE_AND_ID_NOT_IN + PAGINATION_SUFFIX,
            Map.of(
                "searchStringPercents", searchString,
                "userIds", excludingUserIds,
                "limit", pageSize,
                "offset", offset
            ),
            userAccountRowMapper);
    }

    public long findByUsernameContainsIgnoreCaseAndIdNotInCount(String searchString, List<Long> excludingUserIds) {
        return jdbcTemplate.queryForObject(
            PREFIX_COUNT + AND_USERNAME_ILIKE_AND_ID_NOT_IN,
            Map.of(
                "searchStringPercents", searchString,
                "userIds", excludingUserIds
            ),
            long.class);
    }
}
