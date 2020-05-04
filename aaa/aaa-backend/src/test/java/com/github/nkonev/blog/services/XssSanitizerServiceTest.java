package com.github.nkonev.blog.services;

import com.github.nkonev.blog.AbstractUtTestRunner;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.web.context.request.RequestContextHolder;
import org.springframework.web.context.request.ServletRequestAttributes;

public class XssSanitizerServiceTest extends AbstractUtTestRunner {

    @Autowired
    private XssSanitizerService xssSanitizerService;

    @BeforeEach
    public void beforeXssTest() throws Exception {
        ServletRequestAttributes servletRequestAttributes = new ServletRequestAttributes(new MockHttpServletRequest());
        RequestContextHolder.setRequestAttributes(servletRequestAttributes);
    }

    @AfterEach
    public void afterXssTest() throws Exception {
        RequestContextHolder.resetRequestAttributes();
    }


    @Test
    public void test() throws Exception {
        String unsafe = "<a href='javascript:alert('XSS')'>часто</a> используемый в печати и вэб-дизайне";

        String safe = xssSanitizerService.sanitize(unsafe);
        Assertions.assertEquals("часто используемый в печати и вэб-дизайне", safe);
    }

    @Test
    public void testNoClosed() throws Exception {
        String unsafe = "<a href='javascript:alert('XSS')'>часто используемый в печати и вэб-дизайне";

        String safe = xssSanitizerService.sanitize(unsafe);
        Assertions.assertEquals("часто используемый в печати и вэб-дизайне", safe);
    }

}
