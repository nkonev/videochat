package com.github.nkonev.blog.entity.jdbc;

import com.github.nkonev.blog.Constants;
import org.springframework.data.annotation.*;
import org.springframework.data.relational.core.mapping.Table;

import java.time.LocalDateTime;

/**
 * This entity "Post" don't have comments because there isn't always need to get Post with Collection<Comment>
 */
@Table(Constants.Schemas.POSTS+ ".post")
public class Post {
    @Id
    private Long id;

    private String title;
    private String text;
    private String titleImg;

    private Long ownerId;

    private LocalDateTime createDateTime;

    private LocalDateTime editDateTime;

    private boolean draft;

    public Post() { }

    public Post(Long id, String title, String text, String titleImg) {
        this.id = id;
        this.title = title;
        this.text = text;
        this.titleImg = titleImg;
    }

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public String getTitle() {
        return title;
    }

    public void setTitle(String title) {
        this.title = title;
    }

    public String getText() {
        return text;
    }

    public void setText(String text) {
        this.text = text;
    }

    public String getTitleImg() {
        return titleImg;
    }

    public void setTitleImg(String titleImg) {
        this.titleImg = titleImg;
    }

    public LocalDateTime getCreateDateTime() {
        return createDateTime;
    }

    public void setCreateDateTime(LocalDateTime createDateTime) {
        this.createDateTime = createDateTime;
    }

    public LocalDateTime getEditDateTime() {
        return editDateTime;
    }

    public void setEditDateTime(LocalDateTime editDateTime) {
        this.editDateTime = editDateTime;
    }

    public boolean isDraft() {
        return draft;
    }

    public void setDraft(boolean draft) {
        this.draft = draft;
    }

    public Long getOwnerId() {
        return ownerId;
    }

    public void setOwnerId(Long ownerId) {
        this.ownerId = ownerId;
    }
}
