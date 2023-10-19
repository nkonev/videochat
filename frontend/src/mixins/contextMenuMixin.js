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
        }
      },
      onShowContextMenuBase(e, menuableItem) {
        this.showContextMenu = false;
        e.preventDefault();
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
    }
  }
}
