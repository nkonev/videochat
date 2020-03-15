package com.example.integration.test

import io.github.bonigarcia.wdm.WebDriverManager
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import okhttp3.OkHttpClient
import okhttp3.Request
import org.openqa.selenium.WebDriver
import org.openqa.selenium.WebElement
import org.openqa.selenium.chrome.ChromeDriver
import org.openqa.selenium.support.FindBy
import org.openqa.selenium.support.PageFactory
import org.openqa.selenium.support.ui.WebDriverWait
import org.slf4j.Logger
import org.slf4j.LoggerFactory
import org.slf4j.bridge.SLF4JBridgeHandler
import org.testng.Assert
import org.testng.annotations.*
import utils.FileUtils.getExistsFile
import utils.ProcessUtils.launch
import java.util.concurrent.TimeUnit

// https://medium.com/kotlin-lang-notes/selenium-kotlintest-4db1da9811cc
class ProfilePage(private val driver: WebDriver) {

    private val pageUrl = "http://site.local:8080/chat/profile"

    init {
        PageFactory.initElements(driver, this)
    }

    fun open() = driver.get(pageUrl)

    fun verifyUrl() {
        WebDriverWait(driver, 10).until { it.currentUrl == pageUrl }
    }

    fun verifyContent() {
        Assert.assertTrue(driver.pageSource.contains("Hello Nikita Konev", false))
    }
}

class LoginPage(private val driver: WebDriver) {

    init {
        PageFactory.initElements(driver, this)
    }

    @FindBy(css = "div#kc-form #kc-form-login input#username")
    lateinit var login: WebElement

    @FindBy(css = "div#kc-form #kc-form-login input#password")
    lateinit var password: WebElement

    @FindBy(css = "div#kc-form #kc-form-login input#kc-login")
    lateinit var loginButton: WebElement

    fun verifyUrl() {
        WebDriverWait(driver, 10).until { it.currentUrl.startsWith("http://auth.site.local:8080/auth") }
    }
}

class KeycloakTest {

    lateinit var driver: WebDriver

    var LOGGER: Logger = LoggerFactory.getLogger(KeycloakTest::class.java)

    lateinit var javaProcess: Process
    lateinit var golangProcess: Process

    lateinit var client: OkHttpClient


    @BeforeSuite
    fun beforeSuite() {
        SLF4JBridgeHandler.removeHandlersForRootLogger()
        SLF4JBridgeHandler.install()

        LOGGER.info("find jar")
        val jar = getExistsFile("../chat/target/chat-app-0.0.0-jar-with-dependencies.jar", "./chat/target/chat-app-0.0.0-jar-with-dependencies.jar")
        GlobalScope.launch {
            LOGGER.info("start jar")
            javaProcess = launch("java", "-jar", jar.canonicalPath)
        }

        LOGGER.info("find go binary")
        val bin = getExistsFile("../user-service/user-service", "./user-service/user-service")
        val cfg = getExistsFile("../user-service/config-dev/config.yml", "./user-service/config-dev/config.yml")
        GlobalScope.launch {
            LOGGER.info("start go binary")
            golangProcess = launch(bin.canonicalPath, "-config", cfg.canonicalPath)
        }

        client = OkHttpClient.Builder()
                .readTimeout(60, TimeUnit.SECONDS)
                .connectTimeout(60 / 2, TimeUnit.SECONDS)
                .writeTimeout(60, TimeUnit.SECONDS)
                .cache(null)
                .build()


        runBlocking {     // but this expression blocks the main thread
            var htmlResponded = false
            var num = 0
            val max = 100
            do {
                try {
                    val request = Request.Builder()
                            .url("http://127.0.0.1:10000/chat/index.html")
                            .build()

                    val response = client.newCall(request).execute()
                    val html = response.body()!!.string()
                    LOGGER.info("{}/{} Response: {}", num, max, html)
                    if (html.contains("Hello World!")) {
                        htmlResponded = true
                    } else {
                        TimeUnit.SECONDS.sleep(1)
                    }
                } catch (e: Exception) {
                    LOGGER.info("Exception: {}", e.message)
                    TimeUnit.SECONDS.sleep(1)
                }
            } while (!htmlResponded && num++ < max)
        }

        runBlocking {     // but this expression blocks the main thread
            var htmlResponded = false
            var num = 0
            val max = 100
            do {
                try {
                    val request = Request.Builder()
                            .url("http://127.0.0.1:1234")
                            .build()

                    val response = client.newCall(request).execute()
                    val html = response.body()!!.string()
                    LOGGER.info("{}/{} Response: {}", num, max, html)
                    if (html.contains("app-container")) {
                        htmlResponded = true
                    } else {
                        TimeUnit.SECONDS.sleep(1)
                    }
                } catch (e: Exception) {
                    LOGGER.info("Exception: {}", e.message)
                    TimeUnit.SECONDS.sleep(1)
                }
            } while (!htmlResponded && num++ < max)
        }

    }

    @BeforeMethod
    fun beforeMethod() {
        // https://sites.google.com/a/chromium.org/chromedriver/
        WebDriverManager.chromedriver().version("79.0.3945.36").setup();

        driver = ChromeDriver()

        driver.manage()?.timeouts()?.implicitlyWait(30, TimeUnit.SECONDS)
        driver.manage()?.window()?.maximize()
    }

    @Test(priority = 1)
    fun `Direct calling microservice requires authentication - positive`() {

        val profilePage = ProfilePage(driver)
        val loginPage = LoginPage(driver)

        profilePage.run {
            open()
        }

        loginPage.verifyUrl()

        loginPage.run {
            login.sendKeys("tester")
            password.sendKeys("tester")
            loginButton.click()
        }

        profilePage.verifyUrl()
        profilePage.verifyContent()
    }

    @Test(priority = 2)
    fun `Direct calling microservice requires authentication - negative`() {

        val profilePage = ProfilePage(driver)
        val loginPage = LoginPage(driver)


        profilePage.run {
            open()
        }

        loginPage.verifyUrl()

        loginPage.run {
            login.sendKeys("tester")
            password.sendKeys("tester2")
            loginButton.click()
        }

        loginPage.verifyUrl()
    }



    @AfterMethod
    fun afterMethod() {
        driver.close()

    }

    @AfterSuite
    fun afterSuite(){
        try {
            javaProcess.destroy()
        } catch (e: Exception) {
            LOGGER.error("Error during stop java process", e)
        }

        try {
            golangProcess.destroy()
        } catch (e: Exception) {
            LOGGER.error("Error during stop go process", e)
        }

    }

}
