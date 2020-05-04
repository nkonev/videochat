package com.github.nkonev.blog.entity.elasticsearch;

import org.springframework.data.annotation.Id;
import org.springframework.data.elasticsearch.annotations.Document;
import static com.github.nkonev.blog.entity.elasticsearch.IndexPost.INDEX;

@Document(indexName = INDEX, createIndex = false)
public class IndexPost {

    public static final String INDEX = "post";

    public static final String FIELD_ID = "id";
    public static final String FIELD_TEXT = "text";
    public static final String FIELD_TITLE = "title";
    public static final String FIELD_TEXT_STD = "text.std";
    public static final String FIELD_TITLE_STD = "title.std";
    public static final String FIELD_DRAFT = "draft";
    public static final String FIELD_OWNER_ID = "ownerId";

    @Id
    private Long id;

    private String title;

    private String text;

    private boolean draft;

    private long ownerId;

    public IndexPost() { }

    public IndexPost(Long id, String title, String text, boolean draft, long ownerId) {
        this.id = id;
        this.title = title;
        this.text = text;
        this.ownerId = ownerId;
        this.draft = draft;
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

    public boolean isDraft() {
        return draft;
    }

    public void setDraft(boolean draft) {
        this.draft = draft;
    }

    public long getOwnerId() {
        return ownerId;
    }

    public void setOwnerId(long ownerId) {
        this.ownerId = ownerId;
    }
}
