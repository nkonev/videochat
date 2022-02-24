import {webUiUrl} from "../constants.mjs";
import {expect} from "@playwright/test";

// https://playwright.dev/docs/pom
export default class ChatList {
    constructor(page, ) {
        this.page = page;
    }
    async navigate() {
        await this.page.goto(webUiUrl);
    }
    async openNewChatDialog() {
        const dialog = this.page.locator('#new-chat-dialog-button');
        await dialog.click();
        const form = this.page.locator('.v-dialog .v-form');
        await expect(form).toBeVisible();
    }

    async createAndSubmit(chatName, participants) {
        await this.page.fill('.v-dialog .v-form #new-chat-text', chatName);

        for (const participantName of participants) {
            console.log("Adding '" + participantName + "' to chat '" + chatName + "'");
            const autocomplete = this.page.locator('.v-autocomplete .v-select__selections');
            await autocomplete.click();

            const autocompleteInput = this.page.locator('.v-autocomplete .v-select__selections input');
            await autocompleteInput.fill(participantName);

            const selectableSuggestion = 0;
            const autocompleteSuggestedElements = this.page.locator('.v-autocomplete__content .v-select-list .v-list-item__content');
            await expect(autocompleteSuggestedElements.nth(selectableSuggestion)).toHaveText(participantName);
            await(autocompleteSuggestedElements.nth(selectableSuggestion).click());

            // close suggestion list
            await this.page.locator('.v-dialog .v-card__title').click()
        }

        const submit = this.page.locator('.v-dialog #chat-save-btn');
        await submit.click();
        const form = this.page.locator('.v-dialog .v-form');
        await expect(form).not.toBeVisible();
    }

    getRowsLocator() {
        return this.page.locator('#chat-list-items .v-list-item .v-list-item__title');
    }

    async getChatItemCount() {
        return await this.getRowsLocator().count();
    }

    async getChatName(index) {
        return (await this.getRowsLocator().nth(index).textContent()).trim()
    }
}
