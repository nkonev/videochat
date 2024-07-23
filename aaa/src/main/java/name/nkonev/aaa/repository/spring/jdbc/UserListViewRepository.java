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

import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.List;
import java.util.Map;

@Repository
public class UserListViewRepository {

    private static final Logger LOGGER = LoggerFactory.getLogger(UserListViewRepository.class);

    @Autowired
    private NamedParameterJdbcTemplate jdbcTemplate;

    private final RowMapper<UserAccount> rowMapper = DataClassRowMapper.newInstance(UserAccount.class);

    private record MinMax(Long leftId, Long rightId) { }

    private final RowMapper<MinMax> mmRowMapper = (rs, rowNum) -> {
        long min = rs.getLong("minid");
        long max = rs.getLong("maxid");
        return new MinMax(min != 0 ? min : null, max != 0 ? max : null);
    };

    private static final String USERNAME_SEARCH = """ 
        (
            (u.username ilike :searchStringPercents)
            or (cyrillic_transliterate(u.username) ilike '%' || cyrillic_transliterate(:searchStringPercents) || '%')
        )
    """;

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

            Long leftItemId = null;
            Long rightItemId = null;
            var searchStringPercents = "";
            if (StringUtils.hasLength(searchString)) {
                searchStringPercents = "%" + searchString + "%";
            }

            if (StringUtils.hasLength(searchString)) {
                MinMax mm = jdbcTemplate.queryForObject(
                    """
                    select inner3.minid, inner3.maxid from (
                        select inner2.*, lag(id, :leftLimit, inner2.mmin) over() as minid, lead(id, :rightLimit, inner2.mmax) over() as maxid from (
                            select inn.*, id = :startingFromItemId as central_element from (
                                select id, row_number() over () as rn, (min(id) over ()) as mmin, (max(id) over ()) as mmax from user_account u where u.id > 0 AND %s order by id
                            ) inn
                        ) inner2
                    ) inner3 where central_element = true
                    """.formatted(USERNAME_SEARCH),
                    Map.of(
                        "startingFromItemId", startingFromItemId,
                        "leftLimit", leftLimit,
                        "rightLimit", rightLimit,
                        "searchStringPercents", searchStringPercents
                    ),
                    mmRowMapper
                );
                if (mm != null) {
                    leftItemId = mm.leftId();
                    rightItemId = mm.rightId();
                }
            } else {
                MinMax mm = jdbcTemplate.queryForObject(
                    """
                    select inner3.minid, inner3.maxid from (
                        select inner2.*, lag(id, :leftLimit, inner2.mmin) over() as minid, lead(id, :rightLimit, inner2.mmax) over() as maxid from (
                            select inn.*, id = :startingFromItemId as central_element from (
                                select id, row_number() over () as rn, (min(id) over ()) as mmin, (max(id) over ()) as mmax from user_account u where u.id > 0 order by id
                            ) inn
                        ) inner2
                    ) inner3 where central_element = true
                    """,
                    Map.of(
                        "startingFromItemId", startingFromItemId,
                        "leftLimit", leftLimit,
                        "rightLimit", rightLimit
                    ),
                    mmRowMapper
                );
                if (mm != null) {
                    leftItemId = mm.leftId();
                    rightItemId = mm.rightId();
                }
            }

            if (leftItemId == null || rightItemId == null) {
                LOGGER.info("Got leftItemId={}, rightItemId={} for startingFromItemId={}, reverse={}, searchString={}, fallback to simple", leftItemId, rightItemId, startingFromItemId, reverse, searchString);
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
                                AND %s
                                ORDER BY u.id %s
                                LIMIT :limit
                            """.formatted(USERNAME_SEARCH, order),
                        Map.of(
                            "limit", limit,
                            "leftMessageId", leftItemId,
                            "rightMessageId", rightItemId,
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
                            "leftMessageId", leftItemId,
                            "rightMessageId", rightItemId
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
                    AND %s
                    ORDER BY u.id %s
                    LIMIT :limit
                    """.formatted(nonEquality, USERNAME_SEARCH, order),
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
            rowMapper);
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
            rowMapper);
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
            rowMapper);
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
            rowMapper);
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
