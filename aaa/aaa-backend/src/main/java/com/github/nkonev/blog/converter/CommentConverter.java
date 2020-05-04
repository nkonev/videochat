package com.github.nkonev.blog.converter;

import com.github.nkonev.blog.dto.CommentDTO;
import com.github.nkonev.blog.dto.CommentDTOExtended;
import com.github.nkonev.blog.dto.CommentDTOWithAuthorization;
import com.github.nkonev.blog.dto.UserAccountDetailsDTO;
import com.github.nkonev.blog.entity.jdbc.Comment;
import com.github.nkonev.blog.exception.BadRequestException;
import com.github.nkonev.blog.security.BlogSecurityService;
import com.github.nkonev.blog.security.permissions.CommentPermissions;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.util.Assert;
import org.springframework.util.StringUtils;

@Service
public class CommentConverter {

    @Autowired
    private BlogSecurityService blogSecurityService;

    @Autowired
    private UserAccountConverter userAccountConverter;

    public Comment convertFromDto(CommentDTO commentDTO, long postId, Comment forUpdate) {
        Assert.notNull(commentDTO, "commentDTO can't be null");
        checkLength(commentDTO.getText());
        if (forUpdate == null) {
            forUpdate = new Comment();
            forUpdate.setPostId(postId);
        }
        forUpdate.setText(commentDTO.getText());
        return forUpdate;
    }

    private void checkLength(String comment) {
        String trimmed = StringUtils.trimWhitespace(comment);
        final int MIN_COMMENT_LENGTH = 1;
        if (trimmed == null || trimmed.length() < MIN_COMMENT_LENGTH) {
            throw new BadRequestException("comment too short, must be longer than " + MIN_COMMENT_LENGTH);
        }
    }

    public CommentDTOWithAuthorization convertToDto(Comment comment, UserAccountDetailsDTO userAccount) {
        Assert.notNull(comment, "comment can't be null");

        return new CommentDTOWithAuthorization(
                comment.getId(),
                comment.getText(),
                userAccountConverter.convertToOwnerDTO(comment.getOwnerId()),
                blogSecurityService.hasCommentPermission(comment, userAccount, CommentPermissions.EDIT),
                blogSecurityService.hasCommentPermission(comment, userAccount, CommentPermissions.DELETE),
                comment.getCreateDateTime(),
                comment.getEditDateTime()
        );
    }

    public CommentDTO convertToDto(Comment comment) {
        Assert.notNull(comment, "comment can't be null");

        return new CommentDTO(
                comment.getId(),
                comment.getText(),
                comment.getCreateDateTime(),
                comment.getEditDateTime()
        );

    }

    public CommentDTOExtended convertToDtoExtended(Comment comment, UserAccountDetailsDTO userAccount, long countInPost) {
        return new CommentDTOExtended(
                comment.getId(),
                comment.getText(),
                userAccountConverter.convertToOwnerDTO(comment.getOwnerId()),
                blogSecurityService.hasCommentPermission(comment, userAccount, CommentPermissions.EDIT),
                blogSecurityService.hasCommentPermission(comment, userAccount, CommentPermissions.DELETE),
                countInPost,
                comment.getCreateDateTime(),
                comment.getEditDateTime()
        );
    }
}
