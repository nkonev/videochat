package name.nkonev.aaa.config;

import name.nkonev.aaa.dto.ExternalPermission;
import name.nkonev.aaa.dto.UserRole;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import org.postgresql.jdbc.PgArray;
import org.springframework.boot.persistence.autoconfigure.EntityScan;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.convert.converter.Converter;
import org.springframework.data.jdbc.core.convert.JdbcConverter;
import org.springframework.data.jdbc.core.convert.JdbcCustomConversions;
import org.springframework.data.jdbc.repository.config.EnableJdbcRepositories;
import org.springframework.jdbc.core.DataClassRowMapper;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.transaction.annotation.EnableTransactionManagement;

import java.sql.SQLException;
import java.util.Arrays;
import java.util.List;

@Configuration
@EntityScan(basePackages = "name.nkonev.aaa.entity.jdbc")
@EnableJdbcRepositories(basePackages = "name.nkonev.aaa.repository.jdbc")
@EnableTransactionManagement
public class DbConfig {

    // sent to JdbcConverter jdbcConverter in AbstractJdbcConfiguration what creates MappingJdbcConverter
    @Bean
    public JdbcCustomConversions jdbcCustomConversions() {
        return new JdbcCustomConversions(List.of(
            new UserRoleArrayReadingConverter(),
            new UserPermissionArrayReadingConverter()
        ));
    }

    @Bean
    public RowMapper<UserAccount> userAccountRowMapper(JdbcConverter jdbcConverter) {
        var mapper = DataClassRowMapper.newInstance(UserAccount.class);
        mapper.setConversionService(jdbcConverter.getConversionService());
        return mapper;
    }
}

class UserRoleArrayReadingConverter implements Converter<PgArray, UserRole[]> {
    @Override
    public UserRole[] convert(PgArray pgObject) {
        try {
            String[] source = (String[]) pgObject.getArray();
            return Arrays.stream(source).map(UserRole::valueOf).toArray(UserRole[]::new);
        } catch (SQLException e) {
            throw new RuntimeException(e);
        }
    }
}

class UserPermissionArrayReadingConverter implements Converter<PgArray, ExternalPermission[]> {
    @Override
    public ExternalPermission[] convert(PgArray pgObject) {
        try {
            String[] source = (String[]) pgObject.getArray();
            return Arrays.stream(source).map(ExternalPermission::valueOf).toArray(ExternalPermission[]::new);
        } catch (SQLException e) {
            throw new RuntimeException(e);
        }
    }
}

// see CREATE CAST in V1__init.sql
