package com.github.nkonev.blog.controllers;

import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.converter.CommentConverter;
import com.github.nkonev.blog.dto.*;
import com.github.nkonev.blog.entity.jdbc.Comment;
import com.github.nkonev.blog.entity.jdbc.UserAccount;
import com.github.nkonev.blog.exception.BadRequestException;
import com.github.nkonev.blog.repository.jdbc.CommentRepository;
import com.github.nkonev.blog.repository.jdbc.UserAccountRepository;
import com.github.nkonev.blog.utils.PageUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.domain.PageRequest;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.Assert;
import org.springframework.web.bind.annotation.*;
import javax.validation.constraints.NotNull;
import java.util.Collection;
import java.util.List;
import java.util.stream.Collectors;

import static com.github.nkonev.blog.utils.TimeUtil.getNowUTC;

@Transactional
@RestController
public class CommentController {

    @Autowired
    private CommentRepository commentRepository;

    @Autowired
    private CommentConverter commentConverter;

    @Autowired
    private UserAccountRepository userAccountRepository;

    /**
     * List post comments
     * @param userAccount
     * @param postId
     * @return
     */
    @GetMapping(Constants.Urls.API+ Constants.Urls.POST+ Constants.Urls.POST_ID + Constants.Urls.COMMENT)
    public Wrapper<CommentDTOWithAuthorization> getPostComments(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount, // nullable
            @PathVariable(Constants.PathVariables.POST_ID) long postId,
            @RequestParam(value = "page", required=false, defaultValue = "0") int page,
            @RequestParam(value = "size", required=false, defaultValue = "0") int size
            ) {

        PageRequest springDataPage = PageRequest.of(PageUtils.fixPage(page), PageUtils.fixSize(size));

        long count = commentRepository.countByPostId(postId);
        List<Comment> comments = commentRepository.findCommentByPostIdOrderByIdAsc(springDataPage.getPageSize(), springDataPage.getOffset(), postId);

        Collection<CommentDTOWithAuthorization> commentsCollection =  comments
                .stream()
                .map(comment -> commentConverter.convertToDto(comment, userAccount))
                .collect(Collectors.toList());

        return new Wrapper<CommentDTOWithAuthorization>(commentsCollection, count);
    }

    @PreAuthorize("@blogSecurityService.hasCommentPermission(#userAccount, T(com.github.nkonev.blog.security.permissions.CommentPermissions).CREATE)")
    @PostMapping(Constants.Urls.API+ Constants.Urls.POST+ Constants.Urls.POST_ID + Constants.Urls.COMMENT)
    public CommentDTOExtended addComment(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount, // nullable
            @PathVariable(Constants.PathVariables.POST_ID) long postId,
            @RequestBody @NotNull CommentDTO commentDTO
    ){
        Assert.notNull(userAccount, "UserAccountDetailsDTO can't be null");
        if (commentDTO.getId()!=0){
            throw new BadRequestException("id cannot be set");
        }

        long count = commentRepository.countByPostId(postId);
        Comment comment = commentConverter.convertFromDto(commentDTO, postId, null);
        comment.setCreateDateTime(getNowUTC());

        UserAccount ua = userAccountRepository.findById(userAccount.getId()).orElseThrow(()->new IllegalArgumentException("User account not found")); // Hibernate caches it
        comment.setOwnerId(ua.getId());
        Comment saved = commentRepository.save(comment);

        return commentConverter.convertToDtoExtended(saved, userAccount, count);
    }

    @PreAuthorize("@blogSecurityService.hasCommentPermission(#commentDTO, #userAccount, T(com.github.nkonev.blog.security.permissions.CommentPermissions).EDIT)")
    @PutMapping(Constants.Urls.API+ Constants.Urls.POST+ Constants.Urls.POST_ID + Constants.Urls.COMMENT)
    public CommentDTOExtended updateComment (
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount, // nullable
            @PathVariable(Constants.PathVariables.POST_ID) long postId,
            @RequestBody @NotNull CommentDTO commentDTO
    ){
        Assert.notNull(userAccount, "UserAccountDetailsDTO can't be null");

        long count = commentRepository.countByPostId(postId);

        Comment found = commentRepository.findById(commentDTO.getId()).orElseThrow(()-> new IllegalArgumentException("Comment with id " + commentDTO.getId() + " not found"));

        Comment updatedEntity = commentConverter.convertFromDto(commentDTO, 0, found);
        updatedEntity.setEditDateTime(getNowUTC());
        Comment saved = commentRepository.save(updatedEntity);

        return commentConverter.convertToDtoExtended(saved, userAccount, count);
    }

    @PreAuthorize("@blogSecurityService.hasCommentPermission(#commentId, #userAccount, T(com.github.nkonev.blog.security.permissions.CommentPermissions).DELETE)")
    @DeleteMapping(Constants.Urls.API+ Constants.Urls.POST+ Constants.Urls.POST_ID + Constants.Urls.COMMENT+ Constants.Urls.COMMENT_ID)
    public long deleteComment(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount, // null if not authenticated
            @PathVariable(Constants.PathVariables.POST_ID) long postId,
            @PathVariable(Constants.PathVariables.COMMENT_ID) long commentId
    ) {
        Assert.notNull(userAccount, "UserAccountDetailsDTO can't be null");
        commentRepository.deleteById(commentId);

        return commentRepository.countByPostId(postId);
    }
}
