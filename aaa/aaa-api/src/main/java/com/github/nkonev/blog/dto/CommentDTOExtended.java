package com.github.nkonev.blog.dto;

import java.time.LocalDateTime;

public class CommentDTOExtended extends CommentDTOWithAuthorization {
    private long commentsInPost;

    public CommentDTOExtended() { }

    public CommentDTOExtended(
            long id,
            String text,
            OwnerDTO owner,
            boolean canEdit,
            boolean canDelete,
            long commentsInPost,
            LocalDateTime createDateTime,
            LocalDateTime editDateTime
    ) {
        super(id, text, owner, canEdit, canDelete, createDateTime, editDateTime);
        this.commentsInPost = commentsInPost;
    }

    public long getCommentsInPost() {
        return commentsInPost;
    }

    public void setCommentsInPost(long commentsInPost) {
        this.commentsInPost = commentsInPost;
    }
}
