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

val indexUrl = "http://127.0.0.1:1235/index.html"
val authUrl = "http://auth.site.local:8080/auth"
val chatUrl = "http://site.local:8080/chat"
// https://sites.google.com/a/chromium.org/chromedriver/
val chromeDriverVersion = "79.0.3945.36"
val webdriverImplicitWaitSeconds = 30L
val webdriverWaitTimeoutSeconds = 10L

// https://medium.com/kotlin-lang-notes/selenium-kotlintest-4db1da9811cc
class ProfilePage(private val driver: WebDriver) {

    private val pageUrl = chatUrl

    init {
        PageFactory.initElements(driver, this)
    }

    fun open() = driver.get(pageUrl)

    fun verifyUrl() {
        WebDriverWait(driver, webdriverWaitTimeoutSeconds).until { it.currentUrl == pageUrl }
    }

    fun verifyContent() {
        Assert.assertTrue(driver.pageSource.contains("Terry", false))
        Assert.assertTrue(driver.pageSource.contains("Perry", false))
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
        WebDriverWait(driver, webdriverWaitTimeoutSeconds).until { it.currentUrl.startsWith(authUrl) }
    }
}

class KeycloakTest {

    lateinit var driver: WebDriver

    var LOGGER: Logger = LoggerFactory.getLogger(KeycloakTest::class.java)

    lateinit var golangProcess: Process

    lateinit var client: OkHttpClient


    @BeforeSuite
    fun beforeSuite() {
        SLF4JBridgeHandler.removeHandlersForRootLogger()
        SLF4JBridgeHandler.install()

        LOGGER.info("find go binary")
        val bin = getExistsFile("../chat/videochat", "./chat/videochat")
        val cfg = getExistsFile("../chat/config-dev/config.yml", "./chat/config-dev/config.yml")
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


        runBlocking {     // this expression blocks the main thread
            var htmlResponded = false
            var num = 0
            val max = 100
            do {
                try {
                    val request = Request.Builder()
                            .url(indexUrl)
                            .build()

                    val response = client.newCall(request).execute()
                    val html = response.body()!!.string()
                    LOGGER.info("{}/{} Response: {}", num, max, html)
                    if (html.contains("Videochat")) {
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
        WebDriverManager.chromedriver().version(chromeDriverVersion).setup();

        driver = ChromeDriver()

        driver.manage()?.timeouts()?.implicitlyWait(webdriverImplicitWaitSeconds, TimeUnit.SECONDS)
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
            golangProcess.destroy()
        } catch (e: Exception) {
            LOGGER.error("Error during stop go process", e)
        }

    }

}
