import {deepCopy, findIndex, replaceOrPrepend} from "@/utils.js";

export const firstPage = 1;
export const pageSize = 20;

export const dtoFactory = () => {return {items: [], count: 0} };

// requires extractDtoFromEventDto(), isCachedRelevantToArguments(), initializeWithArguments(),
// resetOnRouteIdChange(), initiateRequest(), initiateFilteredCountRequest(), initiateCountRequest(),
// clearOnClose(), clearOnReset()

// optionally transformItems(), performMarking()

export default () => {
    return {
        data() {
            return {
                show: false,
                itemsDto: dtoFactory(),
                loading: false,
                page: firstPage,
                dataLoaded: false,
            }
        },
        computed: {
            pagesCount() {
                const count = Math.ceil(this.itemsDto.count / pageSize);
                return count;
            },
            shouldShowPagination() {
                return this.itemsDto != null && this.itemsDto.items && this.itemsDto.count > pageSize
            },
        },
        methods: {
            showModal(data) {
                console.log("Opening modal, data=", data);
                if (!this.isCachedRelevantToArguments(data)) {
                    this.reset();
                }

                this.initializeWithArguments(data);

                this.show = true;

                if (!this.dataLoaded) {
                    this.updateItems();
                } else if (this.performMarking) {
                    this.performMarking();
                }
            },
            translatePage() {
                return this.page - 1;
            },
            // smart fetching
            updateItems(silent) {
                if (!silent) {
                    this.loading = true;
                }
                this.initiateRequest()
                    .then((response) => {
                        const dto = deepCopy(response.data);
                        if (this.transformItems) {
                            this.transformItems(dto);
                        }
                        this.itemsDto = dto;
                    })
                    .finally(() => {
                        if (!silent) {
                            this.loading = false;
                        }
                        this.dataLoaded = true;
                        if (this.performMarking) {
                            this.performMarking();
                        }
                    })
            },

            getTotalVisible() {
                if (!this.isMobile()) {
                    return 7
                } else if (this.page == firstPage || this.page == this.pagesCount) {
                    return 3
                } else {
                    return 1
                }
            },

            removeItem(dto) {
                console.debug("Removing item", dto);
                const idxToRemove = findIndex(this.itemsDto.items, dto);
                this.itemsDto.items.splice(idxToRemove, 1);
            },
            replaceItem(dto) {
                console.debug("Replacing item", dto);
                replaceOrPrepend(this.itemsDto.items, [dto]);
            },
            addItem(dto) {
                console.debug("Adding item", dto);
                if (this.transformItem) {
                    this.transformItem(dto);
                }
                this.itemsDto.items.unshift(dto);
            },

            onItemCreatedEvent(dto) {
                if (!this.dataLoaded) {
                    return
                }
                console.debug("onItemCreatedEvent", dto);

                if (this.page == firstPage) {
                    // filter and load items count
                    this.initiateFilteredCountRequest(this.extractDtoFromEventDto(dto)).then((response) => {
                        this.itemsDto.count = response.data.count;
                        if (response.data.found) {
                            this.addItem(this.extractDtoFromEventDto(dto));
                            // remove the last to fit to pageSize
                            if (this.itemsDto.items.length > pageSize) {
                                this.itemsDto.items.splice(this.itemsDto.items.length - 1, 1);
                            }

                            if (this.performMarking) {
                                this.$nextTick(() => {
                                    this.performMarking();
                                })
                            }
                        }
                    })
                }
            },
            onItemUpdatedEvent(dto) {
                if (!this.dataLoaded) {
                    return
                }
                console.debug("onItemUpdatedEvent", dto);
                this.replaceItem(this.extractDtoFromEventDto(dto));
                if (this.performMarking) {
                    this.$nextTick(() => {
                        this.performMarking();
                    })
                }
            },
            onItemRemovedEvent(dto) {
                if (!this.dataLoaded) {
                    return
                }
                console.debug("onItemRemovedEvent", dto);
                this.removeItem(this.extractDtoFromEventDto(dto));
                // load items count
                this.initiateCountRequest().then((response) => {
                        this.itemsDto.count = response.data.count;
                    }).then(() => {
                    if (this.page > this.pagesCount) { // fix case when we stay on the last page but there is lesser pages on the server
                        this.page = this.pagesCount; // this causes update() because of watch
                        return
                    }

                    const notEnoughItemsOnPage = this.itemsDto.count > pageSize && this.itemsDto.items.length < pageSize;
                    const nonLastPage = this.page != this.pagesCount;
                    if (notEnoughItemsOnPage && nonLastPage) {
                        this.updateItems(true);
                    }
                })
            },


            closeModal() {
                this.show = false;
                this.clearOnClose();
            },
            reset() {
                this.page = firstPage;
                this.itemsDto = dtoFactory();
                this.dataLoaded = false;
                this.clearOnReset();
                this.clearOnClose();
            },
            onLogout() {
                this.reset();
                this.closeModal();
            },
        },
        watch: {
            show(newValue) {
                if (!newValue) {
                    this.closeModal();
                }
            },
            page(newValue) {
                if (this.show) {
                    console.debug("SettingNewPage", newValue);
                    this.itemsDto = dtoFactory();
                    this.updateItems();
                }
            },
            '$route.params.id': function (newValue, oldValue) {
                if (newValue != oldValue && this.resetOnRouteIdChange()) {
                    this.reset();
                }
            }
        },
    }
}
