package com.github.nkonev.blog.security;

import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.dto.CommentDTO;
import com.github.nkonev.blog.dto.LockDTO;
import com.github.nkonev.blog.dto.PostDTO;
import com.github.nkonev.blog.dto.UserAccountDetailsDTO;
import com.github.nkonev.blog.entity.jdbc.Comment;
import com.github.nkonev.blog.entity.jdbc.Post;
import com.github.nkonev.blog.entity.jdbc.UserAccount;
import com.github.nkonev.blog.dto.UserRole;
import com.github.nkonev.blog.repository.jdbc.CommentRepository;
import com.github.nkonev.blog.repository.jdbc.PostRepository;
import com.github.nkonev.blog.repository.jdbc.UserAccountRepository;
import com.github.nkonev.blog.security.permissions.CommentPermissions;
import com.github.nkonev.blog.security.permissions.PostPermissions;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.access.hierarchicalroles.RoleHierarchy;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.stereotype.Service;
import org.springframework.util.Assert;

import java.util.Optional;

/**
 * Central entrypoint for access decisions
 */
@Service
public class BlogSecurityService {
    @Autowired
    private RoleHierarchy roleHierarchy;

    @Autowired
    private PostRepository postRepository;

    @Autowired
    private CommentRepository commentRepository;

    @Autowired
    private UserAccountRepository userAccountRepository;

    public boolean hasPostPermission(PostDTO dto, UserAccountDetailsDTO userAccount, PostPermissions permission) {
        Assert.notNull(dto, "PostDTO can't be null");
        return hasPostPermission(dto.getId(), userAccount, permission);
    }

    private Post getPostOrException(long id) {
        Post post = postRepository.findById(id).orElseThrow(()->new IllegalArgumentException("Post with id "+id+" not found"));
        return post;
    }

    public boolean hasPostPermission(long id, UserAccountDetailsDTO userAccount, PostPermissions permission) {
        Post post = getPostOrException(id);
        return hasPostPermission(post, userAccount, permission);
    }

    public boolean hasPostPermission(UserAccountDetailsDTO userAccount, PostPermissions permission) {
        return hasPostPermission((Post)null, userAccount, permission);
    }


    public boolean hasPostPermission(Post post, UserAccountDetailsDTO userAccount, PostPermissions permission) {
        if (userAccount == null) {return false;}

        if (permission == PostPermissions.CREATE) {
            return true;
        }

        if (permission == PostPermissions.READ_MY) {
            return true;
        }

        if (post == null) {
            return false;
        }

        if (roleHierarchy.getReachableGrantedAuthorities(userAccount.getAuthorities()).contains(new SimpleGrantedAuthority(UserRole.ROLE_MODERATOR.name()))){
            return true;
        }
        if (post.getOwnerId().equals(userAccount.getId()) && permission==PostPermissions.EDIT){
            return true;
        }
        return false;
    }





    public boolean hasCommentPermission(CommentDTO dto, UserAccountDetailsDTO userAccount, CommentPermissions permission) {
        Assert.notNull(dto, "CommentDTO can't be null");
        return hasCommentPermission(dto.getId(), userAccount, permission);
    }

    public boolean hasCommentPermission(long id, UserAccountDetailsDTO userAccount, CommentPermissions permission) {
        Comment comment = commentRepository.findById(id).orElseThrow(()->new IllegalArgumentException("Comment with id "+id+" not found"));
        return hasCommentPermission(comment, userAccount, permission);
    }

    public boolean hasCommentPermission(UserAccountDetailsDTO userAccount, CommentPermissions permission) {
        return hasCommentPermission((Comment)null, userAccount, permission);
    }

    public boolean hasCommentPermission(Comment comment, UserAccountDetailsDTO userAccount, CommentPermissions permission) {
        if (userAccount == null) {
            return false;
        }

        if (permission == CommentPermissions.CREATE) {
            return true;
        }

        if (comment == null) {
            return false;
        }

        if (roleHierarchy.getReachableGrantedAuthorities(userAccount.getAuthorities()).contains(new SimpleGrantedAuthority(UserRole.ROLE_MODERATOR.name()))){
            return true;
        }
        if (Long.valueOf(comment.getOwnerId()).equals(userAccount.getId())){
            return true;
        }

        return false;
    }

    public boolean hasSessionManagementPermission(UserAccountDetailsDTO userAccount) {
        if (userAccount==null){
            return false;
        }
        return roleHierarchy.getReachableGrantedAuthorities(userAccount.getAuthorities()).contains(new SimpleGrantedAuthority(UserRole.ROLE_MODERATOR.name()));
    }

    public boolean canLock(UserAccountDetailsDTO userAccount, LockDTO lockDTO) {
        if (userAccount==null){
            return false;
        }
        if (lockDTO!=null && userAccount.getId().equals(lockDTO.getUserId())){
            return false;
        }
        if (roleHierarchy.getReachableGrantedAuthorities(userAccount.getAuthorities()).contains(new SimpleGrantedAuthority(UserRole.ROLE_MODERATOR.name()))){
            return true;
        } else {
            return false;
        }
    }

    public boolean hasSettingsPermission(UserAccountDetailsDTO userAccount) {
        return Optional
                .ofNullable(userAccount)
                .map(u -> u.getAuthorities()
                        .contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name())))
                .orElse(false);
    }

    public boolean canDelete(UserAccountDetailsDTO userAccount, long userIdToDelete) {
        UserAccount deleted = userAccountRepository.findByUsername(Constants.DELETED).orElseThrow();
        if (deleted.getId().equals(userIdToDelete)){
            return false;
        }
        return Optional
                .ofNullable(userAccount)
                .map(u -> u.getAuthorities()
                        .contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name())) &&
                        !u.getId().equals(userIdToDelete))
                .orElse(false);
    }

    public boolean canSelfDelete(UserAccountDetailsDTO userAccount) {
        return Optional
                .ofNullable(userAccount).isPresent();
    }

    public boolean canChangeRole(UserAccountDetailsDTO currentUser, long userAccountId) {
        UserAccount userAccount = userAccountRepository.findById(userAccountId).orElseThrow();
        return lockAndDelete(currentUser, userAccount);
    }

    public boolean canLock(UserAccountDetailsDTO currentUser, UserAccount userAccount) {
        return lockAndDelete(currentUser, userAccount);
    }

    public boolean canDelete(UserAccountDetailsDTO currentUser, UserAccount userAccount) {
        return lockAndDelete(currentUser, userAccount);
    }

    public boolean canChangeRole(UserAccountDetailsDTO currentUser, UserAccount userAccount) {
        return lockAndDelete(currentUser, userAccount);
    }

    private boolean lockAndDelete(UserAccountDetailsDTO currentUser, UserAccount userAccount) {
        if (userAccount == null) {
            return false;
        }
        if (currentUser == null) {
            return  false;
        }
        UserAccount deleted = userAccountRepository.findByUsername(Constants.DELETED).orElseThrow();
        if (deleted.getId().equals(userAccount.getId())){
            return false;
        }
        if (userAccount.getId().equals(currentUser.getId())){
            return false;
        }
        return roleHierarchy.getReachableGrantedAuthorities(currentUser.getAuthorities()).contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name()));
    }
}
