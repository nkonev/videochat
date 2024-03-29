package com.github.nkonev.aaa.entity.jdbc;

import com.github.nkonev.aaa.dto.Language;
import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Table;

@Table("user_settings")
public record UserSettings(
    @Id Long id,

    String[] smileys,

    Language language
) {

}
