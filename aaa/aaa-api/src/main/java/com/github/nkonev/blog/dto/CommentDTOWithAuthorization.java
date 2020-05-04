package com.github.nkonev.blog.dto;

import java.time.LocalDateTime;

public class CommentDTOWithAuthorization extends CommentDTO {
    private OwnerDTO owner;
    private boolean canEdit;
    private boolean canDelete;

    public CommentDTOWithAuthorization() { }

    public CommentDTOWithAuthorization(
            long id,
            String text,
            OwnerDTO owner,
            boolean canEdit,
            boolean canDelete,
            LocalDateTime createDateTime,
            LocalDateTime editDateTime
    ) {
        super(id, text, createDateTime, editDateTime);
        this.owner = owner;
        this.canEdit = canEdit;
        this.canDelete = canDelete;
    }

    public OwnerDTO getOwner() {
        return owner;
    }

    public void setOwner(OwnerDTO owner) {
        this.owner = owner;
    }

    public boolean isCanEdit() {
        return canEdit;
    }

    public void setCanEdit(boolean canEdit) {
        this.canEdit = canEdit;
    }

    public boolean isCanDelete() {
        return canDelete;
    }

    public void setCanDelete(boolean canDelete) {
        this.canDelete = canDelete;
    }
}
