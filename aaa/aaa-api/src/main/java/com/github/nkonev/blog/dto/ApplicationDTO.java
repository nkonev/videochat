package com.github.nkonev.blog.dto;

public class ApplicationDTO extends BaseApplicationDTO {
    private int id;

    public ApplicationDTO(int id, String title, String baseUrl, boolean enabled) {
        this.id = id;
        this.setTitle(title);
        this.setSrcUrl(baseUrl);
        this.setEnabled(enabled);
    }

    public ApplicationDTO() {
    }

    public int getId() {
        return id;
    }

    public void setId(int id) {
        this.id = id;
    }

}
