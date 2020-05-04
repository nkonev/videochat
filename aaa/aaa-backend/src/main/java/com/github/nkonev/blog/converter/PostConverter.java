package com.github.nkonev.blog.converter;

import com.github.nkonev.blog.controllers.ImagePostTitleUploadController;
import com.github.nkonev.blog.dto.PostDTO;
import com.github.nkonev.blog.dto.PostDTOWithAuthorization;
import com.github.nkonev.blog.dto.UserAccountDetailsDTO;
import com.github.nkonev.blog.entity.elasticsearch.IndexPost;
import com.github.nkonev.blog.entity.jdbc.Post;
import com.github.nkonev.blog.exception.BadRequestException;
import com.github.nkonev.blog.security.BlogSecurityService;
import com.github.nkonev.blog.security.permissions.PostPermissions;
import com.github.nkonev.blog.utils.ImageDownloader;
import org.jsoup.Jsoup;
import org.jsoup.nodes.Document;
import org.jsoup.nodes.Element;
import org.jsoup.select.Elements;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.util.Assert;
import org.springframework.util.StringUtils;
import org.springframework.web.util.UriComponents;
import org.springframework.web.util.UriComponentsBuilder;

import java.util.List;

@Service
public class PostConverter {
    @Autowired
    private BlogSecurityService blogSecurityService;

    @Autowired
    private UserAccountConverter userAccountConverter;

    @Value("${custom.set.first.image.as.title:true}")
    boolean setFirstImageAsTitle;

    @Autowired
    private ImageDownloader imageDownloader;

    @Autowired
    private ImagePostTitleUploadController imagePostTitleUploadController;

    private static final Logger LOGGER = LoggerFactory.getLogger(PostConverter.class);

    public PostDTOWithAuthorization convertToDto(Post saved, UserAccountDetailsDTO userAccount) {
        Assert.notNull(saved, "Post can't be null");

        return new PostDTOWithAuthorization(
                saved.getId(),
                saved.getTitle(),
                (saved.getText()),
                saved.getTitleImg(),
                userAccountConverter.convertToOwnerDTO(saved.getOwnerId()),
                blogSecurityService.hasPostPermission(saved, userAccount, PostPermissions.EDIT),
                blogSecurityService.hasPostPermission(saved, userAccount, PostPermissions.DELETE),
                saved.getCreateDateTime(),
                saved.getEditDateTime(),
                saved.isDraft()
        );
    }

    private void checkLength(String comment) {
        final int MIN_POST_LENGTH = 1;
        String trimmed = StringUtils.trimWhitespace(comment);
        if (trimmed == null || trimmed.length() < MIN_POST_LENGTH) {
            throw new BadRequestException("post is too short, must be longer than " + MIN_POST_LENGTH);
        }
    }

    public Post convertToPost(PostDTO postDTO, Post forUpdate) {
        Assert.notNull(postDTO, "postDTO can't be null");

        if (forUpdate == null){ forUpdate = new Post(); }
        String sanitizedHtml = postDTO.getText();
        checkLength(sanitizedHtml);
        forUpdate.setText(sanitizedHtml);
        forUpdate.setTitle(cleanHtmlTags(postDTO.getTitle()));
        if (Boolean.TRUE.equals(postDTO.getRemoveTitleImage())) {
            forUpdate.setTitleImg(null);
        } else {
            String titleImg = getTitleImg(postDTO);
            forUpdate.setTitleImg(titleImg);
        }
        forUpdate.setDraft(postDTO.isDraft());
        return forUpdate;
    }

    private String getTitleImg(PostDTO postDTO) {
        String titleImg = postDTO.getTitleImg();
        if (setFirstImageAsTitle && StringUtils.isEmpty(titleImg)) {
            try {
                Document document = Jsoup.parse(postDTO.getText());

                // try to get title url from post content's images
                Elements images = document.getElementsByTag("img");
                if (!images.isEmpty()) {
                    Element element = images.get(0);
                    return element.attr("src");
                }

                // try to get title url from post content's videos
                Elements iframes = document.getElementsByTag("iframe");
                if (!iframes.isEmpty()) {
                    Element element = iframes.get(0);
                    String iframeSrcUrl = element.attr("src");
                    if (iframeSrcUrl.contains("youtube.com")) {
                        String youtubeVideoId = getYouTubeVideoId(iframeSrcUrl);
                        String youtubeThumbnailUrl = "https://i.ytimg.com/vi/"+youtubeVideoId+"/maxresdefault.jpg";
                        return imageDownloader.downloadImageAndSave(youtubeThumbnailUrl, imagePostTitleUploadController);
                    }
                }

            } catch (RuntimeException e) {
                if (LOGGER.isDebugEnabled()){
                    LOGGER.warn("Error during parse image from content: {}", e.getMessage(), e);
                } else {
                    LOGGER.warn("Error during parse image from content: {}", e.getMessage());
                }
                return null;
            }
        }
        return titleImg;
    }

    public static String getYouTubeVideoId(String iframeSrcUrl) {
        UriComponents build = UriComponentsBuilder.fromHttpUrl(iframeSrcUrl).build();
        List<String> pathSegments = build.getPathSegments();
        return pathSegments.get(pathSegments.size() - 1);
    }

    public IndexPost toElasticsearchPost(com.github.nkonev.blog.entity.jdbc.Post jpaPost) {
        String sanitizedHtml = jpaPost.getText();
        return new IndexPost(jpaPost.getId(), jpaPost.getTitle(), cleanHtmlTags(sanitizedHtml), jpaPost.isDraft(), jpaPost.getOwnerId());
    }

    public static void cleanTags(PostDTO postDTO) {
        postDTO.setText(cleanHtmlTags(postDTO.getText()));
    }

    /**
     * Used in main page
     * @param html
     * @return
     */
    public static String cleanHtmlTags(String html) {
        return html == null ? null : Jsoup.parse(html).text();
    }

    /**
     * Used in main page
     */
    public PostDTO convertToPostDTO(Post post) {
        if (post==null) {return null;}

        return new PostDTO(
                post.getId(),
                post.getTitle(),
                post.getText(),
                post.getTitleImg(),
                post.getCreateDateTime(),
                post.getEditDateTime(),
                userAccountConverter.convertToOwnerDTO(post.getOwnerId()),
                post.isDraft()
        );
    }

}
