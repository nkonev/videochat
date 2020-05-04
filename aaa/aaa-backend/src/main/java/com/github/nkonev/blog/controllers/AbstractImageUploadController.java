package com.github.nkonev.blog.controllers;

import com.github.nkonev.blog.config.CustomConfig;
import com.github.nkonev.blog.config.ImageConfig;
import com.github.nkonev.blog.exception.DataNotFoundException;
import com.github.nkonev.blog.exception.PayloadTooLargeException;
import com.github.nkonev.blog.exception.UnsupportedMessageTypeException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.util.Assert;
import org.springframework.web.util.UriComponentsBuilder;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import javax.sql.DataSource;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.sql.Connection;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.time.LocalDateTime;
import java.time.ZoneOffset;
import java.time.temporal.ChronoUnit;
import java.util.UUID;

public abstract class AbstractImageUploadController implements ImageOperations {

    @Autowired
    private DataSource dataSource;

    @Autowired
    protected JdbcTemplate jdbcTemplate;

    @Value("${custom.image.emulateCache:true}")
    private boolean emulateCache;

    @Value("${custom.image.chunkSize:4096}") // bytes
    private int chunkSize;

    private static final Logger LOGGER = LoggerFactory.getLogger(AbstractImageUploadController.class);

    @Autowired
    protected CustomConfig customConfig;

    @Autowired
    protected ImageConfig imageConfig;

    public static final String IMAGE_PART = "image";

    public static class ImageResponse {
        private String relativeUrl;
        private String url;

        public ImageResponse() { }

        public ImageResponse(String relativeUrl, String url) {
            this.relativeUrl = relativeUrl;
            this.url = url;
        }

        public String getRelativeUrl() {
            return relativeUrl;
        }

        public void setRelativeUrl(String relativeUrl) {
            this.relativeUrl = relativeUrl;
        }

        public String getUrl() {
            return url;
        }

        public void setUrl(String url) {
            this.url = url;
        }
    }

    protected ImageResponse postImage(
            String sql,
            String urlTemplate,
            long contentLength,
            String contentType,
            InputStream inputStream
	) throws SQLException {
		contentLength = getCorrectContentLength(contentLength);
        contentType = getCorrectContentType(contentType);

        try(Connection conn = dataSource.getConnection();) {
            try (PreparedStatement ps = conn.prepareStatement(sql)){
                ps.setString(2, contentType);
                ps.setBinaryStream(1, inputStream, (int) contentLength);
                try(ResultSet resp = ps.executeQuery()) {
                    if(!resp.next()) {
                        throw new RuntimeException("Expected result");
                    }
                    return getResponse(urlTemplate, resp.getObject("id", UUID.class), contentType);
                }

            } catch (SQLException e) {
                throw new RuntimeException(e);
            }

        }
    }

    private ImageResponse getResponse(String template, UUID uuid, String contentType) {
        String relativeUrl = UriComponentsBuilder.fromUriString(template)
                .buildAndExpand(uuid, getExtension(contentType))
                .toUriString();
        return new ImageResponse(relativeUrl, customConfig.getBaseUrl() + relativeUrl);
    }

    private String getCorrectContentType(String contentType) {
        MediaType inputMt = MediaType.valueOf(contentType);

        for(MediaType mediaType: imageConfig.getAllowedMimeTypes()){
            if (mediaType.isCompatibleWith(inputMt)) {
                return contentType;
            }
        }
        throw new UnsupportedMessageTypeException("Incompatible content type. Allowed: " + imageConfig.getAllowedMimeTypes());
    }

    private long getCorrectContentLength(long contentLength) {
        if (contentLength > imageConfig.getMaxBytes()) {
            throw new PayloadTooLargeException("Image must be <= "+ imageConfig.getMaxBytes() + " bytes");
        }
        return contentLength;
    }

    private String getExtension(String contentType) {
        Assert.notNull(contentType, "cannot be null");
        MediaType mt = MediaType.valueOf(contentType);
        return mt.getSubtype();
    }

    protected void getImage(String sql, UUID id, HttpServletRequest request, HttpServletResponse response, String imageType, String errorNotFoundMessage) {
        if(!shouldReturnLikeCache(id, request, response, imageType)){
            try(Connection conn = dataSource.getConnection();) {
                try (PreparedStatement ps = conn.prepareStatement(sql);) {
                    ps.setObject(1, id);
                    try (ResultSet rs = ps.executeQuery();) {
                        if (rs.next()) {
                            response.setContentType(rs.getString("content_type"));
                            response.setContentLength(rs.getInt("content_length"));
                            addCacheHeaders(id, "create_date_time", rs, response, imageType);
                            try (InputStream imgStream = rs.getBinaryStream("img");) {
                                copyStream(imgStream, response.getOutputStream());
                            } catch (SQLException | IOException e) {
                                throw new RuntimeException(e);
                            }
                        } else {
                            throw new DataNotFoundException(errorNotFoundMessage);
                        }
                    }
                }
            } catch (SQLException e) {
                throw new RuntimeException(e);
            }
        }
    };

    private void copyStream(InputStream from, OutputStream to) throws IOException {
		byte[] buffer = new byte[chunkSize];
		int len;
		while ((len = from.read(buffer)) != -1) {
			to.write(buffer, 0, len);
		}
	}

    private void addCacheHeaders(UUID id, String dateTimeColumnName, ResultSet resultSet, HttpServletResponse response, String imageType) throws SQLException {
        response.setHeader(HttpHeaders.CACHE_CONTROL, "public");
        response.addHeader(HttpHeaders.CACHE_CONTROL, "max-age="+imageConfig.getMaxAge());
        LocalDateTime ldt = resultSet.getObject(dateTimeColumnName, LocalDateTime.class);
        response.setDateHeader(HttpHeaders.LAST_MODIFIED, ldt.toEpochSecond(ZoneOffset.UTC)*1000);
        response.setDateHeader(HttpHeaders.EXPIRES, ldt.plus(imageConfig.getMaxAge(), ChronoUnit.SECONDS).toEpochSecond(ZoneOffset.UTC)*1000);
        response.setHeader(HttpHeaders.ETAG, convertToEtag(id, imageType));
    }

    private boolean shouldReturnLikeCache(UUID id, HttpServletRequest request, HttpServletResponse response, String imageType) {
        final String headerValue = request.getHeader(HttpHeaders.IF_NONE_MATCH);
        if (emulateCache && headerValue!=null && headerValue.equals(convertToEtag(id, imageType))) {
            response.setStatus(304);
            return true;
        } else {
            return false;
        }
    }

    private String convertToEtag(UUID id, String imageType){
        return imageType + "_" + id.toString();
    }

}
 