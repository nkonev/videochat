package name.nkonev.aaa.config;

import org.springframework.boot.autoconfigure.domain.EntityScan;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.jdbc.repository.config.AbstractJdbcConfiguration;
import org.springframework.data.jdbc.repository.config.EnableJdbcRepositories;
import org.springframework.transaction.annotation.EnableTransactionManagement;

@Configuration
@EntityScan(basePackages = "name.nkonev.aaa.entity.jdbc")
@EnableJdbcRepositories(basePackages = "name.nkonev.aaa.repository.jdbc")
@EnableTransactionManagement
public class DbConfig extends AbstractJdbcConfiguration {

}
