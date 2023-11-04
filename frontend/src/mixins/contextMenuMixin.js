export default () => {
  return {
    data(){
      return {
        showContextMenu: false,
        menuableItem: null,
        contextMenuX: 0,
        contextMenuY: 0,
      }
    },
    methods: {
      setPosition() {
        const element = document.querySelector("." + this.className() + " .v-overlay__content");
        if (element) {
          element.style.position = "absolute";
          element.style.top = this.contextMenuY + "px";
          element.style.left = this.contextMenuX + "px";

          const bottom = Number(getComputedStyle(element).bottom.replace("px", ''));
          if (bottom < 0) {
            const newTop = this.contextMenuY + bottom - 8;
            element.style.top = newTop + "px";
          }

          const width = Number(getComputedStyle(element).width.replace("px", ''));
          if (width < 260) {
              const delta = Math.abs(260 - width);
              const newLeft = this.contextMenuX - delta - 8;
              element.style.left = newLeft + "px";
          }
        }
      },
      onShowContextMenuBase(e, menuableItem) {
        e.preventDefault();
        if (this.showContextMenu) {
          return
        }
        this.contextMenuX = e.clientX;
        this.contextMenuY = e.clientY;

        this.menuableItem = menuableItem;

        this.$nextTick(() => {
          this.showContextMenu = true;
        }).then(() => {
          this.setPosition()
        })
      },
      onCloseContextMenuBase() {
        this.showContextMenu = false;
        this.menuableItem = null;
      },
      onUpdate(v) {
        if (!v) {
            this.onCloseContextMenu();
        }
      },
    }
  }
}
