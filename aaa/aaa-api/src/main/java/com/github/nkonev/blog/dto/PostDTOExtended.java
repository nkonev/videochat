package com.github.nkonev.blog.dto;

import java.time.LocalDateTime;

public class PostDTOExtended extends PostDTOWithAuthorization {
    private PostPreview left;
    private PostPreview right;

    public PostDTOExtended() { }

    public PostDTOExtended(
            long id,
            String title,
            String text,
            String titleImg,
            OwnerDTO userAccountDTO,
            boolean canEdit,
            boolean canDelete,
            PostPreview left,
            PostPreview right,
            LocalDateTime createDateTime,
            LocalDateTime editDateTime,
            boolean draft
    ) {
        super(id, title, text, titleImg, userAccountDTO, canEdit, canDelete, createDateTime, editDateTime, draft);
        this.left = left;
        this.right = right;
    }

    public PostPreview getLeft() {
        return left;
    }

    public void setLeft(PostPreview left) {
        this.left = left;
    }

    public PostPreview getRight() {
        return right;
    }

    public void setRight(PostPreview right) {
        this.right = right;
    }
}
