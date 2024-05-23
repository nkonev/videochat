import {deepCopy, findIndex, replaceInArray, replaceOrPrepend} from "@/utils.js";

export const firstPage = 1;
export const pageSize = 20;

export const dtoFactory = () => {return {items: [], count: 0} };

// requires extractDtoFromEventDto(), isCachedRelevantToArguments(), initializeWithArguments(),
// resetOnRouteIdChange(), initiateRequest(), initiateFilteredCountRequest(), initiateCountRequest(),
// clearOnClose(), clearOnReset(), shouldReactOnPageChange()

// optionally transformItems(), performMarking(), onInitialized(), afterFirstDrawItems()

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
                console.debug("Opening modal, data=", data);
                if (!this.isCachedRelevantToArguments(data)) {
                    this.reset();
                }

                this.initializeWithArguments(data);

                this.show = true;

                if (!this.dataLoaded) {
                    this.updateItems().then(()=>{
                        if (this.onInitialized) {
                            this.onInitialized()
                        }
                    })
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
                return this.initiateRequest()
                    .then((response) => {
                        const dto = deepCopy(response.data);
                        if (this.transformItems) {
                            this.transformItems(dto?.items);
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
                        if (this.afterFirstDrawItems){
                            this.afterFirstDrawItems()
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

            removeItems(dtos) {
                console.debug("Removing items", dtos);
                for (const dto of dtos) {
                    const idxToRemove = findIndex(this.itemsDto.items, dto);
                    if (idxToRemove !== -1) {
                        this.itemsDto.items.splice(idxToRemove, 1);
                    }
                }
            },
            replaceItems(dtos) {
                console.debug("Replacing items", dtos);
                for (const dto of dtos) {
                    replaceInArray(this.itemsDto.items, dto);
                }
            },
            addItems(dtos) {
                console.debug("Adding items", dtos);
                replaceOrPrepend(this.itemsDto.items, dtos)
            },

            onItemCreatedEvent(dto) {
                if (!this.dataLoaded) {
                    return
                }
                console.debug("onItemCreatedEvent", dto);

                // filter and load items count
                this.initiateCountRequest(dto).then((response) => {
                    this.itemsDto.count = response.data.count;
                }).then(()=> {
                    if (this.page == firstPage) {
                        this.initiateFilteredRequest(dto).then((response) => {
                            const extracted = this.extractDtoFromEventDto(dto);
                            const filteredItems = [];
                            extracted.forEach((item) => {
                                const foundIndex = findIndex(response.data, item);
                                if (foundIndex !== -1) {
                                    filteredItems.push(item);
                                }
                            })

                            const transformedItems = deepCopy(filteredItems);
                            if (this.transformItems) {
                                this.transformItems(transformedItems);
                            }

                            this.addItems(transformedItems);
                            // remove the last to fit to pageSize
                            if (this.itemsDto.items.length > pageSize) {
                                this.itemsDto.items.splice(this.itemsDto.items.length - 1, 1);
                            }

                            if (this.performMarking) {
                                this.$nextTick(() => {
                                    this.performMarking();
                                })
                            }
                        })
                    }
                })

            },
            onItemUpdatedEvent(dto) {
                if (!this.dataLoaded) {
                    return
                }
                console.debug("onItemUpdatedEvent", dto);
                const tmp = deepCopy(this.extractDtoFromEventDto(dto));
                if (this.transformItems) {
                    this.transformItems(tmp);
                }

                this.replaceItems(tmp);
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
                this.removeItems(this.extractDtoFromEventDto(dto));
                // load items count
                this.initiateCountRequest(dto).then((response) => {
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
                this.$nextTick(()=>{
                    this.closeModal(); // make show false to
                }).then(()=>{
                    this.reset(); // not to react in watch on page and not to load
                })
            },
        },
        watch: {
            show(newValue) {
                if (!newValue) {
                    this.closeModal();
                }
            },
            page(newValue) {
                if (this.shouldReactOnPageChange()) {
                    console.debug("Setting new page", newValue);
                    this.itemsDto = dtoFactory();
                    this.updateItems();
                }
            },
            // needed for case when we switch over chats and probably can occasionally receive wrong data over bus or not to unsubscribe and produce a leak
            '$route.params.id': function (newValue, oldValue) {
                if (newValue != oldValue && this.resetOnRouteIdChange()) {
                    this.reset();
                }
            }
        },
    }
}
