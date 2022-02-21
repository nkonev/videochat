const { test, expect } = require('@playwright/test');

const defaultAdminUser = {
    user: "admin",
    password: "admin"
}

test('login successful', async ({ page }) => {
    await page.goto('http://localhost:8081/');
    const submit = page.locator('.v-dialog .v-form #button-login');
    await page.fill('.v-dialog .v-form #text-login', defaultAdminUser.user);
    await page.fill('.v-dialog .v-form #text-password', defaultAdminUser.password);
    await expect(submit).toBeVisible();
    await submit.click();

    await expect(page.locator('#chat-list-items')).toBeVisible();
    const count = await page.locator('#chat-list-items .v-list-item').count();
    expect(count).toBeGreaterThanOrEqual(1);
});
