export default () => {
  return {
    data(){
      return {
        showContextMenu: false,
        menuableItem: null,
        selection: null,
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
      onShowContextMenu(e, menuableItem) {
        this.showContextMenu = false;
        e.preventDefault();
        this.contextMenuX = e.clientX;
        this.contextMenuY = e.clientY;

        this.menuableItem = menuableItem;
        this.selection = this.getSelection();

        this.$nextTick(() => {
          this.showContextMenu = true;
        }).then(() => {
          this.setPosition()
        })
      },
      onCloseContextMenu() {
        this.showContextMenu = false;
        this.menuableItem = null;
        this.selection = null;
      },
    }
  }
}
