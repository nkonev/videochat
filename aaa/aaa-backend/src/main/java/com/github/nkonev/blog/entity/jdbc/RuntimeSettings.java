package com.github.nkonev.blog.entity.jdbc;

import com.github.nkonev.blog.Constants;
import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Table;

/**
 * Insert data only from migration
 */
@Table(Constants.Schemas.SETTINGS + ".runtime_settings")
public class RuntimeSettings {
    @Id
    private String key;

    private String value;

    public RuntimeSettings() {
    }

    public String getKey() {
        return key;
    }

    public void setKey(String key) {
        this.key = key;
    }

    public String getValue() {
        return value;
    }

    public void setValue(String value) {
        this.value = value;
    }
}
