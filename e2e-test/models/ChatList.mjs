import {webUiChatsUrl, webUiUrl} from "../constants.mjs";
import {expect} from "@playwright/test";

// https://playwright.dev/docs/pom
export default class ChatList {
    constructor(page) {
        this.page = page;
    }
    async navigate() {
        await this.page.goto(webUiChatsUrl);
    }
    async openNewChatDialog() {
        const dialog = this.page.locator('#test-new-chat-dialog-button');
        await dialog.click();
        const form = this.page.locator('.v-dialog .v-form');
        await expect(form).toBeVisible();
    }

    async createAndSubmit(chatName, participants) {
        await this.page.fill('.v-dialog .v-form #test-chat-text', chatName);

        for (const participantName of participants) {
            console.log("Adding '" + participantName + "' to chat '" + chatName + "'");
            const autocomplete = this.page.locator('.v-autocomplete .v-field__input');
            await autocomplete.click();

            const autocompleteInput = this.page.locator('.v-autocomplete .v-field__input input');
            await autocompleteInput.fill(participantName);

            const selectableSuggestion = 0;
            const autocompleteSuggestedElements = this.page.locator('.v-autocomplete__content .v-list .v-list-item');
            await expect(autocompleteSuggestedElements.nth(selectableSuggestion)).toHaveText(participantName);
            await(autocompleteSuggestedElements.nth(selectableSuggestion).click());

            // close suggestion list
            await this.page.locator('.v-dialog .v-card-title').click()
        }

        const submit = this.page.locator('.v-dialog #test-chat-save-btn');
        await submit.click();
        const form = this.page.locator('.v-dialog .v-form');
        await expect(form).not.toBeVisible();
    }

    getRowsLocator() {
        return this.page.locator('.my-chat-scroller .v-list-item .v-list-item-title .chat-name');
    }

    async openChat(idx) {
        const chatElement = this.page.locator(`.my-chat-scroller>a>>nth=${idx}`);
        await chatElement.click();
    }

    async assertChatItemCount(expected) {
        return expect(this.getRowsLocator()).toHaveCount(expected);
    }

    async getChatName(index) {
        return (await this.getRowsLocator().nth(index).textContent()).trim()
    }
}
