package com.github.nkonev.aaa.repository.spring.jdbc;

import com.github.nkonev.aaa.entity.jdbc.UserAccount;
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

            long leftMessageId, rightMessageId;
            var searchStringPercents = "";
            if (StringUtils.hasLength(searchString)) {
                searchStringPercents = "%" + searchString + "%";
            }

            if (StringUtils.hasLength(searchString)) {
                leftMessageId = jdbcTemplate.queryForObject("""
                        SELECT MIN(inn.id) FROM (SELECT u.id FROM user_account u WHERE id <= :startingFromItemId AND u.username ILIKE :searchStringPercents ORDER BY id DESC LIMIT :leftLimit) inn
                    """,
                    Map.of(
                        "startingFromItemId", startingFromItemId,
                        "leftLimit", leftLimit,
                        "searchStringPercents", searchStringPercents
                    ),
                    long.class
                );
            } else {
                leftMessageId = jdbcTemplate.queryForObject("""
                        SELECT MIN(inn.id) FROM (SELECT u.id FROM user_account u WHERE id <= :startingFromItemId ORDER BY id DESC LIMIT :leftLimit) inn
                    """,
                    Map.of(
                        "startingFromItemId", startingFromItemId,
                        "leftLimit", leftLimit
                    ),
                    long.class
                );
            }

            if (StringUtils.hasLength(searchString)) {
                rightMessageId = jdbcTemplate.queryForObject("""
                        SELECT MAX(inn.id) + 1 FROM (SELECT u.id FROM user_account u WHERE id >= :startingFromItemId AND u.username ILIKE :searchStringPercents ORDER BY id ASC LIMIT :rightLimit) inn
                    """,
                    Map.of(
                        "startingFromItemId", startingFromItemId,
                        "rightLimit", rightLimit,
                        "searchStringPercents", searchStringPercents
                    ),
                    long.class
                );
            } else {
                rightMessageId = jdbcTemplate.queryForObject("""
                        SELECT MAX(inn.id) + 1 FROM (SELECT u.id FROM user_account u WHERE id >= :startingFromItemId ORDER BY id ASC LIMIT :rightLimit) inn
                    """,
                    Map.of(
                        "startingFromItemId", startingFromItemId,
                        "rightLimit", rightLimit
                    ),
                    long.class
                );
            }

            var order = "asc";
            if (reverse) {
                order = "desc";
            }

            if (StringUtils.hasLength(searchString)){
                list = jdbcTemplate.query("""
                        SELECT u.* FROM user_account u
                        WHERE
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
        } else {
            // otherwise, startingFromItemId is used as the top or the bottom limit of the portion
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
        }

        return list;
    }
}
